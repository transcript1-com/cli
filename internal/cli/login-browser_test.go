package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/transcript1-com/cli/internal/config"
)

const (
	verifyURL     = "https://transcript1.com/cli/login?code=WXYZ-2345"
	openLine      = "Open " + verifyURL + " and confirm the code WXYZ-2345.\n"
	waitingLine   = "Waiting for approval (Ctrl-C to cancel)...\n"
	loggedInLine  = "Logged in as demo@transcript1.com. Monthly plan, 1,000 minutes left.\n"
	createPath    = "POST /cli/login-requests"
	pollPath      = "POST /cli/login-requests/req-1/poll"
	cancelPath    = "POST /cli/login-requests/req-1/cancel"
	accountPath   = "GET /v1/account"
	revokeOldPath = "DELETE /user/tokens/tok-1"
)

func loadSavedFile(t *testing.T) config.KeyFile {
	t.Helper()
	file, err := config.LoadKeyFile()
	if err != nil {
		t.Fatal(err)
	}

	return file
}

func expectSaved(t *testing.T, want config.KeyFile) {
	t.Helper()
	if got := loadSavedFile(t); got != want {
		t.Fatalf("saved %+v, want %+v", got, want)
	}
}

func expectExit(t *testing.T, got outcome, code int) {
	t.Helper()
	if got.code != code {
		t.Fatalf("code %d, want %d, stdout %q stderr %q", got.code, code, got.stdout, got.stderr)
	}
}

func expectLog(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("log %v, want %v", got, want)
	}
}

func (h *harness) writeSavedFile(file config.KeyFile) {
	h.t.Helper()
	path, err := config.KeyPath()
	if err != nil {
		h.t.Fatal(err)
	}

	body, err := json.Marshal(file)
	if err != nil {
		h.t.Fatal(err)
	}

	h.write(path, string(body))
}

func TestBrowserLoginFlow(t *testing.T) {
	t.Run("logs in without a terminal or a browser", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("login")
		expectExit(t, got, 0)
		if got.stdout != loggedInLine {
			t.Fatalf("stdout %q", got.stdout)
		}

		if !strings.HasPrefix(got.stderr, openLine+waitingLine) {
			t.Fatalf("stderr %q", got.stderr)
		}

		if strings.ContainsAny(got.stderr, "\r\x1b") {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectLog(t, h.srv.Log(), createPath, pollPath, pollPath, accountPath)
		expectSaved(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
	})

	t.Run("names the machine in the request", func(t *testing.T) {
		h := newHarness(t)
		expectExit(t, h.run("login"), 0)
		var body struct {
			Hostname string `json:"hostname"`
			OS       string `json:"os"`
		}
		if err := json.Unmarshal(h.srv.Requests()[0].Body, &body); err != nil {
			t.Fatal(err)
		}

		hostname, err := os.Hostname()
		if err != nil {
			t.Fatal(err)
		}

		if body.Hostname != hostname || body.OS != runtime.GOOS {
			t.Fatalf("body %+v, want %q %q", body, hostname, runtime.GOOS)
		}
	})

	t.Run("warns when the environment key takes precedence", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("login")
		expectContains(t, got.stderr, "TRANSCRIPT1_API_KEY is set in this shell and takes precedence over the saved key.\n")
		delete(h.env, config.EnvAPIKey)
		got = h.run("login")
		if strings.Contains(got.stderr, "precedence") {
			t.Fatalf("stderr %q", got.stderr)
		}
	})
}

func TestBrowserLoginOpensBrowser(t *testing.T) {
	t.Run("opens the link and says so", func(t *testing.T) {
		h := newHarness(t)
		h.browser = true
		h.env["DISPLAY"] = ":0"
		got := h.run("login")
		expectExit(t, got, 0)
		if !slices.Equal(h.opened, []string{verifyURL}) {
			t.Fatalf("opened %v", h.opened)
		}

		expectContains(t, got.stderr, openLine+"Opened your browser.\n"+waitingLine)
	})

	t.Run("stays quiet when the browser fails", func(t *testing.T) {
		h := newHarness(t)
		h.browser = true
		h.openErr = fmt.Errorf("no browser")
		h.env["DISPLAY"] = ":0"
		got := h.run("login")
		expectExit(t, got, 0)
		if strings.Contains(got.stderr, "Opened your browser") {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectContains(t, got.stderr, openLine+waitingLine)
	})

	t.Run("skips the browser when asked or when remote", func(t *testing.T) {
		cases := []struct {
			name string
			args []string
			env  map[string]string
		}{
			{"flag", []string{"login", "--no-browser"}, map[string]string{"DISPLAY": ":0"}},
			{"ssh connection", []string{"login"}, map[string]string{"DISPLAY": ":0", "SSH_CONNECTION": "1"}},
			{"ssh tty", []string{"login"}, map[string]string{"DISPLAY": ":0", "SSH_TTY": "/dev/pts/0"}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				h := newHarness(t)
				h.browser = true
				for name, value := range c.env {
					h.env[name] = value
				}

				got := h.run(c.args...)
				expectExit(t, got, 0)
				if len(h.opened) != 0 {
					t.Fatalf("opened %v", h.opened)
				}

				expectContains(t, got.stderr, openLine)
			})
		}
	})

	t.Run("prints but never opens a link that is not http", func(t *testing.T) {
		for _, link := range []string{"javascript:alert(1)", "file:///etc/passwd", "/cli/login", ""} {
			h := newHarness(t)
			h.browser = true
			h.env["DISPLAY"] = ":0"
			stub := newLoginStub(t, h)
			stub.verificationURL = link
			got := h.run("login")
			if len(h.opened) != 0 {
				t.Fatalf("link %q opened %v", link, h.opened)
			}

			if link == "" {
				expectExit(t, got, 1)
				continue
			}

			expectExit(t, got, 0)
			expectContains(t, got.stderr, "Open "+link+" and confirm")
		}
	})
}

