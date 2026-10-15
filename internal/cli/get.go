package cli

import (
	"context"
	"errors"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/batch"
)

var errGetUsage = errors.New("get takes one transcription id: transcript1 get [flags] <id>")

func get(ctx context.Context, args []string, env Env) int {
	s, raws, err := parseFlags(true, args)
	if code, done := env.handleEarly(s, err); done {
		return code
	}

	if err := validateFlags(s); err != nil {
		return env.failUsage(err)
	}

	if len(raws) != 1 {
		return env.failUsage(errGetUsage)
	}

	opt := s.buildOptions(1)
	if err := prepareOutput(opt); err != nil {
		return env.failUsage(err)
	}

	c, err := env.buildClient()
	if err != nil {
		return env.failUsage(err)
	}

	ss := newSession(env, s, 1)
	res := batch.Fetch(ctx, c, raws[0], opt, ss.emit)

	return ss.finish(ctx, []batch.Result{res}, findAuthError(res), false)
}

func findAuthError(res batch.Result) error {
	if res.Status != batch.Failed {
		return nil
	}

	if api.IsCode(res.Err, api.CodeUnauthorized) {
		return res.Err
	}

	return nil
}
