package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/auth"
)

// maxJSONBody bounds JSON request bodies; none of them need more.
const maxJSONBody = 64 << 10

type errorBody struct {
	Error apiError `json:"error"`
}

// apiError is the body of every non-2xx response. Code is stable and meant
// for the client to switch on; Message is readable and may change.
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: apiError{Code: code, Message: message}})
}

// writeInternalError logs the cause and answers with a generic 500.
func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "internal error",
		"err", err,
		"path", r.URL.Path,
		"request_id", middleware.GetReqID(r.Context()),
	)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
}

// writeRateLimited answers 429 with Retry-After when err is a rate limit.
func writeRateLimited(w http.ResponseWriter, err error) bool {
	var rl *auth.RateLimitError
	if !errors.As(err, &rl) {
		return false
	}
	seconds := int(math.Ceil(rl.RetryAfter.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(max(seconds, 1)))
	writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Please wait a moment and try again.")
	return true
}

// decodeJSON reads a JSON body into v and answers 400 itself on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "The request body is not valid JSON.")
		return false
	}
	return true
}
