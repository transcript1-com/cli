package cli

import (
	"testing"

	"github.com/transcript1/cli/internal/config"
)

func loadSavedKey(t *testing.T) string {
	t.Helper()
	key, err := config.LoadKey()
	if err != nil {
		t.Fatal(err)
	}

	return key
}

func TestLoginPiped(t *testing.T) {
	t.Run("saves a key the API accepts and prints the account", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "t1_test\r\n"
		got := h.run("login")
		want := "Signed in as demo@transcript1.com. Monthly plan, 1,000 minutes left.\n"
		if got.code != 0 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stdout != want {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if key := loadSavedKey(t); key != "t1_test" {
			t.Fatalf("saved key %q", key)
		}
	})

	t.Run("saves nothing for a rejected key", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "t1_bad\n"
		got := h.run("login")
		if got.code != 2 {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		if got.stdout != "" {
			t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
		}

		expectContains(t, got.stderr, "not accepted", "https://transcript1.com/dashboard/account/tokens")
		if key := loadSavedKey(t); key != "" {
			t.Fatalf("saved key %q", key)
		}
	})

	t.Run("rejects an empty line", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "\n"
		got := h.run("login")
		if got.code != 2 {
			t.Fatalf("code %d stderr %q log %v", got.code, got.stderr, h.srv.Log())
		}

		if len(h.srv.Log()) != 0 {
			t.Fatalf("code %d stderr %q log %v", got.code, got.stderr, h.srv.Log())
		}
	})

	t.Run("says no plan when the account has none", func(t *testing.T) {
		if got := buildSignedInLine("a@b.c", "", 0); got != "Signed in as a@b.c. No plan, 0 minutes left." {
			t.Fatalf("got %q", got)
		}
	})
}

func TestLoginOnTerminal(t *testing.T) {
	h := newHarness(t)
	h.secret = func() (string, error) { return "t1_test", nil }
	got := h.run("login")
	if got.code != 0 {
		t.Fatalf("code %d stderr %q", got.code, got.stderr)
	}

	expectContains(t, got.stderr, "API key")
	if key := loadSavedKey(t); key != "t1_test" {
		t.Fatalf("saved key %q", key)
	}
}

func TestLogout(t *testing.T) {
	h := newHarness(t)
	h.stdin = "t1_test\n"
	h.run("login")
	got := h.run("logout")
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "Signed out.\n" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	got = h.run("logout")
	if got.code != 0 {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}

	if got.stdout != "No saved key.\n" {
		t.Fatalf("code %d stdout %q stderr %q", got.code, got.stdout, got.stderr)
	}
}
