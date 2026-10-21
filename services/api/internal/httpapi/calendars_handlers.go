package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/calendars"
)

func (s *Server) calendarPerson(p calendars.Person) personResponse {
	return s.person(p.ID, p.Username, p.AvatarKey)
}

type calendarSummaryResponse struct {
	ID   uuid.UUID      `json:"id"`
	Name string         `json:"name"`
	Role calendars.Role `json:"role"`
	// CanInvite: whether the caller may invite friends.
	CanInvite   bool             `json:"canInvite"`
	MemberCount int              `json:"memberCount"`
	Members     []personResponse `json:"members"`
}

type calendarsResponse struct {
	Calendars []calendarSummaryResponse `json:"calendars"`
}

type memberResponse struct {
	User      personResponse `json:"user"`
	Role      calendars.Role `json:"role"`
	CanInvite bool           `json:"canInvite"`
	JoinedAt  time.Time      `json:"joinedAt"`
}

type calendarResponse struct {
	ID               uuid.UUID        `json:"id"`
	Name             string           `json:"name"`
	MembersCanInvite bool             `json:"membersCanInvite"`
	CreatedAt        time.Time        `json:"createdAt"`
	Role             calendars.Role   `json:"role"`
	CanInvite        bool             `json:"canInvite"`
	Members          []memberResponse `json:"members"`
}

func (s *Server) calendarResponse(c calendars.Calendar) calendarResponse {
	resp := calendarResponse{
		ID:               c.ID,
		Name:             c.Name,
		MembersCanInvite: c.MembersCanInvite,
		CreatedAt:        c.CreatedAt,
		Role:             c.Me.Role,
		CanInvite:        c.Me.Can(calendars.ActionInvite),
		Members:          make([]memberResponse, 0, len(c.Members)),
	}
	for _, m := range c.Members {
		resp.Members = append(resp.Members, memberResponse{
			User:      s.calendarPerson(m.Person),
			Role:      m.Role,
			CanInvite: m.CanInvite,
			JoinedAt:  m.JoinedAt,
		})
	}
	return resp
}

