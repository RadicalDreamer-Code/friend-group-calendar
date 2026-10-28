package calendars

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/store"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/testutil"
)

func TestCan(t *testing.T) {
	owner := Membership{Role: RoleOwner}
	editor := Membership{Role: RoleEditor}
	viewer := Membership{Role: RoleViewer}
	inviter := Membership{Role: RoleViewer, CanInvite: true}
	everyone := Membership{Role: RoleViewer, MembersCanInvite: true}

	tests := []struct {
		name string
		m    Membership
		a    Action
		want bool
	}{
		{"everyone views", viewer, ActionView, true},
		{"owner invites", owner, ActionInvite, true},
		{"editor without right cannot invite", editor, ActionInvite, false},
		{"viewer without right cannot invite", viewer, ActionInvite, false},
		{"member with right invites", inviter, ActionInvite, true},
		{"calendar-wide right", everyone, ActionInvite, true},
		{"owner manages", owner, ActionManage, true},
		{"editor cannot manage", editor, ActionManage, false},
		{"invite right is not manage", inviter, ActionManage, false},
		{"owner transfers", owner, ActionTransfer, true},
		{"editor cannot transfer", editor, ActionTransfer, false},
		{"owner deletes", owner, ActionDelete, true},
		{"editor cannot delete", editor, ActionDelete, false},
		{"owner cannot leave", owner, ActionLeave, false},
		{"editor leaves", editor, ActionLeave, true},
		{"viewer leaves", viewer, ActionLeave, true},
		{"owner edits entries", owner, ActionEditEntries, true},
		{"editor edits entries", editor, ActionEditEntries, true},
		{"viewer cannot edit entries", viewer, ActionEditEntries, false},
		{"invite right is not editing entries", inviter, ActionEditEntries, false},
		{"viewer responds", viewer, ActionRespond, true},
		{"owner responds", owner, ActionRespond, true},
		{"unknown action", owner, Action(99), false},
	}
	for _, tt := range tests {
		if got := tt.m.Can(tt.a); got != tt.want {
			t.Errorf("%s: Can = %v, want %v", tt.name, got, tt.want)
		}
	}
	if err := owner.require(ActionLeave); !errors.Is(err, ErrMustTransfer) {
		t.Errorf("owner leaving: %v, want ErrMustTransfer", err)
	}
	if err := editor.require(ActionDelete); !errors.Is(err, ErrForbidden) {
		t.Errorf("editor deleting: %v, want ErrForbidden", err)
	}
}

func TestNormalizeName(t *testing.T) {
	if got, err := NormalizeName("  Ski trip 2027 ⛷️ "); err != nil || got != "Ski trip 2027 ⛷️" {
		t.Errorf("NormalizeName = %q, %v", got, err)
	}
	if _, err := NormalizeName(strings.Repeat("ü", MaxNameLength)); err != nil {
		t.Errorf("%d characters should be fine: %v", MaxNameLength, err)
	}
	for _, in := range []string{"", "   ", strings.Repeat("a", MaxNameLength+1), "a\nb", "tab\there"} {
		if _, err := NormalizeName(in); !errors.Is(err, ErrInvalidName) {
			t.Errorf("NormalizeName(%q) should fail", in)
		}
	}
}

type fixture struct {
	svc *Service
	q   *store.Queries
	db  *pgxpool.Pool
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testutil.DB(t)
	return &fixture{svc: NewService(pool), q: store.New(pool), db: pool}
}

func (f *fixture) user(t *testing.T, username string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	u, err := f.q.UpsertUserByEmail(ctx, uuid.NewString()+"@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if username != "" {
		if _, err := f.q.UpdateUsername(ctx, store.UpdateUsernameParams{ID: u.ID, Username: username}); err != nil {
			t.Fatal(err)
		}
	}
	return u.ID
}

