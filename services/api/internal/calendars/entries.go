package calendars

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

const (
	MaxTitleLength = 80
	// MaxEntriesPerDay keeps a day in the month view readable.
	MaxEntriesPerDay = 20
	// MaxEntries bounds one calendar.
	MaxEntries = 5000
	// MaxEntryDays is the longest an entry can last, both ends included.
	MaxEntryDays = 31
	// MaxRangeDays is the most days ListEntriesRange covers, both ends included.
	MaxRangeDays = 92

	dateLayout  = "2006-01-02"
	monthLayout = "2006-01"
)

// Entries are whole days between FirstDay and LastDay.
var (
	FirstDay = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	LastDay  = time.Date(2099, time.December, 31, 0, 0, 0, 0, time.UTC)
)

// Colors are the preset entry colors, the same as in the client's tokens.
var Colors = []string{"fjord", "sage", "clay", "rose", "plum", "sand", "moss", "slate"}

// Covers are the scenes an entry can show as its cover. The client draws
// them, so there are no uploads.
var Covers = []string{"lake", "mountains", "beach", "city", "night_out", "dinner", "game_night"}

var (
	ErrInvalidTitle = fmt.Errorf("entry names are 1–%d characters", MaxTitleLength)
	ErrInvalidDate  = errors.New("dates are YYYY-MM-DD, from 2000 to 2099")
	// ErrInvalidEndDate: the end is before the start or too far after it.
	ErrInvalidEndDate = fmt.Errorf("an entry ends on or after its first day and lasts at most %d days", MaxEntryDays)
	ErrInvalidMonth   = errors.New("months are YYYY-MM, from 2000 to 2099")
	ErrInvalidRange   = fmt.Errorf("a range ends on or after its first day and covers at most %d days", MaxRangeDays)
	ErrInvalidColor   = errors.New("unknown entry color")
	ErrInvalidCover   = errors.New("unknown entry cover")
	ErrInvalidReply   = errors.New("the answer must be accepted or declined")
	// ErrEntryNotFound: no such entry in this calendar.
	ErrEntryNotFound  = errors.New("entry not found")
	ErrDayFull        = fmt.Errorf("a day can have at most %d entries", MaxEntriesPerDay)
	ErrTooManyEntries = fmt.Errorf("a calendar can have at most %d entries", MaxEntries)
)

// Reply is a member's answer to an entry. Not having answered yet is
// ReplyPending, which is never stored.
type Reply string

const (
	ReplyAccepted Reply = "accepted"
	ReplyDeclined Reply = "declined"
	ReplyPending  Reply = ""
)

// ParseReply accepts the answers a member can give: accepted or declined.
func ParseReply(s string) (Reply, error) {
	switch r := Reply(s); r {
	case ReplyAccepted, ReplyDeclined:
		return r, nil
	}
	return "", ErrInvalidReply
}

// Entry runs over one or more whole days and carries everyone's answers. Members who are
// in neither list have not answered.
type Entry struct {
	ID         uuid.UUID
	CalendarID uuid.UUID
	Title      string
	// Day is midnight UTC of the entry's first date, EndDay of its last.
	// They are the same day for single-day entries.
	Day    time.Time
	EndDay time.Time
	Color  string
	// Cover is one of Covers, or empty for none.
	Cover string
	// CreatedBy is nil once the account that added it is deleted.
	CreatedBy *Person
	CreatedAt time.Time
	UpdatedAt time.Time
	// Accepted and Declined hold member IDs in the order they answered.
	Accepted []uuid.UUID
	Declined []uuid.UUID
}

// Reply returns the user's answer to the entry.
func (e Entry) Reply(userID uuid.UUID) Reply {
	switch {
	case slices.Contains(e.Accepted, userID):
		return ReplyAccepted
	case slices.Contains(e.Declined, userID):
		return ReplyDeclined
	}
	return ReplyPending
}

// NormalizeTitle trims an entry name and checks its length.
func NormalizeTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(title)
	if n == 0 || n > MaxTitleLength || strings.ContainsFunc(title, unicode.IsControl) {
		return "", ErrInvalidTitle
	}
	return title, nil
}

// ParseDate reads a YYYY-MM-DD date between FirstDay and LastDay.
func ParseDate(s string) (time.Time, error) {
	d, err := time.Parse(dateLayout, s)
	if err != nil || d.Before(FirstDay) || d.After(LastDay) {
		return time.Time{}, ErrInvalidDate
	}
	return d, nil
}

// FormatDate writes a day the way ParseDate reads it.
func FormatDate(d time.Time) string {
	return d.Format(dateLayout)
}

