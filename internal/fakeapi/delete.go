package fakeapi

import (
	"net/http"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.findOwned(w, r)
	if j == nil {
		return
	}

	if isTerminal(j.status) {
		delete(s.jobs, j.id)
		w.WriteHeader(http.StatusNoContent)

		return
	}

	j.status = api.StatusCanceled
	j.token = ""
	j.completedAt = time.Now().UTC()
	s.refund(j)
	w.WriteHeader(http.StatusNoContent)
}