func (f *fixture) befriend(t *testing.T, a, b uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.q.CreateFriendRequest(ctx, store.CreateFriendRequestParams{RequesterID: a, AddresseeID: b}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.q.AcceptFriendRequest(ctx, store.AcceptFriendRequestParams{RequesterID: a, AddresseeID: b}); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) create(t *testing.T, owner uuid.UUID, name string) Calendar {
	t.Helper()
	c, err := f.svc.Create(context.Background(), owner, name)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *fixture) invite(t *testing.T, cal Calendar, from, to uuid.UUID, role Role) Calendar {
	t.Helper()
	c, err := f.svc.Invite(context.Background(), cal.ID, from, to, role)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	return c
}

func member(c Calendar, id uuid.UUID) (Member, bool) {
	for _, m := range c.Members {
		if m.ID == id {
			return m, true
		}
	}
	return Member{}, false
}

func TestCreateListAndGet(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	f.befriend(t, ada, bob)

	ski := f.create(t, ada, "  Ski trip ")
	if ski.Name != "Ski trip" || ski.Me.Role != RoleOwner || len(ski.Members) != 1 || ski.Members[0].Username != "ada" {
		t.Fatalf("created = %+v", ski)
	}
	f.create(t, ada, "beach")
	f.invite(t, ski, ada, bob, "")

	list, err := f.svc.List(ctx, ada)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by name regardless of case.
	if len(list) != 2 || list[0].Name != "beach" || list[1].Name != "Ski trip" {
		t.Fatalf("list = %+v", list)
	}
	if s := list[1]; s.MemberCount != 2 || len(s.Preview) != 2 || s.Preview[0].Username != "ada" || !s.Me.Can(ActionInvite) {
		t.Fatalf("ski summary = %+v", s)
	}

	// Bob sees only the calendar he was invited to, as a viewer.
	list, err = f.svc.List(ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Me.Role != RoleViewer || list[0].Me.Can(ActionInvite) {
		t.Fatalf("bob's list = %+v", list)
	}

	// Outsiders cannot tell the calendar exists.
	carla := f.user(t, "carla")
	if _, err := f.svc.Get(ctx, ski.ID, carla); !errors.Is(err, ErrNotFound) {
		t.Fatalf("outsider get: %v", err)
	}
	if _, err := f.svc.Get(ctx, uuid.New(), ada); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown calendar: %v", err)
	}
	if empty, err := f.svc.List(ctx, carla); err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list = %#v, %v", empty, err)
	}
}

func TestPreviewIsLimited(t *testing.T) {
	f := newFixture(t)
	owner := f.user(t, "owner")
	cal := f.create(t, owner, "Big group")
	for i := range PreviewSize + 2 {
		friend := f.user(t, "friend"+string(rune('a'+i)))
		f.befriend(t, owner, friend)
		f.invite(t, cal, owner, friend, "")
	}
	list, err := f.svc.List(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].MemberCount != PreviewSize+3 || len(list[0].Preview) != PreviewSize || list[0].Preview[0].ID != owner {
		t.Fatalf("summary = %+v", list[0])
	}
}

func TestInviteRules(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	carla := f.user(t, "carla")
	dan := f.user(t, "dan")
	stranger := f.user(t, "stranger")
	f.befriend(t, ada, bob)
	f.befriend(t, ada, carla)
	f.befriend(t, bob, dan)
	cal := f.create(t, ada, "Trip")

	if _, err := f.svc.Invite(ctx, cal.ID, ada, stranger, ""); !errors.Is(err, ErrNotFriends) {
		t.Fatalf("invite stranger: %v", err)
	}
	if _, err := f.svc.Invite(ctx, cal.ID, ada, bob, RoleOwner); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("invite as owner: %v", err)
	}
	if _, err := f.svc.Invite(ctx, cal.ID, ada, uuid.New(), ""); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("invite unknown user: %v", err)
	}
	if _, err := f.svc.Invite(ctx, cal.ID, ada, ada, ""); !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("invite self: %v", err)
	}

	c := f.invite(t, cal, ada, bob, RoleEditor)
	if m, _ := member(c, bob); m.Role != RoleEditor || m.CanInvite {
		t.Fatalf("bob = %+v", m)
	}
	if _, err := f.svc.Invite(ctx, cal.ID, ada, bob, ""); !errors.Is(err, ErrAlreadyMember) {
		t.Fatalf("invite twice: %v", err)
	}

	// Bob is an editor without the invite right.
	if _, err := f.svc.Invite(ctx, cal.ID, bob, dan, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("invite without right: %v", err)
	}
	editor := RoleEditor
	yes := true
	if _, err := f.svc.UpdateMember(ctx, cal.ID, ada, bob, MemberChanges{CanInvite: &yes}); err != nil {
		t.Fatal(err)
	}
	// Bob can now invite his own friends, as editor at most.
	c = f.invite(t, cal, bob, dan, RoleEditor)
	if m, _ := member(c, dan); m.Role != RoleEditor {
		t.Fatalf("dan = %+v", m)
	}

	// A viewer with the right still cannot invite editors.
	viewer := RoleViewer
	if _, err := f.svc.UpdateMember(ctx, cal.ID, ada, bob, MemberChanges{Role: &viewer}); err != nil {
		t.Fatal(err)
	}
	f.befriend(t, bob, carla)
	if _, err := f.svc.Invite(ctx, cal.ID, bob, carla, editor); !errors.Is(err, ErrRoleTooHigh) {
		t.Fatalf("viewer inviting editor: %v", err)
	}
	c = f.invite(t, cal, bob, carla, "")
	if m, _ := member(c, carla); m.Role != RoleViewer {
		t.Fatalf("carla = %+v", m)
	}
}

