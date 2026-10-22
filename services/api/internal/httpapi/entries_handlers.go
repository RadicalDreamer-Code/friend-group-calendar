package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/calendars"
)

type entryResponse struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	// Date is the entry's whole day, YYYY-MM-DD.
	Date      string          `json:"date"`
	Color     string          `json:"color"`
	CreatedBy *personResponse `json:"createdBy"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
	// Accepted and Declined are member IDs; everyone else has not answered.
	Accepted []uuid.UUID `json:"accepted"`
	Declined []uuid.UUID `json:"declined"`
}

type entriesResponse struct {
	Entries []entryResponse `json:"entries"`
}

func (s *Server) entryResponse(e calendars.Entry) entryResponse {
	resp := entryResponse{
		ID:        e.ID,
		Title:     e.Title,
		Date:      calendars.FormatDate(e.Day),
		Color:     e.Color,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		Accepted:  e.Accepted,
		Declined:  e.Declined,
	}
	if e.CreatedBy != nil {
		p := s.calendarPerson(*e.CreatedBy)
		resp.CreatedBy = &p
	}
	return resp
}

func (s *Server) handleListEntries(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	entries, err := s.calendars.ListEntries(r.Context(), id, p.User.ID, r.URL.Query().Get("month"))
	if !writeEntryError(w, r, err) {
		return
	}
	resp := entriesResponse{Entries: make([]entryResponse, 0, len(entries))}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, s.entryResponse(e))
	}
	writeJSON(w, http.StatusOK, resp)
}

type createEntryRequest struct {
	Title string `json:"title"`
	Date  string `json:"date"`
	Color string `json:"color"`
}

func (s *Server) handleCreateEntry(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, ok := calendarIDParam(w, r)
	if !ok {
		return
	}
	var req createEntryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e, err := s.calendars.CreateEntry(r.Context(), id, p.User.ID, calendars.EntryInput{Title: req.Title, Date: req.Date, Color: req.Color})
	if writeEntryError(w, r, err) {
		writeJSON(w, http.StatusCreated, s.entryResponse(e))
	}
}

func (s *Server) handleGetEntry(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, entryID, ok := entryParams(w, r)
	if !ok {
		return
	}
	e, err := s.calendars.GetEntry(r.Context(), id, p.User.ID, entryID)
	if writeEntryError(w, r, err) {
		writeJSON(w, http.StatusOK, s.entryResponse(e))
	}
}

type updateEntryRequest struct {
	Title *string `json:"title"`
	Date  *string `json:"date"`
	Color *string `json:"color"`
}

func (s *Server) handleUpdateEntry(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, entryID, ok := entryParams(w, r)
	if !ok {
		return
	}
	var req updateEntryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	e, err := s.calendars.UpdateEntry(r.Context(), id, p.User.ID, entryID, calendars.EntryChanges{Title: req.Title, Date: req.Date, Color: req.Color})
	if writeEntryError(w, r, err) {
		writeJSON(w, http.StatusOK, s.entryResponse(e))
	}
}

func (s *Server) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, entryID, ok := entryParams(w, r)
	if !ok {
		return
	}
	if writeEntryError(w, r, s.calendars.DeleteEntry(r.Context(), id, p.User.ID, entryID)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

type respondRequest struct {
	Status string `json:"status"`
}

// handleRespond accepts or declines an entry for the caller.
func (s *Server) handleRespond(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, entryID, ok := entryParams(w, r)
	if !ok {
		return
	}
	var req respondRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	reply, err := calendars.ParseReply(req.Status)
	if !writeEntryError(w, r, err) {
		return
	}
	e, err := s.calendars.Respond(r.Context(), id, p.User.ID, entryID, reply)
	if writeEntryError(w, r, err) {
		writeJSON(w, http.StatusOK, s.entryResponse(e))
	}
}

// handleDeleteResponse takes the caller's answer back.
func (s *Server) handleDeleteResponse(w http.ResponseWriter, r *http.Request) {
	p := mustPrincipal(r.Context())
	id, entryID, ok := entryParams(w, r)
	if !ok {
		return
	}
	e, err := s.calendars.Respond(r.Context(), id, p.User.ID, entryID, calendars.ReplyPending)
	if writeEntryError(w, r, err) {
		writeJSON(w, http.StatusOK, s.entryResponse(e))
	}
}

// entryParams reads {calendarID} and {entryID}.
func entryParams(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	id, ok := calendarIDParam(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	entryID, err := uuid.Parse(chi.URLParam(r, "entryID"))
	if err != nil {
		writeEntryError(w, r, calendars.ErrEntryNotFound)
		return uuid.Nil, uuid.Nil, false
	}
	return id, entryID, true
}

// writeEntryError is writeCalendarError plus the errors only entries have.
func writeEntryError(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, calendars.ErrInvalidTitle):
		writeError(w, http.StatusBadRequest, "invalid_title", "Entry names are 1–80 characters.")
	case errors.Is(err, calendars.ErrInvalidDate):
		writeError(w, http.StatusBadRequest, "invalid_date", "Pick a date between 2000 and 2099.")
	case errors.Is(err, calendars.ErrInvalidMonth):
		writeError(w, http.StatusBadRequest, "invalid_month", "The month must look like 2026-10, between 2000 and 2099.")
	case errors.Is(err, calendars.ErrInvalidColor):
		writeError(w, http.StatusBadRequest, "invalid_color", "Pick one of the entry colors.")
	case errors.Is(err, calendars.ErrInvalidReply):
		writeError(w, http.StatusBadRequest, "invalid_status", "The answer must be accepted or declined.")
	case errors.Is(err, calendars.ErrEntryNotFound):
		writeError(w, http.StatusNotFound, "entry_not_found", "This entry does not exist anymore.")
	case errors.Is(err, calendars.ErrDayFull):
		writeError(w, http.StatusConflict, "day_full", "A day can have at most 20 entries.")
	case errors.Is(err, calendars.ErrTooManyEntries):
		writeError(w, http.StatusConflict, "too_many_entries", "A calendar can have at most 5000 entries.")
	default:
		return writeCalendarError(w, r, err)
	}
	return false
}
