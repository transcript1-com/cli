package api

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

const loginRequestReply = `{"request_id":"req-1","user_code":"WXYZ-2345","poll_secret":"t1l_x",` +
	`"verification_url":"https://transcript1.com/cli/login?code=WXYZ-2345","expires_in":600,"interval":2}`

func TestCreateLoginRequest(t *testing.T) {
	t.Run("posts the machine details without credentials", func(t *testing.T) {
		var got *http.Request
		var body string
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
			writeJSON(w, 201, loginRequestReply)
		})

		lr, err := c.CreateLoginRequest(context.Background(), "mba", "darwin", "1.1.0")
		if err != nil {
			t.Fatal(err)
		}

		if got.Method != "POST" || got.URL.Path != "/cli/login-requests" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.Header.Get("Authorization") != "" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("Idempotency-Key") != "" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content type %q", got.Header.Get("Content-Type"))
		}

		if body != `{"hostname":"mba","os":"darwin","cli_version":"1.1.0"}` {
			t.Fatalf("body %s", body)
		}

		want := LoginRequest{
			RequestID:       "req-1",
			UserCode:        "WXYZ-2345",
			PollSecret:      "t1l_x",
			VerificationURL: "https://transcript1.com/cli/login?code=WXYZ-2345",
			ExpiresIn:       600,
			Interval:        2,
		}
		if *lr != want {
			t.Fatalf("got %+v", *lr)
		}
	})

	t.Run("does not retry a 429", func(t *testing.T) {
		var calls atomic.Int32
		c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			writeJSON(w, 429, `{"error":{"code":"rate_limit_exceeded","message":"Too many requests","retryable":true}}`)
		})

		_, err := c.CreateLoginRequest(context.Background(), "mba", "darwin", "1.1.0")
		if !IsCode(err, CodeRateLimitExceeded) {
			t.Fatalf("got %v", err)
		}

		if calls.Load() != 1 || len(s.slept) != 0 {
			t.Fatalf("calls %d, sleeps %v", calls.Load(), s.slept)
		}
	})
}

func TestPollLoginRequest(t *testing.T) {
	t.Run("posts the secret to the request path", func(t *testing.T) {
		var got *http.Request
		var body string
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
			writeJSON(w, 200, `{"status":"slow_down","interval":7}`)
		})

		poll, err := c.PollLoginRequest(context.Background(), "req-1", "t1l_x")
		if err != nil {
			t.Fatal(err)
		}

		if got.Method != "POST" || got.URL.Path != "/cli/login-requests/req-1/poll" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.Header.Get("Authorization") != "" || got.Header.Get("Idempotency-Key") != "" {
			t.Fatalf("headers %v", got.Header)
		}

		if got.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content type %q", got.Header.Get("Content-Type"))
		}

		if body != `{"poll_secret":"t1l_x"}` {
			t.Fatalf("body %s", body)
		}

		if poll.Status != LoginSlowDown || poll.Interval != 7 {
			t.Fatalf("got %+v", *poll)
		}
	})

	t.Run("decodes an approved reply", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 200, `{"status":"approved","api_key":"t1_new","token_id":"tok-1","email":"a@b.com"}`)
		})

		poll, err := c.PollLoginRequest(context.Background(), "req-1", "t1l_x")
		if err != nil {
			t.Fatal(err)
		}

		want := LoginPoll{
			Status:  LoginApproved,
			APIKey:  "t1_new",
			TokenID: "tok-1",
			Email:   "a@b.com",
		}
		if *poll != want {
			t.Fatalf("got %+v", *poll)
		}
	})

	t.Run("retries 503 replies", func(t *testing.T) {
		var calls atomic.Int32
		c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) < 3 {
				writeJSON(w, 503, `{"error":{"code":"service_unavailable","message":"Down","retryable":true}}`)
				return
			}

			writeJSON(w, 200, `{"status":"pending","interval":2}`)
		})

		poll, err := c.PollLoginRequest(context.Background(), "req-1", "t1l_x")
		if err != nil {
			t.Fatal(err)
		}

		if poll.Status != LoginPending || len(s.slept) != 2 {
			t.Fatalf("status %q, sleeps %v", poll.Status, s.slept)
		}
	})

	t.Run("does not retry a 429", func(t *testing.T) {
		var calls atomic.Int32
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			writeJSON(w, 429, `{"error":{"code":"rate_limit_exceeded","message":"Too many requests","retryable":true}}`)
		})

		_, err := c.PollLoginRequest(context.Background(), "req-1", "t1l_x")
		if !IsCode(err, CodeRateLimitExceeded) || calls.Load() != 1 {
			t.Fatalf("got %v after %d calls", err, calls.Load())
		}
	})

	t.Run("does not retry an expired request", func(t *testing.T) {
		var calls atomic.Int32
		c, s := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			writeJSON(w, 410, `{"error":{"code":"cli_login_expired","message":"Expired","retryable":false}}`)
		})

		_, err := c.PollLoginRequest(context.Background(), "req-1", "t1l_x")
		if !IsCode(err, CodeCliLoginExpired) {
			t.Fatalf("got %v", err)
		}

		if calls.Load() != 1 || len(s.slept) != 0 {
			t.Fatalf("calls %d, sleeps %v", calls.Load(), s.slept)
		}
	})

	t.Run("escapes the request id", func(t *testing.T) {
		var got string
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r.URL.EscapedPath()
			writeJSON(w, 200, `{"status":"pending","interval":2}`)
		})

		if _, err := c.PollLoginRequest(context.Background(), "a/b", "t1l_x"); err != nil {
			t.Fatal(err)
		}

		if got != "/cli/login-requests/a%2Fb/poll" {
			t.Fatalf("path %s", got)
		}
	})
}

func TestCancelLoginRequest(t *testing.T) {
	var got *http.Request
	var body string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeJSON(w, 200, `{"status":"canceled"}`)
	})

	if err := c.CancelLoginRequest(context.Background(), "req-1", "t1l_x"); err != nil {
		t.Fatal(err)
	}

	if got.Method != "POST" || got.URL.Path != "/cli/login-requests/req-1/cancel" {
		t.Fatalf("got %s %s", got.Method, got.URL.Path)
	}

	if got.Header.Get("Authorization") != "" || got.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("headers %v", got.Header)
	}

	if body != `{"poll_secret":"t1l_x"}` {
		t.Fatalf("body %s", body)
	}
}

func TestRevokeToken(t *testing.T) {
	t.Run("sends DELETE with the bearer key", func(t *testing.T) {
		var got *http.Request
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r
			writeJSON(w, 200, `{}`)
		})

		if err := c.RevokeToken(context.Background(), "tok-1"); err != nil {
			t.Fatal(err)
		}

		if got.Method != "DELETE" || got.URL.Path != "/user/tokens/tok-1" {
			t.Fatalf("got %s %s", got.Method, got.URL.Path)
		}

		if got.Header.Get("Authorization") != "Bearer t1_x" {
			t.Fatalf("headers %v", got.Header)
		}
	})

	t.Run("returns pat_not_found on a 404", func(t *testing.T) {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, 404, `{"error":{"code":"pat_not_found","message":"No such token","retryable":false}}`)
		})

		if err := c.RevokeToken(context.Background(), "tok-1"); !IsCode(err, CodePatNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}
