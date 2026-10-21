// Package calendars manages shared calendars, their members and what each
// member is allowed to do.
//
// Every calendar has exactly one owner. The owner renames and deletes it,
// manages members and their roles, decides who may invite, and can hand
// ownership to another member. Editors will add and edit entries, viewers
// only see and answer them. Inviting is open to the owner and to members
// with the invite right, which the owner grants per member or to everyone.
//
// Who may do what is decided in one place, Membership.Can, and every change
// goes through Service.change, which locks the calendar and checks it.
package calendars

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
)

const (
	MaxNameLength = 50
	// MaxMembers bounds one calendar; it is for a group of friends.
	MaxMembers = 50
	// MaxOwned bounds the calendars one user can own.
	MaxOwned = 50
	// PreviewSize is how many members the calendar list shows per calendar.
	PreviewSize = 4
)

var (
	ErrInvalidName = fmt.Errorf("calendar names are 1–%d characters", MaxNameLength)
	// ErrNotFound: no such calendar, or the user is not a member. Both look
	// the same so outsiders cannot probe which calendars exist.
	ErrNotFound       = errors.New("calendar not found")
	ErrForbidden      = errors.New("not allowed")
	ErrMustTransfer   = errors.New("the owner has to hand over ownership before leaving")
	ErrInvalidRole    = errors.New("role must be editor or viewer")
	ErrRoleTooHigh    = errors.New("cannot invite someone with a higher role than your own")
	ErrNotFriends     = errors.New("you can only invite friends")
	ErrUserNotFound   = errors.New("user not found")
	ErrAlreadyMember  = errors.New("already a member")
	ErrMemberNotFound = errors.New("not a member of this calendar")
	// ErrOwnerMember: the owner's role cannot be changed and the owner cannot
	// be removed; ownership moves with TransferOwnership.
	ErrOwnerMember      = errors.New("the owner's membership cannot be changed")
	ErrCalendarFull     = fmt.Errorf("a calendar can have at most %d members", MaxMembers)
	ErrTooManyCalendars = fmt.Errorf("you can own at most %d calendars", MaxOwned)
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

func (r Role) rank() int {
	switch r {
	case RoleOwner:
		return 2
	case RoleEditor:
		return 1
	default:
		return 0
	}
}

// ParseRole accepts the roles a member can be given: editor or viewer.
// Owner is not one of them; ownership is transferred.
func ParseRole(s string) (Role, error) {
	switch r := Role(s); r {
	case RoleEditor, RoleViewer:
		return r, nil
	}
	return "", ErrInvalidRole
}

// Action is something a member may or may not do in a calendar.
type Action int

const (
	ActionView Action = iota
	ActionInvite
	// ActionManage covers the calendar's settings and its members: renaming,
	// invite rights, roles and removing people.
	ActionManage
	ActionTransfer
	ActionDelete
	ActionLeave
)

// Membership is a user's place in a calendar.
type Membership struct {
	Role Role
	// CanInvite is the member's own invite right.
	CanInvite bool
	// MembersCanInvite is the calendar-wide invite right.
	MembersCanInvite bool
}

// Can is the single place that decides what a member may do.
func (m Membership) Can(a Action) bool {
	switch a {
	case ActionView:
		return true
	case ActionInvite:
		return m.Role == RoleOwner || m.CanInvite || m.MembersCanInvite
	case ActionLeave:
		return m.Role != RoleOwner
	case ActionManage, ActionTransfer, ActionDelete:
		return m.Role == RoleOwner
	}
	return false
}

func (m Membership) require(a Action) error {
	switch {
	case m.Can(a):
		return nil
	case a == ActionLeave:
		return ErrMustTransfer
	default:
		return ErrForbidden
	}
}

// Person is what members see of each other: never the email.
type Person struct {
	ID        uuid.UUID
	Username  string
	AvatarKey *string
}

type Member struct {
	Person
	Role Role
	// CanInvite is the member's own invite right; always true for the owner.
	CanInvite bool
	JoinedAt  time.Time
}

// Summary is a calendar in the user's list.
type Summary struct {
	ID          uuid.UUID
	Name        string
	Me          Membership
	MemberCount int
	// Preview holds the first PreviewSize members, owner first.
	Preview []Person
}

// Calendar is one calendar with all of its members.
type Calendar struct {
	ID               uuid.UUID
	Name             string
	MembersCanInvite bool
	CreatedAt        time.Time
	Me               Membership
	Members          []Member
}

type Service struct {
	db *pgxpool.Pool
	q  *store.Queries
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db, q: store.New(db)}
}

