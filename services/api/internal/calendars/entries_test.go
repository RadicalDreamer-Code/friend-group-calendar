package calendars

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseEntryInput(t *testing.T) {
	if got, err := NormalizeTitle("  Board games 🎲 "); err != nil || got != "Board games 🎲" {
		t.Errorf("NormalizeTitle = %q, %v", got, err)
	}
	for _, in := range []string{"", "  ", strings.Repeat("a", MaxTitleLength+1), "a\nb"} {
		if _, err := NormalizeTitle(in); !errors.Is(err, ErrInvalidTitle) {
			t.Errorf("NormalizeTitle(%q) should fail", in)
		}
	}

	if d, err := ParseDate("2026-02-28"); err != nil || FormatDate(d) != "2026-02-28" || d.Location() != time.UTC {
		t.Errorf("ParseDate = %v, %v", d, err)
	}
	for _, in := range []string{"", "2026-2-28", "2026-02-30", "1999-12-31", "2100-01-01", "2026-02-28T00:00:00Z"} {
		if _, err := ParseDate(in); !errors.Is(err, ErrInvalidDate) {
			t.Errorf("ParseDate(%q) should fail", in)
		}
	}

	if m, err := ParseMonth("2099-12"); err != nil || FormatDate(m) != "2099-12-01" {
		t.Errorf("ParseMonth = %v, %v", m, err)
	}
	for _, in := range []string{"", "2026-13", "2026-1", "1999-12", "2100-01", "2026-10-01"} {
		if _, err := ParseMonth(in); !errors.Is(err, ErrInvalidMonth) {
			t.Errorf("ParseMonth(%q) should fail", in)
		}
	}

	day, _ := ParseDate("2026-10-30")
	for _, tt := range []struct {
		end  string
		want error
	}{
		{"2026-10-30", nil},
		{"2026-11-29", nil}, // 31 days, both ends included
		{"2026-11-30", ErrInvalidEndDate},
		{"2026-10-29", ErrInvalidEndDate},
	} {
		end, _ := ParseDate(tt.end)
		if err := checkSpan(day, end); !errors.Is(err, tt.want) {
			t.Errorf("checkSpan(2026-10-30, %s) = %v, want %v", tt.end, err, tt.want)
		}
	}
	if err := checkSpan(LastDay, LastDay.AddDate(0, 0, 1)); !errors.Is(err, ErrInvalidDate) {
		t.Errorf("checkSpan past 2099 = %v", err)
	}

	if _, err := ParseReply("maybe"); !errors.Is(err, ErrInvalidReply) {
		t.Errorf("ParseReply(maybe) = %v", err)
	}
	if _, err := parseColor("red"); !errors.Is(err, ErrInvalidColor) {
		t.Errorf("parseColor(red) = %v", err)
	}
	if c, err := parseCover(""); c != nil || err != nil {
		t.Errorf("parseCover() = %v, %v", c, err)
	}
	if _, err := parseCover("forest"); !errors.Is(err, ErrInvalidCover) {
		t.Errorf("parseCover(forest) = %v", err)
	}
}

// group is a calendar owned by ada with bob as editor and carla as viewer.
type group struct {
	cal             Calendar
	ada, bob, carla uuid.UUID
}

func newGroup(t *testing.T, f *fixture) group {
	t.Helper()
	g := group{ada: f.user(t, "ada"), bob: f.user(t, "bob"), carla: f.user(t, "carla")}
	f.befriend(t, g.ada, g.bob)
	f.befriend(t, g.ada, g.carla)
	g.cal = f.create(t, g.ada, "Trip")
	f.invite(t, g.cal, g.ada, g.bob, RoleEditor)
	g.cal = f.invite(t, g.cal, g.ada, g.carla, RoleViewer)
	return g
}

func (f *fixture) entry(t *testing.T, g group, by uuid.UUID, title, date string) Entry {
	t.Helper()
	e, err := f.svc.CreateEntry(context.Background(), g.cal.ID, by, EntryInput{Title: title, Date: date, Color: "sage"})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	return e
}