func TestMembersCanInvite(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	carla := f.user(t, "carla")
	f.befriend(t, ada, bob)
	f.befriend(t, bob, carla)
	cal := f.create(t, ada, "Trip")
	f.invite(t, cal, ada, bob, "")

	if _, err := f.svc.Invite(ctx, cal.ID, bob, carla, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("before: %v", err)
	}
	yes := true
	// Only the owner changes it.
	if _, err := f.svc.Update(ctx, cal.ID, bob, Changes{MembersCanInvite: &yes}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer changing settings: %v", err)
	}
	c, err := f.svc.Update(ctx, cal.ID, ada, Changes{MembersCanInvite: &yes})
	if err != nil || !c.MembersCanInvite || c.Name != "Trip" {
		t.Fatalf("update = %+v, %v", c, err)
	}
	c = f.invite(t, cal, bob, carla, "")
	if !c.Me.Can(ActionInvite) || len(c.Members) != 3 {
		t.Fatalf("after = %+v", c)
	}
}

func TestOwnerManagesMembers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	carla := f.user(t, "carla")
	f.befriend(t, ada, bob)
	f.befriend(t, ada, carla)
	cal := f.create(t, ada, "Trip")
	f.invite(t, cal, ada, bob, "")
	f.invite(t, cal, ada, carla, "")

	editor := RoleEditor
	owner := RoleOwner
	if _, err := f.svc.UpdateMember(ctx, cal.ID, bob, carla, MemberChanges{Role: &editor}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer changing roles: %v", err)
	}
	if _, err := f.svc.UpdateMember(ctx, cal.ID, ada, carla, MemberChanges{Role: &owner}); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("role owner: %v", err)
	}
	if _, err := f.svc.UpdateMember(ctx, cal.ID, ada, ada, MemberChanges{Role: &editor}); !errors.Is(err, ErrOwnerMember) {
		t.Fatalf("demoting the owner: %v", err)
	}
	if _, err := f.svc.UpdateMember(ctx, cal.ID, ada, uuid.New(), MemberChanges{Role: &editor}); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("unknown member: %v", err)
	}
	c, err := f.svc.UpdateMember(ctx, cal.ID, ada, carla, MemberChanges{Role: &editor})
	if err != nil {
		t.Fatal(err)
	}
	// Owner first, then editors, then viewers.
	if c.Members[1].ID != carla || c.Members[1].Role != RoleEditor || c.Members[2].ID != bob {
		t.Fatalf("members = %+v", c.Members)
	}

	// Rename: owner only, and validated.
	name := "Summer"
	if _, err := f.svc.Update(ctx, cal.ID, carla, Changes{Name: &name}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("editor renaming: %v", err)
	}
	bad := " "
	if _, err := f.svc.Update(ctx, cal.ID, ada, Changes{Name: &bad}); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("empty name: %v", err)
	}
	if c, err := f.svc.Update(ctx, cal.ID, ada, Changes{Name: &name}); err != nil || c.Name != "Summer" {
		t.Fatalf("rename = %+v, %v", c, err)
	}

	// Remove and leave.
	if err := f.svc.RemoveMember(ctx, cal.ID, carla, bob); !errors.Is(err, ErrForbidden) {
		t.Fatalf("editor removing: %v", err)
	}
	if err := f.svc.RemoveMember(ctx, cal.ID, ada, bob); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Get(ctx, cal.ID, bob); !errors.Is(err, ErrNotFound) {
		t.Fatalf("removed member still sees the calendar: %v", err)
	}
	if err := f.svc.RemoveMember(ctx, cal.ID, ada, bob); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("removing twice: %v", err)
	}
	if err := f.svc.RemoveMember(ctx, cal.ID, ada, ada); !errors.Is(err, ErrMustTransfer) {
		t.Fatalf("owner leaving: %v", err)
	}
	if err := f.svc.RemoveMember(ctx, cal.ID, carla, carla); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if c, _ := f.svc.Get(ctx, cal.ID, ada); len(c.Members) != 1 {
		t.Fatalf("members after leaving = %+v", c.Members)
	}
}

