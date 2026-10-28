// Package mcpserver is the MCP endpoint that lets an AI assistant see a
// user's calendars and add entries for them.
//
// It runs as Streamable HTTP in stateless mode with plain JSON responses:
// every POST is one self-contained exchange, so there are no sessions to
// keep. The caller authenticates the request with an access token and puts
// the principal in the context with WithPrincipal; each request then gets
// a server whose tools act as that user. The tools call calendars.Service,
// so every permission check stays where it already is.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/calendars"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/ratelimit"
)

const (
	// DefaultRangeDays is how far list_entries looks without a `to`.
	DefaultRangeDays = 30
	// maxBody bounds a request; tool calls are small JSON.
	maxBody = 64 << 10
)

const instructions = `Friend Group Calendar: shared calendars that groups of friends plan get-togethers and trips in.

Entries are whole days, with no times and no time zones. A single-day entry has the same date and end_date; a trip spans several days.

When the user names a calendar, call list_calendars first to find its id. Only owners and editors can add entries; viewers can only look.`

// Limits bound what one user can do through MCP, on top of the calendar's
// own limits per day and per calendar.
type Limits struct {
	// Creates caps create_entry calls per user.
	Creates auth.Limit
}

var DefaultLimits = Limits{Creates: auth.Limit{N: 30, Window: time.Hour}}

type Options struct {
	Calendars *calendars.Service
	// AppURL is the web app's address, for links to entries.
	AppURL string
	Limits Limits
	// Today returns the current day; it defaults to today in UTC.
	Today func() time.Time
}

type Server struct {
	calendars *calendars.Service
	appURL    string
	creates   *ratelimit.Limiter
	today     func() time.Time
	// Every request builds its own mcp.Server; the cache keeps that from
	// working out the tool schemas again each time.
	schemas *mcp.SchemaCache
}

func New(o Options) *Server {
	today := o.Today
	if today == nil {
		today = func() time.Time { return time.Now().UTC() }
	}
	return &Server{
		calendars: o.Calendars,
		appURL:    strings.TrimRight(o.AppURL, "/"),
		creates:   ratelimit.New(o.Limits.Creates.N, o.Limits.Creates.Window),
		today:     today,
		schemas:   mcp.NewSchemaCache(),
	}
}

type principalKey struct{}

// WithPrincipal marks a request as made by p. Handler serves only requests
// that carry one.
func WithPrincipal(ctx context.Context, p auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Handler serves MCP over Streamable HTTP. Authentication is the caller's
// job; see WithPrincipal.
func (s *Server) Handler() http.Handler {
	h := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		p, ok := r.Context().Value(principalKey{}).(auth.Principal)
		if !ok {
			// The SDK answers 400 when there is no server.
			return nil
		}
		return s.serverFor(p)
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, MaxRequestBodyBytes: maxBody})
	return h
}

func (s *Server) serverFor(p auth.Principal) *mcp.Server {
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "friend-group-calendar", Title: "Friend Group Calendar", Version: "1.0.0"},
		&mcp.ServerOptions{Instructions: instructions, SchemaCache: s.schemas},
	)
	t := &tools{s: s, p: p}
	addTool(srv, p, &mcp.Tool{
		Name:        "list_calendars",
		Title:       "List calendars",
		Description: "Lists the calendars the user is a member of, with their role and whether they can add entries.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)},
	}, t.listCalendars)
	addTool(srv, p, &mcp.Tool{
		Name:  "list_entries",
		Title: "List entries",
		Description: fmt.Sprintf("Lists the entries of one calendar that overlap the days from `from` to `to`, both included, sorted by date. "+
			"Entries are whole days with no times or time zones; a trip that starts before `from` or ends after `to` is included. "+
			"`from` defaults to today (UTC) and `to` to %d days later; the range covers at most %d days. "+
			"Each entry says who is going, who can't, and the user's own answer.", DefaultRangeDays, calendars.MaxRangeDays),
		InputSchema: listEntriesSchema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)},
	}, t.listEntries)
	addTool(srv, p, &mcp.Tool{
		Name:  "create_entry",
		Title: "Add an entry",
		Description: fmt.Sprintf("Adds an entry to a calendar where the user is the owner or an editor. "+
			"Entries are whole days with no times or time zones: give the first day as `date` and, for a trip, the last day as `end_date` "+
			"(at most %d days in all). Leave out `end_date` for a single day.", calendars.MaxEntryDays),
		InputSchema: createEntrySchema,
		Annotations: &mcp.ToolAnnotations{DestructiveHint: new(false), OpenWorldHint: new(false)},
	}, t.createEntry)
	return srv
}

// addTool registers a tool and logs each call with the user and token ID,
// never the token.
func addTool[In, Out any](srv *mcp.Server, p auth.Principal, tool *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	mcp.AddTool(srv, tool, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		start := time.Now()
		res, out, err := h(ctx, req, in)
		slog.InfoContext(ctx, "mcp tool call",
			"tool", tool.Name,
			"user_id", p.User.ID,
			"token_id", p.TokenID,
			"error", err != nil || (res != nil && res.IsError),
			"duration", time.Since(start),
		)
		return res, out, err
	})
}