func TestEntriesCreateListUpdateDelete(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	e := f.entry(t, g, g.bob, "  Ski day ", "2026-10-31")
	if e.Title != "Ski day" || FormatDate(e.Day) != "2026-10-31" || e.Color != "sage" ||
		e.CreatedBy == nil || e.CreatedBy.Username != "bob" || len(e.Accepted) != 0 || len(e.Declined) != 0 {
		t.Fatalf("created = %+v", e)
	}
	f.entry(t, g, g.ada, "Dinner", "2026-10-01")
	f.entry(t, g, g.ada, "Second dinner", "2026-10-01")
	f.entry(t, g, g.ada, "November", "2026-11-01")

	// Viewers see entries but cannot change them.
	if _, err := f.svc.CreateEntry(ctx, g.cal.ID, g.carla, EntryInput{Title: "Mine", Date: "2026-10-02", Color: "sage"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer creating: %v", err)
	}
	list, err := f.svc.ListEntries(ctx, g.cal.ID, g.carla, "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, e := range list {
		titles = append(titles, e.Title)
	}
	if !slices.Equal(titles, []string{"Dinner", "Second dinner", "Ski day"}) {
		t.Fatalf("October = %v", titles)
	}

	title, color := "Ski weekend", "fjord"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.carla, e.ID, EntryChanges{Title: &title}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer editing: %v", err)
	}
	e, err = f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, e.ID, EntryChanges{Title: &title, Color: &color})
	if err != nil || e.Title != title || e.Color != color || FormatDate(e.Day) != "2026-10-31" {
		t.Fatalf("updated = %+v, %v", e, err)
	}
	bad := "red"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, e.ID, EntryChanges{Color: &bad}); !errors.Is(err, ErrInvalidColor) {
		t.Fatalf("bad color: %v", err)
	}

	if err := f.svc.DeleteEntry(ctx, g.cal.ID, g.carla, e.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer deleting: %v", err)
	}
	if err := f.svc.DeleteEntry(ctx, g.cal.ID, g.bob, e.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteEntry(ctx, g.cal.ID, g.bob, e.ID); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("deleting twice: %v", err)
	}
	if _, err := f.svc.GetEntry(ctx, g.cal.ID, g.bob, e.ID); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("deleted entry: %v", err)
	}
}

func TestNextEntry(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	if _, ok, err := f.svc.NextEntry(ctx, g.cal.ID, g.carla, "2026-10-24"); err != nil || ok {
		t.Fatalf("empty calendar: ok %v, %v", ok, err)
	}
	f.entry(t, g, g.ada, "Past", "2026-10-23")
	first := f.entry(t, g, g.ada, "First", "2026-10-31")
	f.entry(t, g, g.bob, "Same day, later", "2026-10-31")
	f.entry(t, g, g.ada, "November", "2026-11-02")
	if _, err := f.svc.Respond(ctx, g.cal.ID, g.carla, first.ID, ReplyAccepted); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct{ from, want string }{
		{"2026-10-24", "First"}, // skips past entries, takes the oldest of the day
		{"2026-10-23", "Past"},  // includes the day itself
		{"2026-11-01", "November"},
	} {
		e, ok, err := f.svc.NextEntry(ctx, g.cal.ID, g.carla, tt.from)
		if err != nil || !ok || e.Title != tt.want {
			t.Fatalf("from %s = %q, %v, %v; want %q", tt.from, e.Title, ok, err, tt.want)
		}
	}
	e, _, _ := f.svc.NextEntry(ctx, g.cal.ID, g.carla, "2026-10-24")
	if !slices.Equal(e.Accepted, []uuid.UUID{g.carla}) || e.CreatedBy == nil {
		t.Fatalf("answers and creator = %+v", e)
	}
	if _, ok, err := f.svc.NextEntry(ctx, g.cal.ID, g.carla, "2026-11-03"); err != nil || ok {
		t.Fatalf("after the last entry: ok %v, %v", ok, err)
	}

	if _, _, err := f.svc.NextEntry(ctx, g.cal.ID, g.carla, "2026-10"); !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("bad day: %v", err)
	}
	outsider := f.user(t, "dora")
	if _, _, err := f.svc.NextEntry(ctx, g.cal.ID, outsider, "2026-10-24"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider: %v", err)
	}
}

