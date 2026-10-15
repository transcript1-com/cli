package fakeapi

import "net/http"

const unlimitedMinutes = 1000

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := s.MinutesLeft
	if remaining < 0 {
		remaining = unlimitedMinutes
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"object":  "account",
		"user":    map[string]any{"email": "demo@transcript1.com"},
		"plan":    map[string]any{"name": "Monthly"},
		"minutes": map[string]any{"remaining": remaining},
		"limits":  map[string]any{"max_active_jobs": s.ActiveLimit},
	})
}
