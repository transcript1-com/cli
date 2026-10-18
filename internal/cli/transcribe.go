package cli

import (
	"context"

	"github.com/transcript1-com/cli/internal/batch"
	"github.com/transcript1-com/cli/internal/files"
	"github.com/transcript1-com/cli/internal/inputs"
)

func transcribe(ctx context.Context, args []string, env Env) int {
	s, raws, err := parseFlags(false, args)
	if code, done := env.handleEarly(s, err); done {
		return code
	}

	if err := validateFlags(s); err != nil {
		return env.failUsage(err)
	}

	ins, err := inputs.Expand(raws, env.Stdin)
	if err != nil {
		return env.failUsage(err)
	}

	if len(ins) == 0 {
		return env.failUsage(errNoInputs)
	}

	if err := validateInputs(s, ins); err != nil {
		return env.failUsage(err)
	}

	if _, err := files.FullNames(ins); err != nil {
		return env.failUsage(err)
	}

	opt := s.buildOptions(len(ins))
	if err := prepareOutput(opt); err != nil {
		return env.failUsage(err)
	}

	c, err := env.buildClient()
	if err != nil {
		return env.failUsage(err)
	}

	ss := newSession(ctx, env, s, len(ins))
	results, authErr := batch.Run(ctx, c, ins, opt, ss.emit)

	return ss.finish(ctx, results, authErr, true)
}