func (f *fixture) trip(t *testing.T, g group, title, date, endDate string) Entry {
	t.Helper()
	e, err := f.svc.CreateEntry(context.Background(), g.cal.ID, g.ada, EntryInput{Title: title, Date: date, EndDate: endDate, Color: "moss"})
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}
	return e
}

func titles(entries []Entry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, e.Title)
	}
	return out
}

func TestMultiDayEntriesOverlapMonths(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	single := f.entry(t, g, g.ada, "Dinner", "2026-10-30")
	if !single.EndDay.Equal(single.Day) {
		t.Fatalf("single-day entry ends %v", single.EndDay)
	}
	trip := f.trip(t, g, "Cabin trip", "2026-10-30", "2026-11-02")
	if FormatDate(trip.Day) != "2026-10-30" || FormatDate(trip.EndDay) != "2026-11-02" {
		t.Fatalf("trip = %+v", trip)
	}
	f.trip(t, g, "Long weekend", "2026-09-27", "2026-10-01")
	f.trip(t, g, "Whole November", "2026-11-01", "2026-11-30")
	f.entry(t, g, g.ada, "September", "2026-09-30")
	f.entry(t, g, g.ada, "December", "2026-12-01")

	for _, tt := range []struct {
		month string
		want  []string
	}{
		{"2026-09", []string{"Long weekend", "September"}},
		{"2026-10", []string{"Long weekend", "Dinner", "Cabin trip"}},
		{"2026-11", []string{"Cabin trip", "Whole November"}},
		{"2026-12", []string{"December"}},
	} {
		list, err := f.svc.ListEntries(ctx, g.cal.ID, g.carla, tt.month)
		if err != nil || !slices.Equal(titles(list), tt.want) {
			t.Fatalf("%s = %v, %v; want %v", tt.month, titles(list), err, tt.want)
		}
	}
}

func TestListEntriesRange(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	f.trip(t, g, "Cabin trip", "2026-10-30", "2026-11-02")
	f.entry(t, g, g.ada, "Dinner", "2026-11-05")
	f.entry(t, g, g.ada, "Later", "2026-11-06")

	for _, tt := range []struct {
		from, to string
		want     []string
	}{
		{"2026-11-01", "2026-11-05", []string{"Cabin trip", "Dinner"}}, // the trip started before the range
		{"2026-10-01", "2026-10-30", []string{"Cabin trip"}},           // and here it ends after it
		{"2026-11-03", "2026-11-04", []string{}},
		{"2026-11-06", "2026-11-06", []string{"Later"}},
	} {
		list, err := f.svc.ListEntriesRange(ctx, g.cal.ID, g.carla, tt.from, tt.to)
		if err != nil || !slices.Equal(titles(list), tt.want) {
			t.Fatalf("%s to %s = %v, %v; want %v", tt.from, tt.to, titles(list), err, tt.want)
		}
	}

	if _, err := f.svc.ListEntriesRange(ctx, g.cal.ID, g.carla, "2026-01-01", "2026-04-02"); err != nil {
		t.Fatalf("92 days: %v", err)
	}
	for _, tt := range []struct {
		from, to string
		want     error
	}{
		{"2026-01-01", "2026-04-03", ErrInvalidRange}, // 93 days
		{"2026-11-02", "2026-11-01", ErrInvalidRange},
		{"2026-11-31", "2026-12-01", ErrInvalidDate},
		{"2026-11-01", "", ErrInvalidDate},
	} {
		if _, err := f.svc.ListEntriesRange(ctx, g.cal.ID, g.carla, tt.from, tt.to); !errors.Is(err, tt.want) {
			t.Errorf("%s to %s: err = %v, want %v", tt.from, tt.to, err, tt.want)
		}
	}
	stranger := f.user(t, "dora")
	if _, err := f.svc.ListEntriesRange(ctx, g.cal.ID, stranger, "2026-11-01", "2026-11-05"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("someone outside the calendar: err = %v, want ErrNotFound", err)
	}
}