func TestBrowserLoginOnTerminal(t *testing.T) {
	t.Run("bolds the code and draws a spinner", func(t *testing.T) {
		h := newHarness(t)
		h.tty = true
		got := h.run("login")
		expectExit(t, got, 0)
		expectContains(t, got.stderr, "\x1b[1mWXYZ-2345\x1b[0m", "Waiting for approval", "\r\x1b[K")
		if strings.Contains(got.stderr, waitingLine) {
			t.Fatalf("stderr %q", got.stderr)
		}

		clear := strings.LastIndex(got.stderr, "\r\x1b[K")
		warn := strings.Index(got.stderr, "precedence")
		if clear > warn {
			t.Fatalf("spinner cleared after the next line: %q", got.stderr)
		}
	})

	t.Run("leaves the code plain with NO_COLOR", func(t *testing.T) {
		h := newHarness(t)
		h.tty = true
		h.env["NO_COLOR"] = "1"
		got := h.run("login")
		expectExit(t, got, 0)
		if strings.Contains(got.stderr, "\x1b[1m") {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectContains(t, got.stderr, "Waiting for approval")
	})

	t.Run("redraws the spinner while waiting", func(t *testing.T) {
		h := newHarness(t)
		h.tty = true
		h.onSleep = func() { time.Sleep(500 * time.Millisecond) }
		got := h.run("login")
		expectExit(t, got, 0)
		expectContains(t, got.stderr, "\r| Waiting", "\r/ Waiting")
	})
}

func TestBrowserLoginPolling(t *testing.T) {
	t.Run("follows the interval the server sets", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginSlowDowns = 1
		h.srv.LoginPending = 1
		expectExit(t, h.run("login"), 0)
		want := []time.Duration{2 * time.Second, 2 * time.Second, 7 * time.Second}
		if !slices.Equal(h.sleeps, want) {
			t.Fatalf("sleeps %v, want %v", h.sleeps, want)
		}
	})

	t.Run("falls back to two seconds without an interval", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginInterval = 0
		expectExit(t, h.run("login"), 0)
		for _, d := range h.sleeps {
			if d != 2*time.Second {
				t.Fatalf("sleeps %v", h.sleeps)
			}
		}
	})

	t.Run("keeps waiting through server errors", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginPollErrors = 4
		got := h.run("login")
		expectExit(t, got, 0)
		expectSaved(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
	})

	t.Run("gives up when the code expires", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginOutcome = "pending"
		h.srv.LoginExpiresIn = 6
		got := h.run("login")
		expectExit(t, got, 2)
		expectContains(t, got.stderr, "transcript1: The code expired. Run transcript1 login again.\n")
		expectSaved(t, config.KeyFile{})
	})

	t.Run("reports the last server error when the code expires", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginPollErrors = 1000
		h.srv.LoginExpiresIn = 6
		got := h.run("login")
		expectExit(t, got, 1)
		expectContains(t, got.stderr, "transcript1: login:")
		expectSaved(t, config.KeyFile{})
	})
}

