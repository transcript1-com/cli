package output

import (
	"fmt"
	"io"
	"sync"

	"github.com/transcript1-com/cli/internal/batch"
)

type Reporter struct {
	mu    sync.Mutex
	w     io.Writer
	mode  Mode
	total int
	color bool
	width int
	last  map[int]batch.Stage
	live  []liveLine
	drawn int
}

func NewReporter(w io.Writer, mode Mode, total int, color bool) *Reporter {
	r := &Reporter{
		w:     w,
		mode:  mode,
		total: total,
		width: getTerminalWidth(w),
		last:  map[int]batch.Stage{},
	}
	if mode == TTY {
		r.color = color
	}

	return r
}

func (r *Reporter) Event(e batch.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mode == JSON {
		return
	}

	if e.Stage == batch.Finished {
		r.handleFinished(e)
		return
	}

	if r.mode == Quiet {
		return
	}

	if r.mode == TTY {
		r.setLive(e.Index, r.buildLiveText(e))
		return
	}

	if r.isRepeated(e) {
		return
	}

	r.last[e.Index] = e.Stage
	fmt.Fprintf(r.w, "%s: %s\n", r.buildLabel(e), buildStageText(e, false))
}

func (r *Reporter) isRepeated(e batch.Event) bool {
	last, seen := r.last[e.Index]
	if !seen {
		return false
	}

	return last == e.Stage
}

func (r *Reporter) Summary(results []batch.Result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mode == Quiet {
		return
	}

	if r.mode == JSON {
		return
	}

	fmt.Fprintln(r.w, buildSummaryLine(results))
}

func (r *Reporter) handleFinished(e batch.Event) {
	res := e.Result
	if res == nil {
		return
	}

	line := r.buildFinalLine(e, res)
	if r.mode == TTY {
		r.removeLive(e.Index, line)
		return
	}

	if line != "" {
		fmt.Fprintln(r.w, line)
	}
}

func (r *Reporter) buildFinalLine(e batch.Event, res *batch.Result) string {
	if res.Status == batch.Failed {
		return r.paint(red, addHint(ErrorLine(Display(e.Input), res.Err), res.GetHint()))
	}

	if r.mode == Quiet {
		return ""
	}

	text, color := buildFinalText(*res)
	if text == "" {
		return ""
	}

	if r.mode == TTY {
		return r.buildLabel(e) + "  " + r.paint(color, text)
	}

	return r.buildLabel(e) + ": " + text
}

func (r *Reporter) buildLabel(e batch.Event) string {
	return fmt.Sprintf("[%d/%d] %s", e.Index+1, r.total, Display(e.Input))
}

func (r *Reporter) paint(color, text string) string {
	if !r.color {
		return text
	}

	if color == "" {
		return text
	}

	if text == "" {
		return text
	}

	return color + text + reset
}