// NormalizeName trims a calendar name and checks its length.
func NormalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(name)
	if n == 0 || n > MaxNameLength || strings.ContainsFunc(name, unicode.IsControl) {
		return "", ErrInvalidName
	}
	return name, nil
}

// List returns the user's calendars by name.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Summary, error) {
	rows, err := s.q.ListCalendarsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list calendars: %w", err)
	}
	out := make([]Summary, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	previews, err := s.q.ListCalendarMemberPreviews(ctx, store.ListCalendarMemberPreviewsParams{CalendarIds: ids, PerCalendar: PreviewSize})
	if err != nil {
		return nil, fmt.Errorf("list member previews: %w", err)
	}
	byCalendar := map[uuid.UUID][]Person{}
	for _, p := range previews {
		byCalendar[p.CalendarID] = append(byCalendar[p.CalendarID], Person{ID: p.ID, Username: p.Username, AvatarKey: p.AvatarKey})
	}
	for _, r := range rows {
		preview := byCalendar[r.ID]
		if preview == nil {
			preview = []Person{}
		}
		out = append(out, Summary{
			ID:          r.ID,
			Name:        r.Name,
			Me:          Membership{Role: Role(r.Role), CanInvite: r.CanInvite, MembersCanInvite: r.MembersCanInvite},
			MemberCount: int(r.MemberCount),
			Preview:     preview,
		})
	}
	return out, nil
}

// Get returns a calendar the user is a member of.
func (s *Service) Get(ctx context.Context, calendarID, userID uuid.UUID) (Calendar, error) {
	cal, err := s.q.GetCalendar(ctx, calendarID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Calendar{}, ErrNotFound
	} else if err != nil {
		return Calendar{}, fmt.Errorf("load calendar: %w", err)
	}
	me, err := membership(ctx, s.q, cal, userID)
	if err != nil {
		return Calendar{}, err
	}
	return s.load(ctx, s.q, cal, me)
}

// Create makes a calendar with the user as its owner.
func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, rawName string) (Calendar, error) {
	name, err := NormalizeName(rawName)
	if err != nil {
		return Calendar{}, err
	}
	var cal store.Calendar
	err = s.inTx(ctx, func(q *store.Queries) error {
		// Serializes this user's creates so two at once cannot pass the limit.
		if _, err := q.GetUserForUpdate(ctx, ownerID); err != nil {
			return fmt.Errorf("lock user: %w", err)
		}
		owned, err := q.CountOwnedCalendars(ctx, ownerID)
		if err != nil {
			return fmt.Errorf("count calendars: %w", err)
		}
		if owned >= MaxOwned {
			return ErrTooManyCalendars
		}
		if cal, err = q.CreateCalendar(ctx, name); err != nil {
			return fmt.Errorf("create calendar: %w", err)
		}
		_, err = q.AddCalendarMember(ctx, store.AddCalendarMemberParams{
			CalendarID: cal.ID, UserID: ownerID, Role: string(RoleOwner), CanInvite: true,
		})
		return err
	})
	if err != nil {
		return Calendar{}, err
	}
	return s.Get(ctx, cal.ID, ownerID)
}

// Changes to a calendar's settings; nil fields stay as they are.
type Changes struct {
	Name             *string
	MembersCanInvite *bool
}

func (s *Service) Update(ctx context.Context, calendarID, userID uuid.UUID, c Changes) (Calendar, error) {
	name := ""
	if c.Name != nil {
		var err error
		if name, err = NormalizeName(*c.Name); err != nil {
			return Calendar{}, err
		}
	}
	return s.changeAndGet(ctx, calendarID, userID, ActionManage, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		params := store.UpdateCalendarParams{ID: cal.ID, Name: cal.Name, MembersCanInvite: cal.MembersCanInvite}
		if c.Name != nil {
			params.Name = name
		}
		if c.MembersCanInvite != nil {
			params.MembersCanInvite = *c.MembersCanInvite
		}
		_, err := q.UpdateCalendar(ctx, params)
		return err
	})
}

