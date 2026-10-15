package cli

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestNoticePrint(t *testing.T) {
	t.Run("prints a ready line after the deadline has passed", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			n := &notice{line: make(chan string, 1), deadline: time.Now().Add(-time.Minute)}
			n.line <- noticeLine
			var stderr bytes.Buffer
			n.print(context.Background(), Env{Stderr: &stderr})
			if stderr.String() != noticeLine+"\n" {
				t.Fatalf("iteration %d: stderr %q", i, stderr.String())
			}
		}
	})

	t.Run("gives up at the deadline when no line arrives", func(t *testing.T) {
		n := &notice{line: make(chan string, 1), deadline: time.Now().Add(50 * time.Millisecond)}
		var stderr bytes.Buffer
		start := time.Now()
		n.print(context.Background(), Env{Stderr: &stderr})
		waited := time.Since(start)
		if waited > time.Second {
			t.Fatalf("waited %v stderr %q", waited, stderr.String())
		}

		if stderr.Len() != 0 {
			t.Fatalf("waited %v stderr %q", waited, stderr.String())
		}
	})
}
