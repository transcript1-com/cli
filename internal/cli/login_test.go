package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/transcript1-com/cli/internal/config"
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
		want := loggedInLine
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

	t.Run("rejects an empty line with --with-key", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "\n"
		got := h.run("login", "--with-key")
		if got.code != 2 {
			t.Fatalf("code %d stderr %q log %v", got.code, got.stderr, h.srv.Log())
		}

		if len(h.srv.Log()) != 0 {
			t.Fatalf("code %d stderr %q log %v", got.code, got.stderr, h.srv.Log())
		}

		expectContains(t, got.stderr, "no key entered")
	})

	t.Run("reads the first non-empty line", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "\n  \n t1_test \nignored\n"
		got := h.run("login")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log(), accountPath)
		expectSaved(t, config.KeyFile{APIKey: "t1_test"})
	})

	t.Run("starts the browser flow when stdin has no key", func(t *testing.T) {
		h := newHarness(t)
		h.stdin = "\n \n"
		got := h.run("login")
		expectExit(t, got, 0)
		expectLog(t, h.srv.Log(), createPath, pollPath, pollPath, accountPath)
	})

	t.Run("says no plan when the account has none", func(t *testing.T) {
		if got := buildLoggedInLine("a@b.c", "", 0); got != "Logged in as a@b.c. No plan, 0 minutes left." {
			t.Fatalf("got %q", got)
		}
	})
}

func TestLoginArguments(t *testing.T) {
	t.Run("rejects leftover arguments", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("login", "extra")
		expectExit(t, got, 2)
		expectContains(t, got.stderr, "login takes no arguments: use --with-key to paste a key")
		if len(h.srv.Log()) != 0 {
			t.Fatalf("log %v", h.srv.Log())
		}
	})

	t.Run("rejects an unknown flag", func(t *testing.T) {
		h := newHarness(t)
		got := h.run("login", "--bogus")
		expectExit(t, got, 2)
		expectContains(t, got.stderr, "transcript1: ", "bogus")
		if len(h.srv.Log()) != 0 {
			t.Fatalf("log %v", h.srv.Log())
		}
	})
}

func TestLoginOnTerminal(t *testing.T) {
	t.Run("pastes a key with --with-key", func(t *testing.T) {
		h := newHarness(t)
		h.secret = func() (string, error) { return "t1_test", nil }
		got := h.run("login", "--with-key")
		if got.code != 0 {
			t.Fatalf("code %d stderr %q", got.code, got.stderr)
		}

		expectContains(t, got.stderr, "API key")
		if key := loadSavedKey(t); key != "t1_test" {
			t.Fatalf("saved key %q", key)
		}
	})

	t.Run("opens the browser flow without asking for a key", func(t *testing.T) {
		h := newHarness(t)
		h.secret = func() (string, error) {
			t.Error("asked for a key")

			return "", nil
		}
		got := h.run("login")
		expectExit(t, got, 0)
		if strings.Contains(got.stderr, "API key") {
			t.Fatalf("stderr %q", got.stderr)
		}

		expectLog(t, h.srv.Log(), createPath, pollPath, pollPath, accountPath)
	})
}

func TestLoginInterrupted(t *testing.T) {
	h := newHarness(t)
	asked := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	h.secret = func() (string, error) {
		close(asked)
		<-release

		return "", nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-asked
		cancel()
	}()

	got := h.runWith(ctx, "login", "--with-key")
	if got.code != 130 {
		t.Fatalf("code %d stderr %q log %v", got.code, got.stderr, h.srv.Log())
	}

	if len(h.srv.Log()) != 0 {
		t.Fatalf("code %d stderr %q log %v", got.code, got.stderr, h.srv.Log())
	}

	if key := loadSavedKey(t); key != "" {
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

	if got.stdout != stillWorksLine {
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
