package fakeapi

import (
	"net/http"
	"slices"

	"github.com/transcript1/cli/internal/api"
)

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.findOwned(w, r)
	if j == nil {
		return
	}

	s.poll(j)
	query := r.URL.Query()
	format := query.Get("format")
	if format == "" {
		format = "json"
	}

	if !slices.Contains([]string{"json", "text", "srt", "vtt"}, format) {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "format must be json, text, srt or vtt", nil)
		return
	}

	if format == "json" {
		writeJSON(w, http.StatusOK, s.serialize(j, true))
		return
	}

	if j.status == api.StatusCompleted {
		contentType, body := j.render(format, query.Get("timestamps") == "true")
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))

		return
	}

	status := http.StatusOK
	if !isTerminal(j.status) {
		status = http.StatusAccepted
	}

	writeJSON(w, status, s.serialize(j, true))
}