func TestNextEntryStillRunning(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	f.trip(t, g, "Cabin trip", "2026-10-30", "2026-11-02")
	f.entry(t, g, g.ada, "Dinner", "2026-11-01")

	for _, tt := range []struct{ from, want string }{
		{"2026-10-29", "Cabin trip"},
		{"2026-11-01", "Cabin trip"}, // still running, and it started first
		{"2026-11-02", "Cabin trip"}, // its last day
	} {
		e, ok, err := f.svc.NextEntry(ctx, g.cal.ID, g.carla, tt.from)
		if err != nil || !ok || e.Title != tt.want {
			t.Fatalf("from %s = %q, %v, %v; want %q", tt.from, e.Title, ok, err, tt.want)
		}
	}
	if _, ok, err := f.svc.NextEntry(ctx, g.cal.ID, g.carla, "2026-11-03"); err != nil || ok {
		t.Fatalf("after the trip: ok %v, %v", ok, err)
	}
}

func TestMultiDayEntryValidation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	for _, tt := range []struct {
		end  string
		want error
	}{
		{"2026-10-29", ErrInvalidEndDate},
		{"2026-11-30", ErrInvalidEndDate},
		{"2026-11-31", ErrInvalidDate},
	} {
		in := EntryInput{Title: "Trip", Date: "2026-10-30", EndDate: tt.end, Color: "sage"}
		if _, err := f.svc.CreateEntry(ctx, g.cal.ID, g.ada, in); !errors.Is(err, tt.want) {
			t.Fatalf("end %s: %v, want %v", tt.end, err, tt.want)
		}
	}
	trip := f.trip(t, g, "Trip", "2026-10-30", "2026-11-29") // the longest allowed

	// Moving the start alone keeps the length; that may not run past 2099.
	late := "2099-12-30"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, EntryChanges{Date: &late}); !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("moving past 2099: %v", err)
	}
	early, tooLate := "2026-10-29", "2026-11-30"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, EntryChanges{Date: &early}); err != nil {
		t.Fatalf("moving a day earlier: %v", err)
	}
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, EntryChanges{EndDate: &tooLate}); !errors.Is(err, ErrInvalidEndDate) {
		t.Fatalf("making it longer: %v", err)
	}
	start, end := "2026-11-05", "2026-11-03"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, EntryChanges{Date: &start, EndDate: &end}); !errors.Is(err, ErrInvalidEndDate) {
		t.Fatalf("ending before the start: %v", err)
	}

	// The database holds the same rules.
	if _, err := f.db.Exec(ctx, "UPDATE entries SET end_day = day - 1 WHERE id = $1", trip.ID); err == nil {
		t.Fatal("end before start was stored")
	}
	if _, err := f.db.Exec(ctx, "UPDATE entries SET end_day = day + 31 WHERE id = $1", trip.ID); err == nil {
		t.Fatal("32-day entry was stored")
	}
}

func TestEntryCovers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	if e := f.entry(t, g, g.ada, "Dinner", "2026-10-01"); e.Cover != "" {
		t.Fatalf("no cover = %q", e.Cover)
	}
	for _, cover := range Covers {
		in := EntryInput{Title: "x", Date: "2026-10-02", Color: "sage", Cover: cover}
		if e, err := f.svc.CreateEntry(ctx, g.cal.ID, g.ada, in); err != nil || e.Cover != cover {
			t.Fatalf("cover %s = %+v, %v", cover, e, err)
		}
	}
	for _, bad := range []string{"forest", "Lake", "night out"} {
		in := EntryInput{Title: "x", Date: "2026-10-03", Color: "sage", Cover: bad}
		if _, err := f.svc.CreateEntry(ctx, g.cal.ID, g.ada, in); !errors.Is(err, ErrInvalidCover) {
			t.Fatalf("cover %q: %v", bad, err)
		}
	}

	in := EntryInput{Title: "Swim", Date: "2026-10-04", Color: "sage", Cover: "lake"}
	e, err := f.svc.CreateEntry(ctx, g.cal.ID, g.ada, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Respond(ctx, g.cal.ID, g.carla, e.ID, ReplyAccepted); err != nil {
		t.Fatal(err)
	}
	if list, err := f.svc.ListEntries(ctx, g.cal.ID, g.carla, "2026-10"); err != nil || list[len(list)-1].Cover != "lake" {
		t.Fatalf("listed = %+v, %v", list, err)
	}

	// Changing other fields keeps the cover, and a new cover keeps the answers.
	title := "Swim and sauna"
	if e, err = f.svc.UpdateEntry(ctx, g.cal.ID, g.bob, e.ID, EntryChanges{Title: &title}); err != nil || e.Cover != "lake" {
		t.Fatalf("after renaming = %+v, %v", e, err)
	}
	cover := "beach"
	if e, err = f.svc.UpdateEntry(ctx, g.cal.ID, g.bob, e.ID, EntryChanges{Cover: &cover}); err != nil || e.Cover != "beach" ||
		!slices.Equal(e.Accepted, []uuid.UUID{g.carla}) {
		t.Fatalf("after new cover = %+v, %v", e, err)
	}
	bad := "forest"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.bob, e.ID, EntryChanges{Cover: &bad}); !errors.Is(err, ErrInvalidCover) {
		t.Fatalf("bad cover: %v", err)
	}
	none := ""
	if e, err = f.svc.UpdateEntry(ctx, g.cal.ID, g.bob, e.ID, EntryChanges{Cover: &none}); err != nil || e.Cover != "" {
		t.Fatalf("after removing = %+v, %v", e, err)
	}
	var stored *string
	if err := f.db.QueryRow(ctx, "SELECT cover FROM entries WHERE id = $1", e.ID).Scan(&stored); err != nil || stored != nil {
		t.Fatalf("stored cover = %v, %v", stored, err)
	}

	// The database holds the same list.
	if _, err := f.db.Exec(ctx, "UPDATE entries SET cover = 'forest' WHERE id = $1", e.ID); err == nil {
		t.Fatal("unknown cover was stored")
	}
}

