package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type sleeper struct {
	mu    sync.Mutex
	slept []time.Duration
}

func (s *sleeper) sleep(_ context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slept = append(s.slept, d)

	return nil
}

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *sleeper) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	s := &sleeper{}
	c := New(srv.URL, "t1_x")
	c.UserAgent = "transcript1-cli/test (linux/amd64)"
	c.Sleep = s.sleep

	return c, s
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func getAPIError(t *testing.T, err error) *Error {
	t.Helper()
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *Error, got %T: %v", err, err)
	}

	return apiErr
}

func TestErrorEnvelope(t *testing.T) {
	body := `{"error":{"code":"insufficient_minutes","message":"Not enough minutes.","retryable":false,` +
		`"minutes_needed":20,"minutes_remaining":3,"action_url":"https://transcript1.com/dashboard/billing"}}`
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusPaymentRequired, body)
	})

	_, err := c.Account(context.Background())
	apiErr := getAPIError(t, err)

	t.Run("reads status, code and message", func(t *testing.T) {
		if apiErr.Status != 402 {
			t.Fatalf("got %+v", apiErr)
		}

		if apiErr.Code != CodeInsufficientMinutes {
			t.Fatalf("got %+v", apiErr)
		}

		if apiErr.Retryable {
			t.Fatalf("got %+v", apiErr)
		}

		if apiErr.Error() != "insufficient_minutes: Not enough minutes" {
			t.Fatalf("got %q", apiErr.Error())
		}
	})

	t.Run("keeps the extra fields as details", func(t *testing.T) {
		if apiErr.Details["minutes_needed"] != 20.0 {
			t.Fatalf("got %v", apiErr.Details)
		}

		if apiErr.Details["minutes_remaining"] != 3.0 {
			t.Fatalf("got %v", apiErr.Details)
		}

		if apiErr.Details["action_url"] != "https://transcript1.com/dashboard/billing" {
			t.Fatalf("got %v", apiErr.Details)
		}

		if _, ok := apiErr.Details["code"]; ok {
			t.Fatalf("code leaked into details: %v", apiErr.Details)
		}
	})

	t.Run("IsCode matches a listed code", func(t *testing.T) {
		if !IsCode(err, CodeNoActivePlan, CodeInsufficientMinutes) {
			t.Fatal("IsCode mismatch")
		}

		if IsCode(err, CodeNotFound) {
			t.Fatal("IsCode mismatch")
		}

		if IsCode(errors.New("boom"), CodeNotFound) {
			t.Fatal("IsCode must be false for other errors")
		}

		if IsCode(nil, CodeNotFound) {
			t.Fatal("IsCode must be false for other errors")
		}
	})
}

func TestNonJSONError(t *testing.T) {
	t.Run("an HTML 502 is retried, then reported with the HTTP status", func(t *testing.T) {
		var calls atomic.Int32
		c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>Bad gateway</html>"))
		})

		_, err := c.Get(context.Background(), "tr_1", 0)
		apiErr := getAPIError(t, err)
		if calls.Load() != 3 {
			t.Fatalf("calls %d, sleeps %v", calls.Load(), s.slept)
		}

		if len(s.slept) != 2 {
			t.Fatalf("calls %d, sleeps %v", calls.Load(), s.slept)
		}

		if apiErr.Status != 502 {
			t.Fatalf("got %+v", apiErr)
		}

		if apiErr.Code != "http_502" {
			t.Fatalf("got %+v", apiErr)
		}

		if !apiErr.Retryable {
			t.Fatalf("got %+v", apiErr)
		}

		if apiErr.Error() != "http_502: the server answered HTTP 502" {
			t.Fatalf("got %q", apiErr.Error())
		}
	})

	t.Run("an empty 503 gets the same treatment", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		_, err := c.Account(context.Background())
		apiErr := getAPIError(t, err)
		if apiErr.Code != "http_503" {
			t.Fatalf("got %+v", apiErr)
		}

		if apiErr.Message != "the server answered HTTP 503" {
			t.Fatalf("got %+v", apiErr)
		}
	})

	t.Run("a plain 404 is not retryable", func(t *testing.T) {
		var calls atomic.Int32
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.NotFound(w, r)
		})

		_, err := c.Account(context.Background())
		apiErr := getAPIError(t, err)
		if calls.Load() != 1 {
			t.Fatalf("calls %d, got %+v", calls.Load(), apiErr)
		}

		if apiErr.Retryable {
			t.Fatalf("calls %d, got %+v", calls.Load(), apiErr)
		}

		if apiErr.Code != "http_404" {
			t.Fatalf("calls %d, got %+v", calls.Load(), apiErr)
		}
	})
}

