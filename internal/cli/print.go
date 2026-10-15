package cli

import (
	"fmt"
	"io"
	"sync"

	"github.com/transcript1/cli/internal/batch"
	"github.com/transcript1/cli/internal/output"
)

type printer struct {
	mu   sync.Mutex
	out  io.Writer
	errw io.Writer
	json bool
}

func (p *printer) shouldPrintLine(e batch.Event) bool {
	if !p.json {
		return false
	}

	if e.Stage != batch.Finished {
		return false
	}

	return e.Result != nil
}

func (p *printer) handleEvent(e batch.Event) {
	if !p.shouldPrintLine(e) {
		return
	}

	line, err := output.JSONLine(*e.Result)
	if err != nil {
		fmt.Fprintln(p.errw, output.ErrorLine(output.Display(e.Input), err))
		return
	}

	p.write(line)
}

func (p *printer) write(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = p.out.Write(data)
}

func (p *printer) shouldPrintID(res batch.Result) bool {
	if res.Status != batch.Started {
		return false
	}

	if p.json {
		return false
	}

	return res.T != nil
}

func (p *printer) printResults(results []batch.Result) {
	for _, res := range results {
		if p.shouldPrintID(res) {
			p.write([]byte(res.T.ID + " " + res.T.Status + "\n"))
		}

		if len(res.Body) > 0 {
			p.write(res.Body)
		}
	}
}