func TestMultiDayEntryAnswers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	trip := f.trip(t, g, "Cabin trip", "2026-10-30", "2026-11-02")

	accept := func() {
		t.Helper()
		if _, err := f.svc.Respond(ctx, g.cal.ID, g.carla, trip.ID, ReplyAccepted); err != nil {
			t.Fatal(err)
		}
	}
	update := func(c EntryChanges) Entry {
		t.Helper()
		e, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, c)
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		return e
	}

	accept()
	same, sameEnd := "2026-10-30", "2026-11-02"
	if e := update(EntryChanges{Date: &same, EndDate: &sameEnd}); len(e.Accepted) != 1 {
		t.Fatalf("same dates cleared the answers: %+v", e)
	}
	end := "2026-11-03"
	if e := update(EntryChanges{EndDate: &end}); len(e.Accepted) != 0 || FormatDate(e.EndDay) != end {
		t.Fatalf("after a new end date = %+v", e)
	}
	accept()
	start := "2026-11-06"
	e := update(EntryChanges{Date: &start})
	if len(e.Accepted) != 0 || FormatDate(e.Day) != start || FormatDate(e.EndDay) != "2026-11-10" {
		t.Fatalf("after a new start date = %+v", e)
	}
}

func TestMultiDayEntryDayLimit(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	// A trip counts on every one of its days.
	for range MaxEntriesPerDay - 1 {
		f.entry(t, g, g.ada, "x", "2026-11-01")
	}
	trip := f.trip(t, g, "Cabin trip", "2026-10-30", "2026-11-02")
	if _, err := f.svc.CreateEntry(ctx, g.cal.ID, g.ada, EntryInput{Title: "y", Date: "2026-11-01", Color: "sage"}); !errors.Is(err, ErrDayFull) {
		t.Fatalf("adding to a day the trip filled: %v", err)
	}
	if _, err := f.svc.CreateEntry(ctx, g.cal.ID, g.ada, EntryInput{Title: "y", Date: "2026-10-25", EndDate: "2026-11-05", Color: "sage"}); !errors.Is(err, ErrDayFull) {
		t.Fatalf("a trip over the full day: %v", err)
	}
	// Changing the trip's dates does not count it against itself.
	end := "2026-11-04"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, EntryChanges{EndDate: &end}); err != nil {
		t.Fatalf("longer trip over its own day: %v", err)
	}
	// Moving it off the day makes room again.
	start := "2026-11-02"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, trip.ID, EntryChanges{Date: &start}); err != nil {
		t.Fatal(err)
	}
	f.entry(t, g, g.ada, "y", "2026-11-01")
}

