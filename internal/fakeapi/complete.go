package fakeapi

import (
	"net/http"
	"time"

	"github.com/transcript1-com/cli/internal/api"
)

func (s *Server) findOwned(w http.ResponseWriter, r *http.Request) *job {
	j := s.jobs[r.PathValue("id")]
	if j == nil {
		writeError(w, http.StatusNotFound, api.CodeNotFound, msgNotFound, nil)
	}

	return j
}

func (s *Server) getProgress(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.findOwned(w, r)
	if j == nil {
		return
	}

	if j.platform != "upload" {
		writeError(w, http.StatusNotFound, api.CodeNotFound, "Upload not found", nil)
		return
	}

	var uploaded int64
	for _, c := range j.chunks {
		if j.received[c.Index] {
			uploaded += c.Size
		}
	}

	var expires any
	if j.token != "" {
		expires = time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"object":          "upload",
		"status":          j.status,
		"total_bytes":     j.size,
		"uploaded_bytes":  uploaded,
		"received_chunks": append([]int{}, j.listReceived()...),
		"chunks":          j.chunks,
		"expires_at":      expires,
	})
}

func (s *Server) shouldDropComplete() bool {
	if !s.CompleteDrop {
		return false
	}

	return !s.completeDropped
}

func (s *Server) complete(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.findOwned(w, r)
	if j == nil {
		return
	}

	if j.status != api.StatusAwaitingUpload {
		writeError(w, http.StatusConflict, api.CodeInvalidState, msgNotAwaiting, nil)
		return
	}

	s.completes++
	if s.completes == 1 {
		for _, index := range s.CompleteMissing {
			delete(j.received, index)
		}
	}

	if missing := j.listMissing(); len(missing) > 0 {
		extra := map[string]any{"missing_chunks": missing}
		writeError(w, http.StatusConflict, api.CodeUploadIncomplete, "Missing chunks", extra)

		return
	}

	if s.refuseStart(w) {
		return
	}

	j.status = api.StatusQueued
	j.token = ""
	s.charge(j)
	s.recordPeak()
	if s.shouldDropComplete() {
		s.completeDropped = true
		panic(http.ErrAbortHandler)
	}

	writeJSON(w, http.StatusAccepted, s.serialize(j, false))
}
