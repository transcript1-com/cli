package main

import (
	"os"
	"slices"
	"testing"
)

func TestHandleSignals(t *testing.T) {
	t.Run("restores the terminal before the second signal exits", func(t *testing.T) {
		var steps []string
		signals := make(chan os.Signal, 2)
		signals <- os.Interrupt
		signals <- os.Interrupt
		cancel := func() { steps = append(steps, "cancel") }
		restore := func() { steps = append(steps, "restore") }
		exit := func(code int) {
			if code != exitInterrupted {
				t.Errorf("exit code %d", code)
			}

			steps = append(steps, "exit")
		}

		handleSignals(signals, cancel, restore, exit)
		if want := []string{"cancel", "restore", "exit"}; !slices.Equal(steps, want) {
			t.Fatalf("steps %v", steps)
		}
	})
}
