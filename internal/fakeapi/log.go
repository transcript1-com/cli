package fakeapi

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

type Request struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
	Chunk  int
	Status int
}

var chunkPath = regexp.MustCompile(`^/v1/transcriptions/[^/]+/upload/chunks/(\d+)$`)

func findChunkIndex(r *http.Request) int {
	if r.Method != http.MethodPut {
		return -1
	}

	m := chunkPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return -1
	}

	index, _ := strconv.Atoi(m[1])

	return index
}

func (s *Server) begin(r *http.Request, body []byte) *Request {
	req := &Request{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
		Body:   body,
		Chunk:  findChunkIndex(r),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, req)

	return req
}

func (s *Server) finish(req *Request, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req.Status = status
}

func (s *Server) Log() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	lines := make([]string, len(s.requests))
	for i, req := range s.requests {
		lines[i] = req.Method + " " + req.Path
		if req.Chunk >= 0 {
			lines[i] = fmt.Sprintf("PUT chunk %d", req.Chunk)
		}
	}

	return lines
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, len(s.requests))
	for i, req := range s.requests {
		out[i] = *req
	}

	return out
}

func (s *Server) PeakChunks() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.peakChunks
}

func (s *Server) PeakActive() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.peakActive
}

func (s *Server) Uploaded(id string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := s.jobs[id]
	if j == nil {
		return nil
	}

	return append([]byte(nil), j.data...)
}
