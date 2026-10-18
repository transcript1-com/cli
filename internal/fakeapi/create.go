package fakeapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

const billingURL = "https://transcript1.com/dashboard/billing"

type createBody struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Source   string `json:"source"`
	Language string `json:"language"`
	Diarize  bool   `json:"diarize"`
	Upload   *struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
		MimeType string `json:"mime_type"`
	} `json:"upload"`
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var body createBody
	raw, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, "Body must be JSON", nil)
		return
	}

	if body.Source == "" {
		body.Source = "auto"
	}

	if body.Language == "" {
		body.Language = "auto"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := r.Header.Get("Idempotency-Key")
	if body.Upload != nil {
		s.createUpload(w, key, body)
		return
	}

	s.createLink(w, key, r.URL.Query().Get("reuse_recent") == "true", body)
}

func (s *Server) replay(w http.ResponseWriter, key, params string) bool {
	j := s.findByKey(key)
	if j == nil {
		return false
	}

	if j.params != params {
		message := "Idempotency-Key was reused with different parameters"
		writeError(w, http.StatusConflict, api.CodeIdempotencyConflict, message, nil)

		return true
	}

	if j.status == api.StatusAwaitingUpload {
		s.issueToken(j)
	}

	s.writeCreated(w, j, false)

	return true
}

func (s *Server) refuseStart(w http.ResponseWriter) bool {
	if s.isOverActiveLimit() {
		w.Header().Set("Retry-After", "1")
		message := fmt.Sprintf("Too many active transcriptions (limit %d)", s.ActiveLimit)
		writeError(w, http.StatusTooManyRequests, api.CodeTooManyActiveJobs, message, nil)

		return true
	}

	return s.refuseMinutes(w)
}

func (s *Server) refuseMinutes(w http.ResponseWriter) bool {
	if !s.isMinutesShort() {
		return false
	}

	writeError(w, http.StatusPaymentRequired, api.CodeInsufficientMinutes, "Not enough minutes", map[string]any{
		"action_url":        billingURL,
		"minutes_needed":    s.MinutesPerJob,
		"minutes_remaining": max(s.MinutesLeft, 0),
	})

	return true
}

func (s *Server) writeCreated(w http.ResponseWriter, j *job, reused bool) {
	obj := s.serialize(j, false)
	obj["reused"] = reused
	status := http.StatusAccepted
	switch {
	case j.status == api.StatusAwaitingUpload:
		status = http.StatusCreated
		obj["upload"] = map[string]any{
			"chunk_url_template": "/v1/transcriptions/" + j.id + "/upload/chunks/{index}",
			"headers":            map[string]string{"X-Upload-Token": j.token},
			"expires_at":         time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			"chunks":             j.chunks,
		}
	case isTerminal(j.status):
		status = http.StatusOK
	}

	writeJSON(w, status, obj)
}
