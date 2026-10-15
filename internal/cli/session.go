package cli

import (
	"context"
	"fmt"

	"github.com/transcript1/cli/internal/batch"
	"github.com/transcript1/cli/internal/output"
)

const msgStillRunning = "still running: %s (%s), fetch with transcript1 get %s\n"

type session struct {
	env    Env
	s      *settings
	rep    *output.Reporter
	out    *printer
	notice *notice
}

func newSession(ctx context.Context, env Env, s *settings, total int) *session {
	return &session{
		env: env,
		s:   s,
		rep: output.NewReporter(env.Stderr, s.getMode(env), total, env.Getenv("NO_COLOR") == ""),
		out: &printer{
			out:  env.Stdout,
			errw: env.Stderr,
			json: s.json,
		},
		notice: startNotice(ctx, env, s),
	}
}

func (s *settings) getMode(env Env) output.Mode {
	switch {
	case s.json:
		return output.JSON
	case s.quiet:
		return output.Quiet
	case env.StderrTTY:
		return output.TTY
	}

	return output.Plain
}

func (ss *session) emit(e batch.Event) {
	ss.rep.Event(e)
	ss.out.handleEvent(e)
}

func (ss *session) finish(ctx context.Context, results []batch.Result, authErr error, summary bool) int {
	ss.out.printResults(results)
	if ctx.Err() != nil {
		ss.printStillRunning(results)
	}

	if summary {
		ss.rep.Summary(results)
	}

	ss.printSignInHint(results, authErr)
	ss.notice.print(ctx, ss.env)

	code := getExitCode(ctx, results, authErr)
	if code != exitOK {
		return code
	}

	if ss.out.hasFailed() {
		return exitFailed
	}

	return code
}

func (ss *session) printStillRunning(results []batch.Result) {
	for _, res := range results {
		if res.Status != batch.StillRunning {
			continue
		}

		if res.T == nil {
			continue
		}

		id := res.T.ID
		fmt.Fprintf(ss.env.Stderr, msgStillRunning, id, res.Input.Raw, id)
	}
}

func (ss *session) printSignInHint(results []batch.Result, authErr error) {
	if authErr == nil {
		return
	}

	if !ss.s.json {
		return
	}

	for _, res := range results {
		if res.Err == authErr {
			fmt.Fprintln(ss.env.Stderr, output.ErrorLine(output.Display(res.Input), authErr))
			return
		}
	}
}
