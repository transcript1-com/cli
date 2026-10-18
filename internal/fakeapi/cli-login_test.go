package fakeapi

import (
	"fmt"
	"testing"
)

const pollBody = `{"poll_secret":"t1l_test"}`

func pollLogin(t *testing.T, s *Server) reply {
	t.Helper()

	return call(t, "POST", s.URL+"/cli/login-requests/req-1/poll", "", nil, []byte(pollBody))
}

func getStatuses(t *testing.T, s *Server, count int) []string {
	t.Helper()
	var out []string
	for range count {
		r := pollLogin(t, s)
		if r.status != 200 {
			out = append(out, fmt.Sprintf("%d %v", r.status, getError(r)["code"]))
			continue
		}

		out = append(out, fmt.Sprintf("%v %v", r.body["status"], r.body["interval"]))
	}

	return out
}

func assertStatuses(t *testing.T, got []string, want ...string) {
	t.Helper()
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoginCreate(t *testing.T) {
	t.Run("creates a request without credentials", func(t *testing.T) {
		s := New(t)
		r := call(t, "POST", s.URL+"/cli/login-requests", "", nil, []byte(`{}`))
		if r.status != 201 {
			t.Fatalf("status %d", r.status)
		}

		if r.body["request_id"] != "req-1" || r.body["user_code"] != "WXYZ-2345" || r.body["poll_secret"] != "t1l_test" {
			t.Fatalf("body %v", r.body)
		}

		if r.body["verification_url"] != "https://transcript1.com/cli/login?code=WXYZ-2345" {
			t.Fatalf("body %v", r.body)
		}

		if r.body["expires_in"] != float64(600) || r.body["interval"] != float64(2) {
			t.Fatalf("body %v", r.body)
		}
	})

	t.Run("answers 429 when asked to", func(t *testing.T) {
		s := New(t)
		s.LoginCreateStatus = 429
		r := call(t, "POST", s.URL+"/cli/login-requests", "", nil, []byte(`{}`))
		if r.status != 429 || getError(r)["code"] != "rate_limit_exceeded" {
			t.Fatalf("status %d body %v", r.status, r.body)
		}
	})
}

func TestLoginPollOrder(t *testing.T) {
	t.Run("approves after one pending reply", func(t *testing.T) {
		s := New(t)
		got := getStatuses(t, s, 2)
		assertStatuses(t, got, "pending 2", "approved <nil>")
		r := pollLogin(t, s)
		if r.body["api_key"] != "t1_test" || r.body["token_id"] != "tok-1" || r.body["email"] != "demo@transcript1.com" {
			t.Fatalf("body %v", r.body)
		}
	})

	t.Run("replies pending, then slow_down, then the outcome", func(t *testing.T) {
		s := New(t)
		s.LoginSlowDowns = 1
		assertStatuses(t, getStatuses(t, s, 3), "pending 2", "slow_down 7", "approved <nil>")
	})

	t.Run("replies 503 before anything else", func(t *testing.T) {
		s := New(t)
		s.LoginPollErrors = 2
		assertStatuses(t, getStatuses(t, s, 4), "503 service_unavailable", "503 service_unavailable", "pending 2", "approved <nil>")
	})

	t.Run("keeps the raised interval after a slow_down", func(t *testing.T) {
		s := New(t)
		s.LoginPending = 2
		s.LoginSlowDowns = 1
		s.LoginOutcome = "pending"
		assertStatuses(t, getStatuses(t, s, 4), "pending 2", "pending 2", "slow_down 7", "pending 7")
	})
}

func TestLoginOutcomes(t *testing.T) {
	cases := map[string]string{
		"denied":    "403 cli_login_denied",
		"expired":   "410 cli_login_expired",
		"consumed":  "410 cli_login_consumed",
		"not_found": "404 cli_login_not_found",
		"pat_limit": "409 pat_limit_reached",
	}
	for outcome, want := range cases {
		t.Run(outcome, func(t *testing.T) {
			s := New(t)
			s.LoginPending = 0
			s.LoginOutcome = outcome
			assertStatuses(t, getStatuses(t, s, 1), want)
		})
	}

	t.Run("rejects a wrong secret", func(t *testing.T) {
		s := New(t)
		r := call(t, "POST", s.URL+"/cli/login-requests/req-1/poll", "", nil, []byte(`{"poll_secret":"nope"}`))
		if r.status != 404 || getError(r)["code"] != "cli_login_not_found" {
			t.Fatalf("status %d body %v", r.status, r.body)
		}
	})
}

func TestLoginCancelAndRevoke(t *testing.T) {
	t.Run("cancels with the secret", func(t *testing.T) {
		s := New(t)
		r := call(t, "POST", s.URL+"/cli/login-requests/req-1/cancel", "", nil, []byte(pollBody))
		if r.status != 200 || r.body["status"] != "canceled" {
			t.Fatalf("status %d body %v", r.status, r.body)
		}

		r = call(t, "POST", s.URL+"/cli/login-requests/req-1/cancel", "", nil, []byte(`{"poll_secret":"nope"}`))
		if r.status != 404 {
			t.Fatalf("status %d", r.status)
		}
	})

	t.Run("revokes behind auth", func(t *testing.T) {
		s := New(t)
		r := call(t, "DELETE", s.URL+"/user/tokens/tok-1", "", nil, nil)
		if r.status != 401 {
			t.Fatalf("status %d", r.status)
		}

		r = call(t, "DELETE", s.URL+"/user/tokens/tok-1", s.Key, nil, nil)
		if r.status != 200 {
			t.Fatalf("status %d", r.status)
		}
	})

	t.Run("answers the configured revoke status", func(t *testing.T) {
		want := map[int]string{404: "pat_not_found", 403: "auth_invalid_session", 503: "service_unavailable"}
		for status, code := range want {
			s := New(t)
			s.RevokeStatus = status
			r := call(t, "DELETE", s.URL+"/user/tokens/tok-1", s.Key, nil, nil)
			if r.status != status || getError(r)["code"] != code {
				t.Fatalf("status %d body %v", r.status, r.body)
			}
		}
	})
}