func TestEntriesStayInTheirCalendar(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	e := f.entry(t, g, g.ada, "Dinner", "2026-10-01")
	other := f.create(t, g.ada, "Other")
	outsider := f.user(t, "dora")

	// The entry is not reachable through another calendar, even its owner's.
	if _, err := f.svc.GetEntry(ctx, other.ID, g.ada, e.ID); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("through another calendar: %v", err)
	}
	title := "Moved"
	if _, err := f.svc.UpdateEntry(ctx, other.ID, g.ada, e.ID, EntryChanges{Title: &title}); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("editing through another calendar: %v", err)
	}
	if err := f.svc.DeleteEntry(ctx, other.ID, g.ada, e.ID); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("deleting through another calendar: %v", err)
	}
	if _, err := f.svc.Respond(ctx, other.ID, g.ada, e.ID, ReplyAccepted); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("answering through another calendar: %v", err)
	}

	// Outsiders do not learn that the calendar exists.
	if _, err := f.svc.ListEntries(ctx, g.cal.ID, outsider, "2026-10"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider listing: %v", err)
	}
	if _, err := f.svc.GetEntry(ctx, g.cal.ID, outsider, e.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider reading: %v", err)
	}
	if _, err := f.svc.Respond(ctx, g.cal.ID, outsider, e.ID, ReplyAccepted); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider answering: %v", err)
	}
}

func TestResponses(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	e := f.entry(t, g, g.bob, "Ski day", "2026-10-31")

	respond := func(user uuid.UUID, r Reply) Entry {
		t.Helper()
		e, err := f.svc.Respond(ctx, g.cal.ID, user, e.ID, r)
		if err != nil {
			t.Fatalf("respond %q: %v", r, err)
		}
		return e
	}
	respond(g.carla, ReplyAccepted)
	respond(g.ada, ReplyDeclined)
	e = respond(g.bob, ReplyAccepted)
	if !slices.Equal(e.Accepted, []uuid.UUID{g.carla, g.bob}) || !slices.Equal(e.Declined, []uuid.UUID{g.ada}) {
		t.Fatalf("answers = %+v / %+v", e.Accepted, e.Declined)
	}
	if e.Reply(g.carla) != ReplyAccepted || e.Reply(g.ada) != ReplyDeclined {
		t.Fatalf("Reply = %q, %q", e.Reply(g.carla), e.Reply(g.ada))
	}

	// Changing your mind moves you; taking it back makes you pending again.
	e = respond(g.carla, ReplyDeclined)
	if !slices.Equal(e.Accepted, []uuid.UUID{g.bob}) || !slices.Equal(e.Declined, []uuid.UUID{g.ada, g.carla}) {
		t.Fatalf("after carla changed her mind = %+v / %+v", e.Accepted, e.Declined)
	}
	e = respond(g.ada, ReplyPending)
	if e.Reply(g.ada) != ReplyPending || len(e.Declined) != 1 {
		t.Fatalf("after ada took it back = %+v", e.Declined)
	}
	respond(g.ada, ReplyPending) // nothing to take back is fine
	if _, err := f.svc.Respond(ctx, g.cal.ID, g.ada, e.ID, "maybe"); !errors.Is(err, ErrInvalidReply) {
		t.Fatalf("maybe: %v", err)
	}

	// The listing carries the same answers.
	list, err := f.svc.ListEntries(ctx, g.cal.ID, g.ada, "2026-10")
	if err != nil || len(list) != 1 || !slices.Equal(list[0].Accepted, e.Accepted) || !slices.Equal(list[0].Declined, e.Declined) {
		t.Fatalf("list = %+v, %v", list, err)
	}

	// A new title keeps the answers, a new date clears them.
	title := "Ski day!"
	if e, _ = f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, e.ID, EntryChanges{Title: &title}); len(e.Accepted)+len(e.Declined) != 2 {
		t.Fatalf("after rename = %+v", e)
	}
	same := "2026-10-31"
	if e, _ = f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, e.ID, EntryChanges{Date: &same}); len(e.Accepted)+len(e.Declined) != 2 {
		t.Fatalf("after setting the same date = %+v", e)
	}
	moved := "2026-11-07"
	e, err = f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, e.ID, EntryChanges{Date: &moved})
	if err != nil || FormatDate(e.Day) != moved || len(e.Accepted)+len(e.Declined) != 0 {
		t.Fatalf("after moving = %+v, %v", e, err)
	}
}

