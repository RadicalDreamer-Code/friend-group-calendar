package mcpserver

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/calendars"
)

// tools are the MCP tools for one signed-in user.
type tools struct {
	s *Server
	p auth.Principal
}

type calendarOut struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role" jsonschema:"owner, editor or viewer"`
	MemberCount   int    `json:"member_count"`
	CanAddEntries bool   `json:"can_add_entries" jsonschema:"true for owners and editors"`
}

type listCalendarsOut struct {
	Calendars []calendarOut `json:"calendars"`
}

func (t *tools) listCalendars(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listCalendarsOut, error) {
	list, err := t.s.calendars.List(ctx, t.p.User.ID)
	if err != nil {
		return nil, listCalendarsOut{}, explain(ctx, err)
	}
	out := listCalendarsOut{Calendars: make([]calendarOut, 0, len(list))}
	var text strings.Builder
	fmt.Fprintf(&text, "%s:", plural(len(list), "calendar", "calendars"))
	for _, c := range list {
		cal := calendarOut{
			ID:            c.ID.String(),
			Name:          c.Name,
			Role:          string(c.Me.Role),
			MemberCount:   c.MemberCount,
			CanAddEntries: c.Me.Can(calendars.ActionEditEntries),
		}
		out.Calendars = append(out.Calendars, cal)
		fmt.Fprintf(&text, "\n- %s (id %s): %s, %s", cal.Name, cal.ID, cal.Role, plural(cal.MemberCount, "member", "members"))
		if !cal.CanAddEntries {
			text.WriteString(", view only")
		}
	}
	return textResult(text.String()), out, nil
}

type listEntriesIn struct {
	CalendarID string `json:"calendar_id"`
	From       string `json:"from,omitempty"`
	To         string `json:"to,omitempty"`
}

type entryOut struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Date    string `json:"date" jsonschema:"first day, YYYY-MM-DD"`
	EndDate string `json:"end_date" jsonschema:"last day, YYYY-MM-DD; the same as date for a single day"`
	Color   string `json:"color"`
	Cover   string `json:"cover,omitempty"`
	// Creator is left out once the account that added the entry is gone.
	Creator  string   `json:"creator,omitempty" jsonschema:"username of who added it"`
	Going    []string `json:"going" jsonschema:"usernames of members who are going"`
	CantGo   []string `json:"cant_go" jsonschema:"usernames of members who can't go"`
	MyAnswer string   `json:"my_answer" jsonschema:"the user's own answer: going, cant_go or none"`
	URL      string   `json:"url" jsonschema:"the entry in the web app"`
}

type calendarRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type listEntriesOut struct {
	Calendar calendarRef `json:"calendar"`
	From     string      `json:"from"`
	To       string      `json:"to"`
	Entries  []entryOut  `json:"entries"`
}

func (t *tools) listEntries(ctx context.Context, _ *mcp.CallToolRequest, in listEntriesIn) (*mcp.CallToolResult, listEntriesOut, error) {
	id, err := parseCalendarID(in.CalendarID)
	if err != nil {
		return nil, listEntriesOut{}, err
	}
	from := cmp.Or(in.From, calendars.FormatDate(t.s.today()))
	to := in.To
	if to == "" {
		day, err := calendars.ParseDate(from)
		if err != nil {
			return nil, listEntriesOut{}, explain(ctx, err)
		}
		// Capped at the last day entries can be on, so a `from` late in 2099
		// does not end in a range ListEntriesRange refuses.
		last := day.AddDate(0, 0, DefaultRangeDays)
		if last.After(calendars.LastDay) {
			last = calendars.LastDay
		}
		to = calendars.FormatDate(last)
	}
	cal, err := t.s.calendars.Get(ctx, id, t.p.User.ID)
	if err != nil {
		return nil, listEntriesOut{}, explain(ctx, err)
	}
	entries, err := t.s.calendars.ListEntriesRange(ctx, id, t.p.User.ID, from, to)
	if err != nil {
		return nil, listEntriesOut{}, explain(ctx, err)
	}
	names := usernames(cal)
	out := listEntriesOut{
		Calendar: calendarRef{ID: cal.ID.String(), Name: cal.Name},
		From:     from,
		To:       to,
		Entries:  make([]entryOut, 0, len(entries)),
	}
	var text strings.Builder
	fmt.Fprintf(&text, "%s in %s from %s to %s", plural(len(entries), "entry", "entries"), cal.Name, from, to)
	if len(entries) > 0 {
		text.WriteString(":")
	}
	for _, e := range entries {
		o := t.entry(e, names)
		out.Entries = append(out.Entries, o)
		fmt.Fprintf(&text, "\n- %s: %s", days(o), o.Title)
		if len(o.Going) > 0 {
			fmt.Fprintf(&text, " (going: %s)", strings.Join(o.Going, ", "))
		}
	}
	return textResult(text.String()), out, nil
}

