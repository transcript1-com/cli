package fakeapi

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
)

const (
	msgNotFound    = "Transcription not found"
	msgNotAwaiting = "Transcription is not awaiting an upload"
)

type Server struct {
	URL string

	Key             string
	ChunkSize       int64
	Polls           int
	ActiveLimit     int
	MinutesLeft     int
	MinutesPerJob   int
	ChunkStatus     map[int][]int
	CompleteMissing []int
	CompleteDrop    bool
	FailCode        string
	Titles          map[string]string
	BlockChunks     chan struct{}
	PollErrors      int
	PollErrorStatus int
	PollDrop        bool

	LoginInterval     int
	LoginExpiresIn    int
	LoginPending      int
	LoginSlowDowns    int
	LoginPollErrors   int
	LoginOutcome      string
	LoginCreateStatus int
	LoginTokenID      string
	LoginKey          string
	RevokeStatus      int

	mu              sync.Mutex
	pollErrors      int
	srv             *httptest.Server
	closing         chan struct{}
	jobs            map[string]*job
	order           []*job
	nextID          int
	nextToken       int
	requests        []*Request
	chunkCalls      map[int]int
	completes       int
	inFlight        int
	peakChunks      int
	peakActive      int
	completeDropped bool
	loginErrors     int
	loginPendings   int
	loginSlowDowns  int
	loginRaised     int
}

func New(t testing.TB) *Server {
	t.Helper()
	s := &Server{
		Key:            "t1_test",
		ChunkSize:      8,
		Polls:          1,
		MinutesLeft:    -1,
		MinutesPerJob:  1,
		LoginInterval:  2,
		LoginExpiresIn: 600,
		LoginPending:   1,
		LoginTokenID:   "tok-1",
		closing:        make(chan struct{}),
		jobs:           map[string]*job{},
		chunkCalls:     map[int]int{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/transcriptions", s.requireAuth(s.create))
	mux.HandleFunc("GET /v1/transcriptions/{id}", s.requireAuth(s.get))
	mux.HandleFunc("DELETE /v1/transcriptions/{id}", s.requireAuth(s.remove))
	mux.HandleFunc("PUT /v1/transcriptions/{id}/upload/chunks/{index}", s.putChunk)
	mux.HandleFunc("GET /v1/transcriptions/{id}/upload", s.requireAuth(s.getProgress))
	mux.HandleFunc("POST /v1/transcriptions/{id}/upload/complete", s.requireAuth(s.complete))
	mux.HandleFunc("GET /v1/account", s.requireAuth(s.getAccount))
	mux.HandleFunc("POST /cli/login-requests", s.createLogin)
	mux.HandleFunc("POST /cli/login-requests/{id}/poll", s.pollLogin)
	mux.HandleFunc("POST /cli/login-requests/{id}/cancel", s.cancelLogin)
	mux.HandleFunc("DELETE /user/tokens/{tokenId}", s.requireAuth(s.revokeToken))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, api.CodeNotFound, "Route not found", nil)
	})

	s.srv = httptest.NewServer(s.record(mux))
	s.URL = s.srv.URL
	t.Cleanup(func() {
		close(s.closing)
		s.srv.Close()
	})

	return s
}

func (s *Server) requireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		key := s.Key
		loginKey := s.LoginKey
		s.mu.Unlock()
		auth := r.Header.Get("Authorization")
		if loginKey != "" && auth == "Bearer "+loginKey {
			h(w, r)

			return
		}

		if auth != "Bearer "+key {
			w.Header().Set("WWW-Authenticate", `Bearer realm="transcript1"`)
			writeError(w, http.StatusUnauthorized, api.CodeUnauthorized, "Invalid or missing API key", nil)

			return
		}

		h(w, r)
	}
}

// read the body first so a canceled client is noticed
func (s *Server) record(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		req := s.begin(r, body)
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		h.ServeHTTP(sw, r)
		s.finish(req, sw.status)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