func TestRetriesWithBackoff(t *testing.T) {
	var calls atomic.Int32
	c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			writeJSON(w, 503, `{"error":{"code":"service_unavailable","message":"Try again","retryable":true}}`)
			return
		}

		writeJSON(w, 200, `{"object":"account","user":{"email":"a@b.c"}}`)
	})

	acc, err := c.Account(context.Background())
	if err != nil {
		t.Fatalf("got %+v, %v", acc, err)
	}

	if acc.User.Email != "a@b.c" {
		t.Fatalf("got %+v, %v", acc, err)
	}

	if len(s.slept) != 2 {
		t.Fatalf("sleeps %v", s.slept)
	}

	if s.slept[0] < time.Second {
		t.Fatalf("first wait %v", s.slept[0])
	}

	if s.slept[0] >= 1250*time.Millisecond {
		t.Fatalf("first wait %v", s.slept[0])
	}

	if s.slept[1] < 2*time.Second {
		t.Fatalf("second wait %v", s.slept[1])
	}

	if s.slept[1] >= 2500*time.Millisecond {
		t.Fatalf("second wait %v", s.slept[1])
	}
}

func TestRetryAfterHonored(t *testing.T) {
	var calls atomic.Int32
	c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "7")
			writeJSON(w, 429, `{"error":{"code":"rate_limited","message":"Slow down","retryable":true}}`)

			return
		}

		writeJSON(w, 200, `{"id":"tr_1","status":"processing"}`)
	})

	tr, err := c.Get(context.Background(), "tr_1", 0)
	if err != nil {
		t.Fatalf("got %+v, %v", tr, err)
	}

	if tr.ID != "tr_1" {
		t.Fatalf("got %+v, %v", tr, err)
	}

	if len(s.slept) != 1 {
		t.Fatalf("sleeps %v", s.slept)
	}

	if s.slept[0] != 7*time.Second {
		t.Fatalf("sleeps %v", s.slept)
	}
}

func TestRetryAfterCapped(t *testing.T) {
	var calls atomic.Int32
	c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "3600")
			writeJSON(w, 503, `{"error":{"code":"service_unavailable","message":"Down","retryable":true}}`)

			return
		}

		writeJSON(w, 200, `{"id":"tr_1","status":"processing"}`)
	})

	if _, err := c.Get(context.Background(), "tr_1", 0); err != nil {
		t.Fatal(err)
	}

	t.Run("sleeps at most 60 seconds between attempts", func(t *testing.T) {
		if len(s.slept) != 1 {
			t.Fatalf("sleeps %v", s.slept)
		}

		if s.slept[0] != 60*time.Second {
			t.Fatalf("sleeps %v", s.slept)
		}
	})

	t.Run("keeps the full value on the error for job-slot waits", func(t *testing.T) {
		res := &response{
			Status: 429,
			Header: http.Header{"Retry-After": {"3600"}},
			Body:   []byte(`{}`),
		}
		if got := buildResponseError(res).RetryAfter; got != time.Hour {
			t.Fatalf("retry after %v", got)
		}
	})
}

func TestNetworkErrorRetried(t *testing.T) {
	var calls atomic.Int32
	c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close()
			}

			return
		}

		writeJSON(w, 200, `{"id":"tr_1","status":"completed"}`)
	})

	tr, err := c.Get(context.Background(), "tr_1", 0)
	if err != nil {
		t.Fatalf("got %+v, %v", tr, err)
	}

	if tr.Status != StatusCompleted {
		t.Fatalf("got %+v, %v", tr, err)
	}

	if calls.Load() != 2 {
		t.Fatalf("calls %d, sleeps %v", calls.Load(), s.slept)
	}

	if len(s.slept) != 1 {
		t.Fatalf("calls %d, sleeps %v", calls.Load(), s.slept)
	}
}

func TestStopsWhenSleepIsCanceled(t *testing.T) {
	var calls atomic.Int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	c.Sleep = func(context.Context, time.Duration) error { return context.Canceled }
	_, err := c.Account(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("calls %d, err %v", calls.Load(), err)
	}

	if calls.Load() != 1 {
		t.Fatalf("calls %d, err %v", calls.Load(), err)
	}
}