// toolError is an error the model sees and can act on, rather than a
// protocol error.
type toolError struct{ msg string }

func (e *toolError) Error() string { return e.msg }

func failf(format string, args ...any) error {
	return &toolError{fmt.Sprintf(format, args...)}
}

// explain turns a service error into a tool error the model can act on.
// Anything unexpected is logged and reported without its details.
func explain(ctx context.Context, err error) error {
	var rl *ratelimit.Error
	switch {
	case errors.As(err, &rl):
		return failf("Too many entries added in the last hour. Try again in %s.", rl.RetryAfter.Round(time.Minute).String())
	case errors.Is(err, calendars.ErrNotFound):
		// A calendar the user is not in looks the same as one that does not exist.
		return failf("Calendar not found. Call list_calendars for the ids of the user's calendars.")
	case errors.Is(err, calendars.ErrForbidden):
		return failf("You can view this calendar but can't add to it.")
	case errors.Is(err, calendars.ErrInvalidTitle):
		return failf("The title must be 1–%d characters.", calendars.MaxTitleLength)
	case errors.Is(err, calendars.ErrInvalidDate):
		return failf("Dates are YYYY-MM-DD, from 2000-01-01 to 2099-12-31.")
	case errors.Is(err, calendars.ErrInvalidEndDate):
		return failf("end_date must be on or after date, and an entry lasts at most %d days.", calendars.MaxEntryDays)
	case errors.Is(err, calendars.ErrInvalidRange):
		return failf("`to` must be on or after `from`, and the range covers at most %d days.", calendars.MaxRangeDays)
	case errors.Is(err, calendars.ErrInvalidColor):
		return failf("color must be one of: %s.", strings.Join(calendars.Colors, ", "))
	case errors.Is(err, calendars.ErrInvalidCover):
		return failf("cover must be one of: %s.", strings.Join(calendars.Covers, ", "))
	case errors.Is(err, calendars.ErrDayFull):
		return failf("A day can have at most %d entries, and one of these days is full.", calendars.MaxEntriesPerDay)
	case errors.Is(err, calendars.ErrTooManyEntries):
		return failf("This calendar is full: it can have at most %d entries.", calendars.MaxEntries)
	}
	slog.ErrorContext(ctx, "mcp tool failed", "err", err)
	return failf("Something went wrong on the server. Please try again.")
}

func parseCalendarID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, failf("calendar_id must be a calendar id from list_calendars.")
	}
	return id, nil
}

// datePattern matches the years entries can be in; ParseDate checks the rest.
const datePattern = `^20[0-9]{2}-[0-9]{2}-[0-9]{2}$`

func dateSchema(description string) *jsonschema.Schema {
	return &jsonschema.Schema{Type: "string", Pattern: datePattern, Description: description}
}

func calendarIDSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "string", Format: "uuid", Description: "The calendar's id, from list_calendars."}
}

// The input schemas are built once: the SDK caches resolved schemas by
// pointer.
var (
	listEntriesSchema = newListEntriesSchema()
	createEntrySchema = newCreateEntrySchema()
)

func newListEntriesSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"calendar_id": calendarIDSchema(),
			"from":        dateSchema("First day, YYYY-MM-DD (2000–2099). Defaults to today in UTC."),
			"to":          dateSchema(fmt.Sprintf("Last day, YYYY-MM-DD, included. Defaults to `from` plus %d days; at most %d days in all.", DefaultRangeDays, calendars.MaxRangeDays)),
		},
		Required:             []string{"calendar_id"},
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
	}
}

func newCreateEntrySchema() *jsonschema.Schema {
	colors := make([]any, len(calendars.Colors))
	for i, c := range calendars.Colors {
		colors[i] = c
	}
	covers := make([]any, len(calendars.Covers))
	for i, c := range calendars.Covers {
		covers[i] = c
	}
	defaultColor, _ := json.Marshal(calendars.Colors[0])
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"calendar_id": calendarIDSchema(),
			"title": {
				Type: "string", MinLength: new(1), MaxLength: new(calendars.MaxTitleLength),
				Description: fmt.Sprintf("What the entry is, 1–%d characters, e.g. \"Board game night\".", calendars.MaxTitleLength),
			},
			"date":     dateSchema("First (or only) day, YYYY-MM-DD (2000–2099)."),
			"end_date": dateSchema(fmt.Sprintf("Last day for a trip, YYYY-MM-DD, on or after `date` and at most %d days in all. Leave out for a single day.", calendars.MaxEntryDays)),
			"color": {
				Type: "string", Enum: colors, Default: defaultColor,
				Description: "The entry's color in the calendar.",
			},
			"cover": {
				Type: "string", Enum: covers,
				Description: "An optional picture for the entry. Leave out for none.",
			},
		},
		Required:             []string{"calendar_id", "title", "date"},
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
	}
}
