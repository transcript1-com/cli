package cli

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/config"
)

const (
	revokedLine      = "Logged out. The key was revoked.\n"
	loggedOutLine    = "Logged out.\n"
	stillWorksLine   = "Logged out. The key still works until you revoke it on the API keys page.\n"
	logoutDeletePath = "DELETE /user/tokens/tok-1"
)

func newLogoutHarness(t *testing.T, file config.KeyFile) *harness {
	t.Helper()
	h := newHarness(t)
	delete(h.env, config.EnvAPIKey)
	h.writeSavedFile(file)

	return h
}

func expectNoSavedFile(t *testing.T) {
	t.Helper()
	path, err := config.KeyPath()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s still exists", path)
	}
}

func buildRevokeWarning(t *testing.T) string {
	t.Helper()
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}

	return "Could not revoke the key on the server. Revoke \"transcript1 CLI (" + host + ")\" on the API keys page.\n"
}

func TestLogoutRevokesKey(t *testing.T) {
	t.Run("revokes the saved key with its own bearer", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		got := h.run("logout")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log(), logoutDeletePath)
		if got.stdout != revokedLine {
			t.Fatalf("stdout %q", got.stdout)
		}

		if got.stderr != "" {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectNoSavedFile(t)
		for _, request := range h.srv.Requests() {
			if request.Header.Get("Authorization") != "Bearer t1_test" {
				t.Fatalf("authorization %q", request.Header.Get("Authorization"))
			}
		}
	})

	t.Run("counts a revoked key as already revoked", func(t *testing.T) {
		for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
			h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
			h.srv.RevokeStatus = status
			got := h.run("logout")
			expectExit(t, got, 0)
			if got.stdout != revokedLine {
				t.Fatalf("status %d stdout %q", status, got.stdout)
			}

			if got.stderr != "" {
				t.Fatalf("status %d stderr %q", status, got.stderr)
			}

			expectNoSavedFile(t)
		}
	})

	t.Run("warns when the server fails", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		h.srv.RevokeStatus = http.StatusServiceUnavailable
		got := h.run("logout")
		expectExit(t, got, 0)
		if got.stdout != loggedOutLine {
			t.Fatalf("stdout %q", got.stdout)
		}

		if got.stderr != buildRevokeWarning(t) {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectNoSavedFile(t)
	})

	t.Run("warns when the server is offline", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		h.env[config.EnvAPIURL] = "http://127.0.0.1:1"
		got := h.run("logout")
		expectExit(t, got, 0)
		if got.stdout != loggedOutLine {
			t.Fatalf("stdout %q", got.stdout)
		}

		if got.stderr != buildRevokeWarning(t) {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectNoSavedFile(t)
	})

	t.Run("deletes the file and warns on Ctrl-C during the revoke", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		h.srv.RevokeStatus = http.StatusServiceUnavailable
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		h.onSleep = cancel
		got := h.runWith(ctx, "logout")
		expectExit(t, got, 130)
		if got.stderr != buildRevokeWarning(t) {
			t.Fatalf("stderr %q", got.stderr)
		}

		if strings.Contains(got.stdout, "Logged out") {
			t.Fatalf("stdout %q", got.stdout)
		}

		expectNoSavedFile(t)
	})

	t.Run("does not call the server for a key without a token id", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test"})
		got := h.run("logout")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log())
		if got.stdout != stillWorksLine {
			t.Fatalf("stdout %q", got.stdout)
		}

		expectNoSavedFile(t)
	})

	t.Run("says there is nothing to do the second time", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		expectExit(t, h.run("logout"), 0)
		got := h.run("logout")
		expectExit(t, got, 0)
		if got.stdout != "No saved key.\n" {
			t.Fatalf("stdout %q", got.stdout)
		}

		expectLog(t, h.srv.Log(), logoutDeletePath)
	})

	t.Run("keeps the env key note", func(t *testing.T) {
		h := newLogoutHarness(t, config.KeyFile{APIKey: "t1_test", TokenID: "tok-1"})
		h.env[config.EnvAPIKey] = "t1_other"
		got := h.run("logout")
		expectExit(t, got, 0)
		expectContains(t, got.stderr, "TRANSCRIPT1_API_KEY is still set in this shell.")
	})
}

func TestBuildKeyName(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{"trims whitespace", "  mba\n", "transcript1 CLI (mba)"},
		{"cuts to 64 runes", strings.Repeat("a", 100), "transcript1 CLI (" + strings.Repeat("a", 64) + ")"},
		{"drops control characters", "\a", "transcript1 CLI (unknown host)"},
		{"empty host", "", "transcript1 CLI (unknown host)"},
		{"trims the end after the cut", strings.Repeat("a", 63) + " b", "transcript1 CLI (" + strings.Repeat("a", 63) + ")"},
		{"counts runes", strings.Repeat("é", 70), "transcript1 CLI (" + strings.Repeat("é", 64) + ")"},
		{"drops bidi override", "mba\u202ecod.exe", "transcript1 CLI (mbacod.exe)"},
		{"drops zero width space", "m\u200bba", "transcript1 CLI (mba)"},
		{"drops bom", "\ufeffmba", "transcript1 CLI (mba)"},
		{"drops line and paragraph separators", "m\u2028b\u2029a", "transcript1 CLI (mba)"},
		{"only format characters", "\u200b\u202e", "transcript1 CLI (unknown host)"},
		{"keeps inner spaces", "my  mac", "transcript1 CLI (my  mac)"},
		{"strips before counting", strings.Repeat("a\u200b", 70), "transcript1 CLI (" + strings.Repeat("a", 64) + ")"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildKeyName(tt.host); got != tt.want {
				t.Fatalf("buildKeyName(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestIsRevoked(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, true},
		{"401", &api.Error{Status: 401, Code: api.CodeUnauthorized}, true},
		{"404", &api.Error{Status: 404, Code: api.CodePatNotFound}, true},
		{"invalid session", &api.Error{Status: 403, Code: api.CodeAuthInvalidSession}, true},
		{"503", &api.Error{Status: 503, Code: api.CodeServiceUnavailable}, false},
		{"other 403", &api.Error{Status: 403, Code: api.CodeInsufficientScope}, false},
		{"plain error", errors.New("boom"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRevoked(tt.err); got != tt.want {
				t.Fatalf("isRevoked(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
