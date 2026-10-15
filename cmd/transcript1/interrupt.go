package main

import (
	"os"

	"golang.org/x/term"
)

const exitInterrupted = 130

// the second signal exits at once, so restore the terminal
func handleSignals(signals <-chan os.Signal, cancel, restore func(), exit func(int)) {
	<-signals
	cancel()
	<-signals
	restore()
	exit(exitInterrupted)
}

// a hidden prompt turns echo off and exiting skips its defer
func buildTerminalRestore() func() {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return func() {}
	}

	state, err := term.GetState(fd)
	if err != nil {
		return func() {}
	}

	return func() { _ = term.Restore(fd, state) }
}
