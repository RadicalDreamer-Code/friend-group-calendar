package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/blob"
	"github.com/RadicalDreamer-Code/friend-group-calendar/services/api/internal/users"
)

// handleGetAvatar serves avatars from the bucket, so the bucket can stay
// private and works the same on RustFS, R2 or S3. Keys are random and change
// on every upload, so responses are public and cached for good.
func (s *Server) handleGetAvatar(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "userID") + "/" + chi.URLParam(r, "file")
	if !users.ValidAvatarKey(key) {
		writeError(w, http.StatusNotFound, "not_found", "Avatar not found.")
		return
	}
	body, size, err := s.avatars.Get(r.Context(), key)
	if errors.Is(err, blob.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "Avatar not found.")
		return
	}
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	defer body.Close()

	h := w.Header()
	h.Set("Content-Type", "image/webp")
	h.Set("Content-Length", strconv.FormatInt(size, 10))
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("X-Content-Type-Options", "nosniff")
	// <img> on the web app's origin loads it; nothing else needs to.
	h.Set("Cross-Origin-Resource-Policy", "same-site")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		slog.WarnContext(r.Context(), "stream avatar", "key", key, "err", err)
	}
}