func (s *Server) handleListCalendars(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	list, err := s.calendars.List(r.Context(), p.User.ID)
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	resp := calendarsResponse{Calendars: make([]calendarSummaryResponse, 0, len(list))}
	for _, c := range list {
		members := make([]personResponse, 0, len(c.Preview))
		for _, m := range c.Preview {
			members = append(members, s.calendarPerson(m))
		}
		resp.Calendars = append(resp.Calendars, calendarSummaryResponse{
			ID:          c.ID,
			Name:        c.Name,
			Role:        c.Me.Role,
			CanInvite:   c.Me.Can(calendars.ActionInvite),
			MemberCount: c.MemberCount,
			Members:     members,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

type createCalendarRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateCalendar(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	var req createCalendarRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.calendars.Create(r.Context(), p.User.ID, req.Name)
	if writeCalendarError(w, r, err) {
		writeJSON(w, http.StatusCreated, s.calendarResponse(c))
	}
}

func (s *Server) handleGetCalendar(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	c, err := s.calendars.Get(r.Context(), id, p.User.ID)
	if writeCalendarError(w, r, err) {
		writeJSON(w, http.StatusOK, s.calendarResponse(c))
	}
}

type updateCalendarRequest struct {
	Name             *string `json:"name"`
	MembersCanInvite *bool   `json:"membersCanInvite"`
}

func (s *Server) handleUpdateCalendar(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	var req updateCalendarRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.calendars.Update(r.Context(), id, p.User.ID, calendars.Changes{Name: req.Name, MembersCanInvite: req.MembersCanInvite})
	if writeCalendarError(w, r, err) {
		writeJSON(w, http.StatusOK, s.calendarResponse(c))
	}
}

func (s *Server) handleDeleteCalendar(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	if writeCalendarError(w, r, s.calendars.Delete(r.Context(), id, p.User.ID)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

type inviteMemberRequest struct {
	UserID uuid.UUID `json:"userId"`
	// Role is optional and defaults to viewer.
	Role calendars.Role `json:"role"`
}

func (s *Server) handleInviteMember(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	var req inviteMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.calendars.Invite(r.Context(), id, p.User.ID, req.UserID, req.Role)
	if writeCalendarError(w, r, err) {
		writeJSON(w, http.StatusOK, s.calendarResponse(c))
	}
}

type updateMemberRequest struct {
	Role      *calendars.Role `json:"role"`
	CanInvite *bool           `json:"canInvite"`
}

func (s *Server) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	member, ok := memberIDParam(w, r)
	if !ok {
		return
	}
	var req updateMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.calendars.UpdateMember(r.Context(), id, p.User.ID, member, calendars.MemberChanges{Role: req.Role, CanInvite: req.CanInvite})
	if writeCalendarError(w, r, err) {
		writeJSON(w, http.StatusOK, s.calendarResponse(c))
	}
}

// handleRemoveMember removes someone, or with the caller's own ID leaves.
func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	member, ok := memberIDParam(w, r)
	if !ok {
		return
	}
	if writeCalendarError(w, r, s.calendars.RemoveMember(r.Context(), id, p.User.ID, member)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

type transferOwnershipRequest struct {
	UserID uuid.UUID `json:"userId"`
}

func (s *Server) handleTransferOwnership(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	var req transferOwnershipRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.calendars.TransferOwnership(r.Context(), id, p.User.ID, req.UserID)
	if writeCalendarError(w, r, err) {
		writeJSON(w, http.StatusOK, s.calendarResponse(c))
	}
}

// calendarIDParam reads {calendarID}; a malformed ID cannot exist, so 404.
func calendarIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "calendarID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "calendar_not_found", "This calendar does not exist or you are no longer a member.")
		return uuid.Nil, false
	}
	return id, true
}

func memberIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "member_not_found", "This person is not a member of the calendar.")
		return uuid.Nil, false
	}
	return id, true
}

// writeCalendarError answers for a failed calendar operation and reports
// whether the handler may continue.
func writeCalendarError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, calendars.ErrInvalidName):
		writeError(w, http.StatusBadRequest, "invalid_name", "Calendar names are 1–50 characters.")
	case errors.Is(err, calendars.ErrNotFound):
		writeError(w, http.StatusNotFound, "calendar_not_found", "This calendar does not exist or you are no longer a member.")
	case errors.Is(err, calendars.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "You are not allowed to do that in this calendar.")
	case errors.Is(err, calendars.ErrMustTransfer):
		writeError(w, http.StatusConflict, "owner_must_transfer", "Make someone else the owner before you leave, or delete the calendar.")
	case errors.Is(err, calendars.ErrInvalidRole):
		writeError(w, http.StatusBadRequest, "invalid_role", "The role must be editor or viewer.")
	case errors.Is(err, calendars.ErrRoleTooHigh):
		writeError(w, http.StatusForbidden, "role_too_high", "You cannot invite someone with a higher role than your own.")
	case errors.Is(err, calendars.ErrNotFriends):
		writeError(w, http.StatusForbidden, "not_friends", "You can only invite your friends.")
	case errors.Is(err, calendars.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user_not_found", "User not found.")
	case errors.Is(err, calendars.ErrAlreadyMember):
		writeError(w, http.StatusConflict, "already_member", "They are already a member of this calendar.")
	case errors.Is(err, calendars.ErrMemberNotFound):
		writeError(w, http.StatusNotFound, "member_not_found", "This person is not a member of the calendar.")
	case errors.Is(err, calendars.ErrOwnerMember):
		writeError(w, http.StatusBadRequest, "member_is_owner", "The owner stays the owner until they make someone else the owner.")
	case errors.Is(err, calendars.ErrCalendarFull):
		writeError(w, http.StatusConflict, "calendar_full", "A calendar can have at most 50 members.")
	case errors.Is(err, calendars.ErrTooManyCalendars):
		writeError(w, http.StatusConflict, "too_many_calendars", "You can own at most 50 calendars.")
	case errors.Is(err, calendars.ErrNewOwnerHasTooMany):
		writeError(w, http.StatusConflict, "new_owner_too_many_calendars", "They already own 50 calendars, the most anyone can own.")
	default:
		writeInternalError(w, r, err)
	}
	return false
}