func TestTransferAndDelete(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	f.befriend(t, ada, bob)
	cal := f.create(t, ada, "Trip")
	f.invite(t, cal, ada, bob, "")

	if _, err := f.svc.TransferOwnership(ctx, cal.ID, bob, bob); !errors.Is(err, ErrForbidden) {
		t.Fatalf("viewer taking over: %v", err)
	}
	if _, err := f.svc.TransferOwnership(ctx, cal.ID, ada, uuid.New()); !errors.Is(err, ErrMemberNotFound) {
		t.Fatalf("transfer to outsider: %v", err)
	}
	c, err := f.svc.TransferOwnership(ctx, cal.ID, ada, bob)
	if err != nil {
		t.Fatal(err)
	}
	if c.Me.Role != RoleEditor || !c.Me.Can(ActionInvite) {
		t.Fatalf("previous owner = %+v", c.Me)
	}
	if m, _ := member(c, bob); m.Role != RoleOwner || !m.CanInvite {
		t.Fatalf("new owner = %+v", m)
	}
	if err := f.svc.Delete(ctx, cal.ID, ada); !errors.Is(err, ErrForbidden) {
		t.Fatalf("previous owner deleting: %v", err)
	}
	if err := f.svc.RemoveMember(ctx, cal.ID, ada, ada); err != nil {
		t.Fatalf("previous owner leaving: %v", err)
	}
	if err := f.svc.Delete(ctx, cal.ID, bob); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Get(ctx, cal.ID, bob); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted calendar: %v", err)
	}
	if err := f.svc.Delete(ctx, cal.ID, bob); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleting twice: %v", err)
	}
}

// Two transfers at once must still leave exactly one owner.
func TestConcurrentTransfers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	bob := f.user(t, "bob")
	carla := f.user(t, "carla")
	f.befriend(t, ada, bob)
	f.befriend(t, ada, carla)
	cal := f.create(t, ada, "Trip")
	f.invite(t, cal, ada, bob, "")
	f.invite(t, cal, ada, carla, "")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, to := range []uuid.UUID{bob, carla} {
		wg.Go(func() {
			_, errs[i] = f.svc.TransferOwnership(ctx, cal.ID, ada, to)
		})
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case !errors.Is(err, ErrForbidden):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	c, err := f.svc.Get(ctx, cal.ID, ada)
	if err != nil {
		t.Fatal(err)
	}
	owners := 0
	for _, m := range c.Members {
		if m.Role == RoleOwner {
			owners++
		}
	}
	if succeeded != 1 || owners != 1 {
		t.Fatalf("%d transfers succeeded, %d owners", succeeded, owners)
	}
}

func TestOwnedLimit(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	ada := f.user(t, "ada")
	for range MaxOwned {
		f.create(t, ada, "Calendar")
	}
	if _, err := f.svc.Create(ctx, ada, "One more"); !errors.Is(err, ErrTooManyCalendars) {
		t.Fatalf("over the limit: %v", err)
	}

	// Nor can someone at the limit be handed another one.
	bob := f.user(t, "bob")
	f.befriend(t, bob, ada)
	cal := f.create(t, bob, "Bob's")
	f.invite(t, cal, bob, ada, "")
	if _, err := f.svc.TransferOwnership(ctx, cal.ID, bob, ada); !errors.Is(err, ErrNewOwnerHasTooMany) {
		t.Fatalf("transfer over the limit: %v", err)
	}
	c, err := f.svc.Get(ctx, cal.ID, bob)
	if err != nil {
		t.Fatal(err)
	}
	if c.Me.Role != RoleOwner {
		t.Fatalf("failed transfer changed the owner: %+v", c.Me)
	}
}
