package cli

import (
	"io"
	"strings"
)

type Env struct {
	Stdin               io.Reader
	Stdout, Stderr      io.Writer
	Getenv              func(string) string
	StdinTTY, StderrTTY bool
	ReadSecret          func() (string, error)
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

	return e
}
