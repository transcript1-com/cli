package fakeapi

import (
	"encoding/json"
	"net/http"

	"github.com/transcript1-com/cli/internal/api"
)

const (
	loginSecret   = "t1l_test"
	slowDownExtra = 5
)

type loginOutcomeType struct {
	status  int
	code    string
	message string
}

var loginOutcomes = map[string]loginOutcomeType{
	"denied":    {http.StatusForbidden, api.CodeCliLoginDenied, "Login request was denied"},
	"expired":   {http.StatusGone, api.CodeCliLoginExpired, "Login request expired"},
	"consumed":  {http.StatusGone, api.CodeCliLoginConsumed, "Login request was already used"},
	"not_found": {http.StatusNotFound, api.CodeCliLoginNotFound, "Login request not found"},
	"pat_limit": {http.StatusConflict, api.CodePatLimitReached, "Too many active API keys"},
}

func (s *Server) createLogin(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.LoginCreateStatus == http.StatusTooManyRequests {
		writeError(w, http.StatusTooManyRequests, api.CodeRateLimitExceeded, "Too many login requests", nil)

		return
	}

	if s.LoginCreateStatus != 0 {
		writeError(w, s.LoginCreateStatus, api.CodeServiceUnavailable, http.StatusText(s.LoginCreateStatus), nil)

		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"request_id":       "req-1",
		"user_code":        "WXYZ-2345",
		"poll_secret":      loginSecret,
		"verification_url": "https://transcript1.com/cli/login?code=WXYZ-2345",
		"expires_in":       s.LoginExpiresIn,
		"interval":         s.LoginInterval,
	})
}

func hasLoginSecret(r *http.Request) bool {
	var body struct {
		PollSecret string `json:"poll_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return false
	}

	return body.PollSecret == loginSecret
}

func writeLoginNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, api.CodeCliLoginNotFound, "Login request not found", nil)
}

// replies come in a fixed order: 503s, pending, slow_down, then the outcome
func (s *Server) pollLogin(w http.ResponseWriter, r *http.Request) {
	if !hasLoginSecret(r) {
		writeLoginNotFound(w)

		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loginErrors < s.LoginPollErrors {
		s.loginErrors++
		writeError(w, http.StatusServiceUnavailable, api.CodeServiceUnavailable, "Service unavailable", nil)

		return
	}

	if s.loginPendings < s.LoginPending {
		s.loginPendings++
		s.writePending(w, api.LoginPending)

		return
	}

	if s.loginSlowDowns < s.LoginSlowDowns {
		s.loginSlowDowns++
		s.loginRaised += slowDownExtra
		s.writePending(w, api.LoginSlowDown)

		return
	}

	s.writeLoginOutcome(w)
}

func (s *Server) writePending(w http.ResponseWriter, status string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   status,
		"interval": s.LoginInterval + s.loginRaised,
	})
}

func (s *Server) writeLoginOutcome(w http.ResponseWriter) {
	if s.LoginOutcome == api.LoginPending {
		s.writePending(w, api.LoginPending)

		return
	}

	if outcome, ok := loginOutcomes[s.LoginOutcome]; ok {
		writeError(w, outcome.status, outcome.code, outcome.message, nil)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   api.LoginApproved,
		"api_key":  s.Key,
		"token_id": s.LoginTokenID,
		"email":    "demo@transcript1.com",
	})
}

func (s *Server) cancelLogin(w http.ResponseWriter, r *http.Request) {
	if !hasLoginSecret(r) {
		writeLoginNotFound(w)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "canceled"})
}

func (s *Server) revokeToken(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	status := s.RevokeStatus
	s.mu.Unlock()
	switch status {
	case 0:
		writeJSON(w, http.StatusOK, map[string]any{})
	case http.StatusNotFound:
		writeError(w, status, api.CodePatNotFound, "API key not found", nil)
	case http.StatusForbidden:
		writeError(w, status, api.CodeAuthInvalidSession, "Session required", nil)
	default:
		writeError(w, status, api.CodeServiceUnavailable, http.StatusText(status), nil)
	}
}