// ParseMonth reads a YYYY-MM month and returns its first day.
func ParseMonth(s string) (time.Time, error) {
	m, err := time.Parse(monthLayout, s)
	if err != nil || m.Before(FirstDay) || m.After(LastDay) {
		return time.Time{}, ErrInvalidMonth
	}
	return m, nil
}

// checkSpan fails unless an entry from day to end lasts 1 to MaxEntryDays days.
func checkSpan(day, end time.Time) error {
	if end.After(LastDay) {
		return ErrInvalidDate
	}
	if end.Before(day) || end.After(day.AddDate(0, 0, MaxEntryDays-1)) {
		return ErrInvalidEndDate
	}
	return nil
}

func parseColor(s string) (string, error) {
	if !slices.Contains(Colors, s) {
		return "", ErrInvalidColor
	}
	return s, nil
}

// parseCover checks a cover; an empty one is no cover, stored as NULL.
func parseCover(s string) (*string, error) {
	if s == "" {
		return nil, nil
	}
	if !slices.Contains(Covers, s) {
		return nil, ErrInvalidCover
	}
	return &s, nil
}

// EntryInput is a new entry as the user typed it. An empty EndDate makes
// it a single-day entry, an empty Cover one without a cover.
type EntryInput struct {
	Title   string
	Date    string
	EndDate string
	Color   string
	Cover   string
}

// Validate reports the error CreateEntry would give for in, without looking
// at the calendar.
func (in EntryInput) Validate() error {
	_, err := in.parse()
	return err
}

type parsedEntry struct {
	title, color string
	day, end     time.Time
	cover        *string
}

func (in EntryInput) parse() (parsedEntry, error) {
	title, err := NormalizeTitle(in.Title)
	if err != nil {
		return parsedEntry{}, err
	}
	day, err := ParseDate(in.Date)
	if err != nil {
		return parsedEntry{}, err
	}
	end := day
	if in.EndDate != "" {
		if end, err = ParseDate(in.EndDate); err != nil {
			return parsedEntry{}, err
		}
	}
	if err := checkSpan(day, end); err != nil {
		return parsedEntry{}, err
	}
	color, err := parseColor(in.Color)
	if err != nil {
		return parsedEntry{}, err
	}
	cover, err := parseCover(in.Cover)
	if err != nil {
		return parsedEntry{}, err
	}
	return parsedEntry{title: title, color: color, day: day, end: end, cover: cover}, nil
}

// EntryChanges to an entry; nil fields stay as they are. A new Date
// without an EndDate keeps the entry's length, and an empty Cover removes
// the cover.
type EntryChanges struct {
	Title   *string
	Date    *string
	EndDate *string
	Color   *string
	Cover   *string
}

// ListEntries returns the entries that overlap one month (YYYY-MM).
func (s *Service) ListEntries(ctx context.Context, calendarID, userID uuid.UUID, month string) ([]Entry, error) {
	from, err := ParseMonth(month)
	if err != nil {
		return nil, err
	}
	return s.listEntries(ctx, calendarID, userID, from, from.AddDate(0, 1, 0))
}

// ListEntriesRange returns the entries that overlap the days from `from` to
// `to` (YYYY-MM-DD, both included), at most MaxRangeDays of them.
func (s *Service) ListEntriesRange(ctx context.Context, calendarID, userID uuid.UUID, from, to string) ([]Entry, error) {
	first, err := ParseDate(from)
	if err != nil {
		return nil, err
	}
	last, err := ParseDate(to)
	if err != nil {
		return nil, err
	}
	if last.Before(first) || last.After(first.AddDate(0, 0, MaxRangeDays-1)) {
		return nil, ErrInvalidRange
	}
	return s.listEntries(ctx, calendarID, userID, first, last.AddDate(0, 0, 1))
}

// listEntries returns the entries that overlap the days from `from` up to,
// not including, `until`.
func (s *Service) listEntries(ctx context.Context, calendarID, userID uuid.UUID, from, until time.Time) ([]Entry, error) {
	if _, _, err := s.view(ctx, calendarID, userID); err != nil {
		return nil, err
	}
	rows, err := s.q.ListEntriesBetween(ctx, store.ListEntriesBetweenParams{
		CalendarID: calendarID, FromDay: from, UntilDay: until,
	})
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	entries := make([]Entry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, entry(store.GetEntryWithCreatorRow(r)))
	}
	return s.withReplies(ctx, entries)
}

