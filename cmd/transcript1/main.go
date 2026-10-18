package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/transcript1-com/cli/internal/browser"
	"github.com/transcript1-com/cli/internal/cli"
	"github.com/transcript1-com/cli/internal/update"
	"golang.org/x/term"
)

func main() {
	if exe, err := os.Executable(); err == nil {
		update.CleanupOld(exe)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restore := buildTerminalRestore()
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go handleSignals(signals, cancel, restore, os.Exit)

	env := cli.Env{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Getenv:      os.Getenv,
		StdinTTY:    term.IsTerminal(int(os.Stdin.Fd())),
		StderrTTY:   term.IsTerminal(int(os.Stderr.Fd())),
		ReadSecret:  readSecret,
		OpenBrowser: browser.Open,
	}
	code := cli.Run(ctx, os.Args[1:], env)
	cancel()
	restore()
	os.Exit(code)
}

func readSecret() (string, error) {
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	return string(key), err
}