// Delete removes the calendar for everyone.
func (s *Service) Delete(ctx context.Context, calendarID, userID uuid.UUID) error {
	return s.change(ctx, calendarID, userID, ActionDelete, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		_, err := q.DeleteCalendar(ctx, cal.ID)
		return err
	})
}

// Invite adds a friend of the inviter as a member. They join with the given
// role (viewer if empty), which cannot be above the inviter's own.
func (s *Service) Invite(ctx context.Context, calendarID, inviterID, inviteeID uuid.UUID, role Role) (Calendar, error) {
	if role == "" {
		role = RoleViewer
	}
	if _, err := ParseRole(string(role)); err != nil {
		return Calendar{}, err
	}
	return s.changeAndGet(ctx, calendarID, inviterID, ActionInvite, func(q *store.Queries, cal store.Calendar, me Membership) error {
		if role.rank() > me.Role.rank() {
			return ErrRoleTooHigh
		}
		if _, err := q.GetCalendarMember(ctx, store.GetCalendarMemberParams{CalendarID: cal.ID, UserID: inviteeID}); err == nil {
			return ErrAlreadyMember
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("load member: %w", err)
		}
		invitee, err := q.GetUser(ctx, inviteeID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && invitee.Username == nil) {
			return ErrUserNotFound
		} else if err != nil {
			return fmt.Errorf("load user: %w", err)
		}
		friends, err := q.AreFriends(ctx, store.AreFriendsParams{UserA: inviterID, UserB: inviteeID})
		if err != nil {
			return fmt.Errorf("check friendship: %w", err)
		}
		if !friends {
			return ErrNotFriends
		}
		count, err := q.CountCalendarMembers(ctx, cal.ID)
		if err != nil {
			return fmt.Errorf("count members: %w", err)
		}
		if count >= MaxMembers {
			return ErrCalendarFull
		}
		_, err = q.AddCalendarMember(ctx, store.AddCalendarMemberParams{
			CalendarID: cal.ID, UserID: inviteeID, Role: string(role), InvitedBy: &inviterID,
		})
		return err
	})
}

// MemberChanges to one member; nil fields stay as they are.
type MemberChanges struct {
	Role      *Role
	CanInvite *bool
}

// UpdateMember changes another member's role or invite right.
func (s *Service) UpdateMember(ctx context.Context, calendarID, userID, memberID uuid.UUID, c MemberChanges) (Calendar, error) {
	if c.Role != nil {
		if _, err := ParseRole(string(*c.Role)); err != nil {
			return Calendar{}, err
		}
	}
	return s.changeAndGet(ctx, calendarID, userID, ActionManage, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		m, err := otherMember(ctx, q, cal.ID, memberID)
		if err != nil {
			return err
		}
		params := store.UpdateCalendarMemberParams{CalendarID: cal.ID, UserID: memberID, Role: m.Role, CanInvite: m.CanInvite}
		if c.Role != nil {
			params.Role = string(*c.Role)
		}
		if c.CanInvite != nil {
			params.CanInvite = *c.CanInvite
		}
		_, err = q.UpdateCalendarMember(ctx, params)
		return err
	})
}

// RemoveMember takes someone out of the calendar. Removing yourself is
// leaving, which every member but the owner may do.
func (s *Service) RemoveMember(ctx context.Context, calendarID, userID, memberID uuid.UUID) error {
	action := ActionManage
	if memberID == userID {
		action = ActionLeave
	}
	return s.change(ctx, calendarID, userID, action, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		if action == ActionManage {
			if _, err := otherMember(ctx, q, cal.ID, memberID); err != nil {
				return err
			}
		}
		_, err := q.DeleteCalendarMember(ctx, store.DeleteCalendarMemberParams{CalendarID: cal.ID, UserID: memberID})
		return err
	})
}

