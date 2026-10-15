package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/transcript1/cli/internal/cli"
	"golang.org/x/term"
)

const exitInterrupted = 130

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		cancel()
		<-signals
		os.Exit(exitInterrupted)
	}()

	env := cli.Env{
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Getenv:     os.Getenv,
		StdinTTY:   term.IsTerminal(int(os.Stdin.Fd())),
		StderrTTY:  term.IsTerminal(int(os.Stderr.Fd())),
		ReadSecret: readSecret,
	}
	code := cli.Run(ctx, os.Args[1:], env)
	cancel()
	os.Exit(code)
}

func readSecret() (string, error) {
	key, err := term.ReadPassword(int(os.Stdin.Fd()))
	return string(key), err
}
