package fakeapi

import (
	"io"
	"net/http"
	"strconv"

	"github.com/transcript1/cli/internal/api"
)

const statusClientClosed = 499

var statusCodes = map[int]string{
	http.StatusUnauthorized:        api.CodeUnauthorized,
	http.StatusNotFound:            api.CodeNotFound,
	http.StatusConflict:            api.CodeInvalidState,
	http.StatusUnprocessableEntity: api.CodeInvalidRequest,
	http.StatusTooManyRequests:     api.CodeRateLimited,
	http.StatusServiceUnavailable:  api.CodeServiceUnavailable,
}

func hasValidToken(r *http.Request, j *job) bool {
	token := r.Header.Get("X-Upload-Token")
	if token == "" {
		return false
	}

	return token == j.token
}

func isUploadClosed(j *job) bool {
	if j.status == api.StatusAwaitingUpload {
		return false
	}

	return j.token == ""
}

func isPlannedChunk(j *job, index int, indexErr error, body []byte) bool {
	if indexErr != nil {
		return false
	}

	if index < 0 {
		return false
	}

	if index >= len(j.chunks) {
		return false
	}

	return int64(len(body)) == j.chunks[index].Size
}

func (s *Server) putChunk(w http.ResponseWriter, r *http.Request) {
	block := s.enterChunk()
	defer s.leaveChunk()
	if block != nil {
		select {
		case <-block:
		case <-r.Context().Done():
			w.WriteHeader(statusClientClosed)
			return
		case <-s.closing:
			w.WriteHeader(statusClientClosed)
			return
		}
	}

	body, _ := io.ReadAll(r.Body)
	index, indexErr := strconv.Atoi(r.PathValue("index"))
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writeScripted(w, index) {
		return
	}

	j := s.jobs[r.PathValue("id")]
	if j == nil {
		writeError(w, http.StatusNotFound, api.CodeNotFound, msgNotFound, nil)
		return
	}

	if !hasValidToken(r, j) {
		if isUploadClosed(j) {
			writeError(w, http.StatusConflict, api.CodeInvalidState, msgNotAwaiting, nil)
			return
		}

		writeError(w, http.StatusUnauthorized, api.CodeUnauthorized, "Invalid or expired upload token", nil)

		return
	}

	if j.status != api.StatusAwaitingUpload {
		writeError(w, http.StatusConflict, api.CodeInvalidState, msgNotAwaiting, nil)
		return
	}

	if !isPlannedChunk(j, index, indexErr, body) {
		message := "Chunk is outside the plan or has the wrong size"
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidRequest, message, nil)

		return
	}

	if j.data == nil {
		j.data = make([]byte, j.size)
	}

	copy(j.data[j.chunks[index].Offset:], body)
	j.received[index] = true
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "upload_chunk",
		"index":  index,
		"size":   len(body),
	})
}

func (s *Server) writeScripted(w http.ResponseWriter, index int) bool {
	n := s.chunkCalls[index]
	s.chunkCalls[index]++
	statuses := s.ChunkStatus[index]
	if n >= len(statuses) {
		return false
	}

	status := statuses[n]
	code, ok := statusCodes[status]
	if !ok {
		code = api.CodeInternalError
	}

	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "1")
	}

	writeError(w, status, code, "Scripted answer "+strconv.Itoa(status), nil)

	return true
}

func (s *Server) enterChunk() chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight++
	s.peakChunks = max(s.peakChunks, s.inFlight)

	return s.BlockChunks
}

func (s *Server) leaveChunk() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inFlight--
}