type createEntryIn struct {
	CalendarID string `json:"calendar_id"`
	Title      string `json:"title"`
	Date       string `json:"date"`
	EndDate    string `json:"end_date,omitempty"`
	Color      string `json:"color,omitempty"`
	Cover      string `json:"cover,omitempty"`
}

func (t *tools) createEntry(ctx context.Context, _ *mcp.CallToolRequest, in createEntryIn) (*mcp.CallToolResult, entryOut, error) {
	id, err := parseCalendarID(in.CalendarID)
	if err != nil {
		return nil, entryOut{}, err
	}
	cal, err := t.s.calendars.Get(ctx, id, t.p.User.ID)
	if err != nil {
		return nil, entryOut{}, explain(ctx, err)
	}
	if !cal.Me.Can(calendars.ActionEditEntries) {
		// The service checks this again; asking here first keeps a viewer's
		// attempts from counting against the rate limit.
		return nil, entryOut{}, explain(ctx, calendars.ErrForbidden)
	}
	input := calendars.EntryInput{
		Title:   in.Title,
		Date:    in.Date,
		EndDate: in.EndDate,
		Color:   cmp.Or(in.Color, calendars.Colors[0]),
		Cover:   in.Cover,
	}
	// Bad input does not count against the rate limit either, so a model
	// that fixes its mistakes is not locked out.
	if err := input.Validate(); err != nil {
		return nil, entryOut{}, explain(ctx, err)
	}
	if ok, wait := t.s.creates.Allow(t.p.User.ID.String()); !ok {
		return nil, entryOut{}, explain(ctx, &auth.RateLimitError{RetryAfter: wait})
	}
	e, err := t.s.calendars.CreateEntry(ctx, id, t.p.User.ID, input)
	if err != nil {
		return nil, entryOut{}, explain(ctx, err)
	}
	out := t.entry(e, usernames(cal))
	return textResult(fmt.Sprintf("Added %q to %s on %s: %s", out.Title, cal.Name, days(out), out.URL)), out, nil
}

func (t *tools) entry(e calendars.Entry, names map[uuid.UUID]string) entryOut {
	out := entryOut{
		ID:       e.ID.String(),
		Title:    e.Title,
		Date:     calendars.FormatDate(e.Day),
		EndDate:  calendars.FormatDate(e.EndDay),
		Color:    e.Color,
		Cover:    e.Cover,
		Going:    lookup(names, e.Accepted),
		CantGo:   lookup(names, e.Declined),
		MyAnswer: "none",
		URL:      fmt.Sprintf("%s/calendars/%s/entries/%s", t.s.appURL, e.CalendarID, e.ID),
	}
	if e.CreatedBy != nil {
		out.Creator = e.CreatedBy.Username
	}
	switch e.Reply(t.p.User.ID) {
	case calendars.ReplyAccepted:
		out.MyAnswer = "going"
	case calendars.ReplyDeclined:
		out.MyAnswer = "cant_go"
	}
	return out
}

func usernames(cal calendars.Calendar) map[uuid.UUID]string {
	names := make(map[uuid.UUID]string, len(cal.Members))
	for _, m := range cal.Members {
		names[m.ID] = m.Username
	}
	return names
}

// lookup returns the usernames of ids, skipping anyone no longer a member.
func lookup(names map[uuid.UUID]string, ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := names[id]; ok {
			out = append(out, name)
		}
	}
	return out
}

func days(e entryOut) string {
	if e.EndDate == e.Date {
		return e.Date
	}
	return e.Date + " to " + e.EndDate
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