// TransferOwnership makes another member the owner. The previous owner
// stays on as an editor who can invite.
func (s *Service) TransferOwnership(ctx context.Context, calendarID, ownerID, newOwnerID uuid.UUID) (Calendar, error) {
	return s.changeAndGet(ctx, calendarID, ownerID, ActionTransfer, func(q *store.Queries, cal store.Calendar, _ Membership) error {
		if newOwnerID == ownerID {
			return nil
		}
		next, err := otherMember(ctx, q, cal.ID, newOwnerID)
		if err != nil {
			return err
		}
		// Demote first: the one-owner index is checked after each statement.
		if _, err := q.UpdateCalendarMember(ctx, store.UpdateCalendarMemberParams{
			CalendarID: cal.ID, UserID: ownerID, Role: string(RoleEditor), CanInvite: true,
		}); err != nil {
			return fmt.Errorf("demote owner: %w", err)
		}
		_, err = q.UpdateCalendarMember(ctx, store.UpdateCalendarMemberParams{
			CalendarID: cal.ID, UserID: newOwnerID, Role: string(RoleOwner), CanInvite: next.CanInvite,
		})
		return err
	})
}

// change runs fn in a transaction once the calendar is locked and the user
// is known to be allowed the action. Every change to a calendar or its
// members goes through here, so neither the check nor the lock is skipped.
func (s *Service) change(ctx context.Context, calendarID, userID uuid.UUID, action Action, fn func(*store.Queries, store.Calendar, Membership) error) error {
	return s.inTx(ctx, func(q *store.Queries) error {
		cal, err := q.GetCalendarForUpdate(ctx, calendarID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return fmt.Errorf("lock calendar: %w", err)
		}
		me, err := membership(ctx, q, cal, userID)
		if err != nil {
			return err
		}
		if err := me.require(action); err != nil {
			return err
		}
		return fn(q, cal, me)
	})
}

// changeAndGet is change followed by the calendar as it is afterwards.
func (s *Service) changeAndGet(ctx context.Context, calendarID, userID uuid.UUID, action Action, fn func(*store.Queries, store.Calendar, Membership) error) (Calendar, error) {
	if err := s.change(ctx, calendarID, userID, action, fn); err != nil {
		return Calendar{}, err
	}
	return s.Get(ctx, calendarID, userID)
}

func (s *Service) inTx(ctx context.Context, fn func(*store.Queries) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// membership loads the user's membership; not being a member is ErrNotFound.
func membership(ctx context.Context, q *store.Queries, cal store.Calendar, userID uuid.UUID) (Membership, error) {
	m, err := q.GetCalendarMember(ctx, store.GetCalendarMemberParams{CalendarID: cal.ID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrNotFound
	} else if err != nil {
		return Membership{}, fmt.Errorf("load membership: %w", err)
	}
	return Membership{Role: Role(m.Role), CanInvite: m.CanInvite, MembersCanInvite: cal.MembersCanInvite}, nil
}

// otherMember loads a member that someone else is about to change; the
// owner cannot be changed that way.
func otherMember(ctx context.Context, q *store.Queries, calendarID, memberID uuid.UUID) (store.CalendarMember, error) {
	m, err := q.GetCalendarMember(ctx, store.GetCalendarMemberParams{CalendarID: calendarID, UserID: memberID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return m, ErrMemberNotFound
	case err != nil:
		return m, fmt.Errorf("load member: %w", err)
	case Role(m.Role) == RoleOwner:
		return m, ErrOwnerMember
	}
	return m, nil
}

func (s *Service) load(ctx context.Context, q *store.Queries, cal store.Calendar, me Membership) (Calendar, error) {
	rows, err := q.ListCalendarMembers(ctx, cal.ID)
	if err != nil {
		return Calendar{}, fmt.Errorf("list members: %w", err)
	}
	out := Calendar{
		ID:               cal.ID,
		Name:             cal.Name,
		MembersCanInvite: cal.MembersCanInvite,
		CreatedAt:        cal.CreatedAt,
		Me:               me,
		Members:          make([]Member, 0, len(rows)),
	}
	for _, r := range rows {
		role := Role(r.Role)
		out.Members = append(out.Members, Member{
			Person:    Person{ID: r.ID, Username: r.Username, AvatarKey: r.AvatarKey},
			Role:      role,
			CanInvite: role == RoleOwner || r.CanInvite,
			JoinedAt:  r.JoinedAt,
		})
	}
	return out, nil
}