// NextEntry returns the first entry that has not ended before the day `from`
// (YYYY-MM-DD), the client's local today, so an entry that is still running
// comes first. ok is false when nothing is planned from then on.
func (s *Service) NextEntry(ctx context.Context, calendarID, userID uuid.UUID, from string) (e Entry, ok bool, err error) {
	day, err := ParseDate(from)
	if err != nil {
		return Entry{}, false, err
	}
	if _, _, err := s.view(ctx, calendarID, userID); err != nil {
		return Entry{}, false, err
	}
	row, err := s.q.NextEntry(ctx, store.NextEntryParams{CalendarID: calendarID, FromDay: day})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, false, nil
	} else if err != nil {
		return Entry{}, false, fmt.Errorf("next entry: %w", err)
	}
	entries, err := s.withReplies(ctx, []Entry{entry(store.GetEntryWithCreatorRow(row))})
	if err != nil {
		return Entry{}, false, err
	}
	return entries[0], true, nil
}

// GetEntry returns one entry of a calendar the user is a member of.
func (s *Service) GetEntry(ctx context.Context, calendarID, userID, entryID uuid.UUID) (Entry, error) {
	if _, _, err := s.view(ctx, calendarID, userID); err != nil {
		return Entry{}, err
	}
	return s.loadEntry(ctx, calendarID, entryID)
}

// CreateEntry adds an entry; the owner and editors may.
func (s *Service) CreateEntry(ctx context.Context, calendarID, userID uuid.UUID, in EntryInput) (Entry, error) {
	p, err := in.parse()
	if err != nil {
		return Entry{}, err
	}
	var id uuid.UUID
	err = s.change(ctx, calendarID, userID, ActionEditEntries, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		if err := checkRoom(ctx, q, cal.ID, p.day, p.end, uuid.Nil); err != nil {
			return err
		}
		total, err := q.CountEntries(ctx, cal.ID)
		if err != nil {
			return fmt.Errorf("count entries: %w", err)
		}
		if total >= MaxEntries {
			return ErrTooManyEntries
		}
		e, err := q.CreateEntry(ctx, store.CreateEntryParams{
			CalendarID: cal.ID, Title: p.title, Day: p.day, EndDay: p.end, Color: p.color, Cover: p.cover, CreatedBy: &userID,
		})
		id = e.ID
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return s.loadEntry(ctx, calendarID, id)
}

// UpdateEntry changes an entry; the owner and editors may. Changing its
// first or last day clears everyone's answers, since they were for the old
// days.
func (s *Service) UpdateEntry(ctx context.Context, calendarID, userID, entryID uuid.UUID, c EntryChanges) (Entry, error) {
	var (
		title, color string
		cover        *string
		day, end     time.Time
		err          error
	)
	if c.Title != nil {
		if title, err = NormalizeTitle(*c.Title); err != nil {
			return Entry{}, err
		}
	}
	if c.Date != nil {
		if day, err = ParseDate(*c.Date); err != nil {
			return Entry{}, err
		}
	}
	if c.EndDate != nil {
		if end, err = ParseDate(*c.EndDate); err != nil {
			return Entry{}, err
		}
	}
	if c.Color != nil {
		if color, err = parseColor(*c.Color); err != nil {
			return Entry{}, err
		}
	}
	if c.Cover != nil {
		if cover, err = parseCover(*c.Cover); err != nil {
			return Entry{}, err
		}
	}
	err = s.change(ctx, calendarID, userID, ActionEditEntries, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		e, err := getEntry(ctx, q, cal.ID, entryID)
		if err != nil {
			return err
		}
		params := store.UpdateEntryParams{ID: e.ID, CalendarID: cal.ID, Title: e.Title, Day: e.Day, EndDay: e.EndDay, Color: e.Color, Cover: e.Cover}
		if c.Title != nil {
			params.Title = title
		}
		if c.Color != nil {
			params.Color = color
		}
		if c.Cover != nil {
			params.Cover = cover
		}
		if c.Date != nil {
			params.Day = day
			params.EndDay = day.Add(e.EndDay.Sub(e.Day))
		}
		if c.EndDate != nil {
			params.EndDay = end
		}
		if !params.Day.Equal(e.Day) || !params.EndDay.Equal(e.EndDay) {
			if err := checkSpan(params.Day, params.EndDay); err != nil {
				return err
			}
			if err := checkRoom(ctx, q, cal.ID, params.Day, params.EndDay, e.ID); err != nil {
				return err
			}
			if err := q.DeleteEntryResponses(ctx, e.ID); err != nil {
				return fmt.Errorf("clear responses: %w", err)
			}
		}
		_, err = q.UpdateEntry(ctx, params)
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return s.loadEntry(ctx, calendarID, entryID)
}

// DeleteEntry removes an entry; the owner and editors may.
func (s *Service) DeleteEntry(ctx context.Context, calendarID, userID, entryID uuid.UUID) error {
	return s.change(ctx, calendarID, userID, ActionEditEntries, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		n, err := q.DeleteEntry(ctx, store.DeleteEntryParams{ID: entryID, CalendarID: cal.ID})
		if err != nil {
			return fmt.Errorf("delete entry: %w", err)
		}
		if n == 0 {
			return ErrEntryNotFound
		}
		return nil
	})
}

// Respond sets the user's answer to an entry; ReplyPending takes it back.
// Every member may answer.
func (s *Service) Respond(ctx context.Context, calendarID, userID, entryID uuid.UUID, reply Reply) (Entry, error) {
	if reply != ReplyPending {
		if _, err := ParseReply(string(reply)); err != nil {
			return Entry{}, err
		}
	}
	err := s.change(ctx, calendarID, userID, ActionRespond, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		if _, err := getEntry(ctx, q, cal.ID, entryID); err != nil {
			return err
		}
		if reply == ReplyPending {
			return q.DeleteEntryResponse(ctx, store.DeleteEntryResponseParams{EntryID: entryID, UserID: userID})
		}
		return q.SetEntryResponse(ctx, store.SetEntryResponseParams{EntryID: entryID, UserID: userID, Status: string(reply)})
	})
	if err != nil {
		return Entry{}, err
	}
	return s.loadEntry(ctx, calendarID, entryID)
}

