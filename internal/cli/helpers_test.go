package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/transcript1/cli/internal/config"
	"github.com/transcript1/cli/internal/fakeapi"
)

const (
	rickID   = "dQw4w9WgXcQ"
	rickLink = "https://www.youtube.com/watch?v=" + rickID
	otherID  = "9bZkp7q19f0"
	otherURL = "https://www.youtube.com/watch?v=" + otherID
	thirdURL = "https://www.youtube.com/watch?v=aqz-KE-bpKQ"

	transcriptText = "Welcome to the show.\nThanks for having me.\n"
)

type harness struct {
	t      *testing.T
	srv    *fakeapi.Server
	env    map[string]string
	stdin  string
	secret func() (string, error)
}

type outcome struct {
	code           int
	stdout, stderr string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("HOME", dir)
	srv := fakeapi.New(t)
	env := map[string]string{config.EnvAPIURL: srv.URL, config.EnvAPIKey: srv.Key}

	return &harness{t: t, srv: srv, env: env}
}

func (h *harness) run(args ...string) outcome {
	return h.runWith(context.Background(), args...)
}

func (h *harness) runWith(ctx context.Context, args ...string) outcome {
	var stdout, stderr bytes.Buffer
	env := Env{
		Stdin:      strings.NewReader(h.stdin),
		Stdout:     &stdout,
		Stderr:     &stderr,
		Getenv:     func(name string) string { return h.env[name] },
		StdinTTY:   h.secret != nil,
		ReadSecret: h.secret,
	}

	code := Run(ctx, args, env)

	return outcome{
		code:   code,
		stdout: stdout.String(),
		stderr: stderr.String(),
	}
}

func (h *harness) write(name, content string) string {
	h.t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		h.t.Fatal(err)
	}

	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		h.t.Fatal(err)
	}

	return name
}

func (h *harness) read(name string) string {
	h.t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		h.t.Fatal(err)
	}

	return string(data)
}

func (h *harness) exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func (h *harness) writeMedia(name string) string {
	return h.write(name, "0123456789abcdefghij")
}

func expectContains(t *testing.T, got string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(got, part) {
			t.Fatalf("%q does not contain %q", got, part)
		}
	}
}