func TestBrowserLoginFailures(t *testing.T) {
	cases := []struct {
		outcome string
		want    string
	}{
		{"denied", "Login was denied in the browser."},
		{"expired", "The code expired. Run transcript1 login again."},
		{"pat_limit", "You have 25 API keys. Revoke one at https://transcript1.com/dashboard/account/tokens, then run transcript1 login again."},
		{"consumed", "This login request is no longer valid. Run transcript1 login again."},
		{"not_found", "This login request is no longer valid. Run transcript1 login again."},
	}
	for _, c := range cases {
		t.Run(c.outcome, func(t *testing.T) {
			h := newHarness(t)
			h.srv.LoginOutcome = c.outcome
			got := h.run("login")
			expectExit(t, got, 2)
			expectContains(t, got.stderr, "transcript1: "+c.want+"\n")
			if got.stdout != "" {
				t.Fatalf("stdout %q", got.stdout)
			}

			expectSaved(t, config.KeyFile{})
		})
	}

	t.Run("too many login requests", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginCreateStatus = http.StatusTooManyRequests
		got := h.run("login")
		expectExit(t, got, 2)
		expectContains(t, got.stderr, "transcript1: Too many login attempts. Try again in a few minutes.\n")
		expectLog(t, h.srv.Log(), createPath)
	})

	t.Run("create fails with a server error", func(t *testing.T) {
		h := newHarness(t)
		h.srv.LoginCreateStatus = http.StatusServiceUnavailable
		got := h.run("login")
		expectExit(t, got, 1)
		expectContains(t, got.stderr, "transcript1: login:")
	})

	t.Run("too many login attempts while polling", func(t *testing.T) {
		h := newHarness(t)
		stub := newLoginStub(t, h)
		stub.pollStatus = http.StatusTooManyRequests
		stub.pollBody = `{"error":{"code":"rate_limit_exceeded","message":"slow down"}}`
		got := h.run("login")
		expectExit(t, got, 2)
		expectContains(t, got.stderr, "Too many login attempts. Try again in a few minutes.")
		if n := stub.count(pollPath); n != 1 {
			t.Fatalf("%d polls", n)
		}
	})

	t.Run("an empty or unknown status is an error", func(t *testing.T) {
		for _, body := range []string{`{}`, `{"status":""}`, `{"status":"weird","interval":1}`} {
			h := newHarness(t)
			stub := newLoginStub(t, h)
			stub.pollBody = body
			got := h.run("login")
			expectExit(t, got, 2)
			expectContains(t, got.stderr, "transcript1: the server sent an unexpected login status")
			if n := stub.count(pollPath); n != 1 {
				t.Fatalf("body %s: %d polls", body, n)
			}

			expectSaved(t, config.KeyFile{})
		}
	})

	t.Run("an incomplete login request is an error", func(t *testing.T) {
		h := newHarness(t)
		stub := newLoginStub(t, h)
		stub.createBody = `{"request_id":"req-1","user_code":"WXYZ-2345","verification_url":"https://transcript1.com/x","expires_in":600}`
		got := h.run("login")
		expectExit(t, got, 1)
		expectContains(t, got.stderr, "transcript1: login:")
		if n := stub.count(pollPath); n != 0 {
			t.Fatalf("%d polls", n)
		}
	})

	t.Run("an approval without a key is an error", func(t *testing.T) {
		h := newHarness(t)
		stub := newLoginStub(t, h)
		stub.pollBody = `{"status":"approved","token_id":"tok-s"}`
		got := h.run("login")
		expectExit(t, got, 1)
		expectContains(t, got.stderr, "transcript1: login:")
		expectSaved(t, config.KeyFile{})
	})

	t.Run("keeps the saved key when the account check fails", func(t *testing.T) {
		h := newHarness(t)
		h.writeSavedFile(config.KeyFile{APIKey: "t1_old", TokenID: "tok-old"})
		stub := newLoginStub(t, h)
		stub.accountStatus = http.StatusInternalServerError
		got := h.run("login")
		expectExit(t, got, 1)
		expectContains(t, got.stderr, "transcript1: login:")
		expectSaved(t, config.KeyFile{APIKey: "t1_stub", TokenID: "tok-s"})
		if n := stub.count("DELETE /user/tokens/tok-old"); n != 0 {
			t.Fatalf("%d revokes", n)
		}
	})
}