// checkRoom fails when any day from first to last already has
// MaxEntriesPerDay entries, not counting the entry `except`.
func checkRoom(ctx context.Context, q *store.Queries, calendarID uuid.UUID, first, last time.Time, except uuid.UUID) error {
	n, err := q.MostEntriesOnADay(ctx, store.MostEntriesOnADayParams{
		CalendarID: calendarID, FirstDay: first, LastDay: last, ExceptID: except,
	})
	if err != nil {
		return fmt.Errorf("count entries per day: %w", err)
	}
	if n >= MaxEntriesPerDay {
		return ErrDayFull
	}
	return nil
}

func getEntry(ctx context.Context, q *store.Queries, calendarID, entryID uuid.UUID) (store.Entry, error) {
	e, err := q.GetEntry(ctx, store.GetEntryParams{ID: entryID, CalendarID: calendarID})
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrEntryNotFound
	} else if err != nil {
		return e, fmt.Errorf("load entry: %w", err)
	}
	return e, nil
}

func (s *Service) loadEntry(ctx context.Context, calendarID, entryID uuid.UUID) (Entry, error) {
	row, err := s.q.GetEntryWithCreator(ctx, store.GetEntryWithCreatorParams{ID: entryID, CalendarID: calendarID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrEntryNotFound
	} else if err != nil {
		return Entry{}, fmt.Errorf("load entry: %w", err)
	}
	entries, err := s.withReplies(ctx, []Entry{entry(row)})
	if err != nil {
		return Entry{}, err
	}
	return entries[0], nil
}

// withReplies fills in the answers of the entries' current members.
func (s *Service) withReplies(ctx context.Context, entries []Entry) ([]Entry, error) {
	if len(entries) == 0 {
		return entries, nil
	}
	ids := make([]uuid.UUID, 0, len(entries))
	index := make(map[uuid.UUID]int, len(entries))
	for i, e := range entries {
		ids = append(ids, e.ID)
		index[e.ID] = i
	}
	rows, err := s.q.ListEntryResponses(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list responses: %w", err)
	}
	for _, r := range rows {
		e := &entries[index[r.EntryID]]
		switch Reply(r.Status) {
		case ReplyAccepted:
			e.Accepted = append(e.Accepted, r.UserID)
		case ReplyDeclined:
			e.Declined = append(e.Declined, r.UserID)
		}
	}
	return entries, nil
}

func entry(r store.GetEntryWithCreatorRow) Entry {
	e := Entry{
		ID:         r.ID,
		CalendarID: r.CalendarID,
		Title:      r.Title,
		Day:        r.Day,
		EndDay:     r.EndDay,
		Color:      r.Color,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		Accepted:   []uuid.UUID{},
		Declined:   []uuid.UUID{},
	}
	if r.Cover != nil {
		e.Cover = *r.Cover
	}
	if r.CreatedBy != nil && r.CreatorUsername != "" {
		e.CreatedBy = &Person{ID: *r.CreatedBy, Username: r.CreatorUsername, AvatarKey: r.CreatorAvatarKey}
	}
	return e
}
