package cli

import (
	"fmt"
	"io"
	"time"
)

const (
	msgOpenURL       = "Open %s and confirm the code %s.\n"
	msgOpenedBrowser = "Opened your browser.\n"
	msgWaiting       = "Waiting for approval (Ctrl-C to cancel)..."

	spinnerFrames = `|/-\`
	spinnerTick   = 120 * time.Millisecond
	boldOn        = "\x1b[1m"
	boldOff       = "\x1b[0m"
	clearLine     = "\r\x1b[K"
)

type waitLine struct {
	stop chan struct{}
	done chan struct{}
}

func formatUserCode(env Env, code string) string {
	if !canUseEscapes(env) {
		return code
	}

	if env.Getenv("NO_COLOR") != "" {
		return code
	}

	return boldOn + code + boldOff
}

func buildFrame(n int) string {
	frame := spinnerFrames[n%len(spinnerFrames)]

	return fmt.Sprintf("\r%c %s", frame, msgWaiting)
}

// on a terminal the line is a spinner, otherwise one plain line
func startWaitLine(env Env) *waitLine {
	if !canUseEscapes(env) {
		fmt.Fprintln(env.Stderr, msgWaiting)

		return &waitLine{}
	}

	fmt.Fprint(env.Stderr, buildFrame(0))
	w := &waitLine{
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go w.spin(env.Stderr)

	return w
}

func (w *waitLine) spin(out io.Writer) {
	defer close(w.done)
	ticker := time.NewTicker(spinnerTick)
	defer ticker.Stop()
	for n := 1; ; n++ {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			fmt.Fprint(out, buildFrame(n))
		}
	}
}

// joins the spinner goroutine, so nothing else writes to stderr while it runs
func (w *waitLine) end(out io.Writer) {
	if w.stop == nil {
		return
	}

	close(w.stop)
	<-w.done
	w.stop = nil
	fmt.Fprint(out, clearLine)
}
