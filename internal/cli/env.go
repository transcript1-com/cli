package cli

import (
	"context"
	"io"
	"strings"
	"time"
)

type Env struct {
	Stdin               io.Reader
	Stdout, Stderr      io.Writer
	Getenv              func(string) string
	StdinTTY, StderrTTY bool
	ReadSecret          func() (string, error)
	OpenBrowser         func(string) error
	Sleep               func(context.Context, time.Duration) error
	Now                 func() time.Time
}

func (e Env) applyDefaults() Env {
	if e.Stdin == nil {
		e.Stdin = strings.NewReader("")
	}

	if e.Stdout == nil {
		e.Stdout = io.Discard
	}

	if e.Stderr == nil {
		e.Stderr = io.Discard
	}

	if e.Getenv == nil {
		e.Getenv = func(string) string { return "" }
	}

	if e.Sleep == nil {
		e.Sleep = sleepContext
	}

	if e.Now == nil {
		e.Now = time.Now
	}

	return e
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