func TestLeavingDropsAnswers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	e := f.entry(t, g, g.bob, "Ski day", "2026-10-31")
	for _, u := range []uuid.UUID{g.bob, g.carla} {
		if _, err := f.svc.Respond(ctx, g.cal.ID, u, e.ID, ReplyAccepted); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.svc.RemoveMember(ctx, g.cal.ID, g.carla, g.carla); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RemoveMember(ctx, g.cal.ID, g.ada, g.bob); err != nil {
		t.Fatal(err)
	}
	e, err := f.svc.GetEntry(ctx, g.cal.ID, g.ada, e.ID)
	if err != nil || len(e.Accepted) != 0 {
		t.Fatalf("after they left = %+v, %v", e, err)
	}
	// The entry still says who added it.
	if e.CreatedBy == nil || e.CreatedBy.Username != "bob" {
		t.Fatalf("created by = %+v", e.CreatedBy)
	}
	// Coming back starts without old answers.
	f.invite(t, g.cal, g.ada, g.carla, "")
	if e, _ = f.svc.GetEntry(ctx, g.cal.ID, g.carla, e.ID); e.Reply(g.carla) != ReplyPending {
		t.Fatalf("carla after rejoining = %q", e.Reply(g.carla))
	}
}

func TestDemotedEditorCannotEdit(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	e := f.entry(t, g, g.bob, "Ski day", "2026-10-31")
	viewer := RoleViewer
	if _, err := f.svc.UpdateMember(ctx, g.cal.ID, g.ada, g.bob, MemberChanges{Role: &viewer}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteEntry(ctx, g.cal.ID, g.bob, e.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("demoted editor deleting their own entry: %v", err)
	}
	if _, err := f.svc.Respond(ctx, g.cal.ID, g.bob, e.ID, ReplyAccepted); err != nil {
		t.Fatalf("viewer answering: %v", err)
	}
}

func TestEntryLimits(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)

	// Fill a day from two editors at once: the limit holds exactly.
	var wg sync.WaitGroup
	errs := make([]error, MaxEntriesPerDay+4)
	for i := range errs {
		by := g.ada
		if i%2 == 1 {
			by = g.bob
		}
		wg.Go(func() {
			_, errs[i] = f.svc.CreateEntry(ctx, g.cal.ID, by, EntryInput{Title: "x", Date: "2026-10-01", Color: "sage"})
		})
	}
	wg.Wait()
	created := 0
	for _, err := range errs {
		switch {
		case err == nil:
			created++
		case !errors.Is(err, ErrDayFull):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if created != MaxEntriesPerDay {
		t.Fatalf("created %d entries on one day, want %d", created, MaxEntriesPerDay)
	}

	// Moving another entry onto the full day fails too.
	e := f.entry(t, g, g.ada, "Elsewhere", "2026-10-02")
	full := "2026-10-01"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, e.ID, EntryChanges{Date: &full}); !errors.Is(err, ErrDayFull) {
		t.Fatalf("moving onto a full day: %v", err)
	}
	// But editing an entry that is already there does not.
	list, _ := f.svc.ListEntries(ctx, g.cal.ID, g.ada, "2026-10")
	title := "renamed"
	if _, err := f.svc.UpdateEntry(ctx, g.cal.ID, g.ada, list[0].ID, EntryChanges{Title: &title, Date: &full}); err != nil {
		t.Fatalf("editing on a full day: %v", err)
	}
}

func TestDeletingCalendarDeletesEntries(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	g := newGroup(t, f)
	e := f.entry(t, g, g.ada, "Dinner", "2026-10-01")
	if _, err := f.svc.Respond(ctx, g.cal.ID, g.carla, e.ID, ReplyAccepted); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Delete(ctx, g.cal.ID, g.ada); err != nil {
		t.Fatal(err)
	}
	if n, err := f.q.CountEntries(ctx, g.cal.ID); err != nil || n != 0 {
		t.Fatalf("entries left: %d, %v", n, err)
	}
}