func TestBrowserLoginInterrupted(t *testing.T) {
	t.Run("cancels the request and says nothing more", func(t *testing.T) {
		h := newHarness(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		h.onSleep = cancel
		got := h.runWith(ctx, "login")
		expectExit(t, got, 130)
		if got.stdout != "" {
			t.Fatalf("stdout %q", got.stdout)
		}

		if got.stderr != openLine+waitingLine {
			t.Fatalf("stderr %q", got.stderr)
		}

		log := h.srv.Log()
		if log[len(log)-1] != cancelPath {
			t.Fatalf("log %v", log)
		}

		expectSaved(t, config.KeyFile{})
	})

	t.Run("stops the spinner before the exit", func(t *testing.T) {
		h := newHarness(t)
		h.tty = true
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		h.onSleep = cancel
		got := h.runWith(ctx, "login")
		expectExit(t, got, 130)
		if !strings.HasSuffix(got.stderr, "\r\x1b[K") {
			t.Fatalf("stderr %q", got.stderr)
		}
	})
}

func TestBrowserLoginReplacesPreviousKey(t *testing.T) {
	t.Run("revokes the old key after checking the account", func(t *testing.T) {
		h := newHarness(t)
		expectExit(t, h.run("login"), 0)
		before := len(h.srv.Log())
		h.srv.LoginTokenID = "tok-2"
		h.srv.LoginKey = "t1_second"
		got := h.run("login")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log()[before:], createPath, pollPath, accountPath, revokeOldPath)
		expectSaved(t, config.KeyFile{APIKey: "t1_second", TokenID: "tok-2"})
		if got.stdout != loggedInLine {
			t.Fatalf("stdout %q", got.stdout)
		}
	})

	t.Run("ignores a failed revoke without a word", func(t *testing.T) {
		for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusServiceUnavailable} {
			h := newHarness(t)
			expectExit(t, h.run("login"), 0)
			h.srv.LoginTokenID = "tok-2"
			h.srv.LoginKey = "t1_second"
			h.srv.RevokeStatus = status
			got := h.run("login")
			expectExit(t, got, 0)
			if strings.Contains(strings.ToLower(got.stderr), "revoke") {
				t.Fatalf("status %d stderr %q", status, got.stderr)
			}

			expectSaved(t, config.KeyFile{APIKey: "t1_second", TokenID: "tok-2"})
		}
	})

	t.Run("does not revoke a file without a token id", func(t *testing.T) {
		h := newHarness(t)
		h.writeSavedFile(config.KeyFile{APIKey: "t1_pasted"})
		expectExit(t, h.run("login"), 0)
		for _, line := range h.srv.Log() {
			if strings.HasPrefix(line, "DELETE") {
				t.Fatalf("log %v", h.srv.Log())
			}
		}
	})

	t.Run("revokes the previous key after a pasted key too", func(t *testing.T) {
		h := newHarness(t)
		h.writeSavedFile(config.KeyFile{APIKey: "t1_old", TokenID: "tok-1"})
		h.stdin = "t1_test\n"
		got := h.run("login")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log(), accountPath, revokeOldPath)
		expectSaved(t, config.KeyFile{APIKey: "t1_test"})
	})

	t.Run("keeps the key and the token id when the same key is pasted again", func(t *testing.T) {
		h := newHarness(t)
		h.writeSavedFile(config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		h.stdin = "t1_test\n"
		got := h.run("login", "--with-key")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log(), accountPath)
		expectSaved(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
	})
}

type loginStub struct {
	mu              sync.Mutex
	requests        []string
	verificationURL string
	createBody      string
	pollStatus      int
	pollBody        string
	accountStatus   int
}

func newLoginStub(t *testing.T, h *harness) *loginStub {
	t.Helper()
	stub := &loginStub{
		verificationURL: verifyURL,
		pollStatus:      http.StatusOK,
		pollBody:        `{"status":"approved","api_key":"t1_stub","token_id":"tok-s","email":"demo@transcript1.com"}`,
		accountStatus:   http.StatusOK,
	}
	srv := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(srv.Close)
	h.env[config.EnvAPIURL] = srv.URL

	return stub
}

func (s *loginStub) count(line string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, request := range s.requests {
		if request == line {
			n++
		}
	}

	return n
}

func (s *loginStub) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.Path)
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch r.Method + " " + r.URL.Path {
	case createPath:
		s.writeCreate(w)
	case pollPath:
		w.WriteHeader(s.pollStatus)
		fmt.Fprint(w, s.pollBody)
	case cancelPath:
		fmt.Fprint(w, `{"status":"canceled"}`)
	case accountPath:
		s.writeAccount(w)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"not_found","message":"no route"}}`)
	}
}

func (s *loginStub) writeCreate(w http.ResponseWriter) {
	w.WriteHeader(http.StatusCreated)
	if s.createBody != "" {
		fmt.Fprint(w, s.createBody)
		return
	}

	body, _ := json.Marshal(map[string]any{
		"request_id":       "req-1",
		"user_code":        "WXYZ-2345",
		"poll_secret":      "t1l_stub",
		"verification_url": s.verificationURL,
		"expires_in":       600,
		"interval":         1,
	})
	w.Write(body)
}

func (s *loginStub) writeAccount(w http.ResponseWriter) {
	if s.accountStatus != http.StatusOK {
		w.WriteHeader(s.accountStatus)
		fmt.Fprint(w, `{"error":{"code":"internal_error","message":"boom"}}`)

		return
	}

	fmt.Fprint(w, `{"object":"account","user":{"email":"demo@transcript1.com"},"plan":{"name":"Monthly"},"minutes":{"remaining":1000}}`)
}
