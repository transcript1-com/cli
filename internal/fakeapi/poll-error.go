package fakeapi

import (
	"net/http"

	"github.com/transcript1-com/cli/internal/api"
)

func (s *Server) failPoll(w http.ResponseWriter, j *job) bool {
	if isTerminal(j.status) {
		return false
	}

	if s.pollErrors >= s.PollErrors {
		return false
	}

	s.pollErrors++
	if s.PollDrop {
		panic(http.ErrAbortHandler)
	}

	status := s.PollErrorStatus
	if status == 0 {
		status = http.StatusServiceUnavailable
	}

	code := api.CodeServiceUnavailable
	if status == http.StatusNotFound {
		code = api.CodeNotFound
	}

	w.Header().Set("Retry-After", "1")
	writeError(w, status, code, http.StatusText(status), nil)

	return true
}
