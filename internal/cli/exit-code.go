package cli

import (
	"context"

	"github.com/transcript1-com/cli/internal/batch"
)

const (
	exitOK          = 0
	exitFailed      = 1
	exitUsage       = 2
	exitInterrupted = 130
)

func getExitCode(ctx context.Context, results []batch.Result, authErr error) int {
	if ctx.Err() != nil {
		return exitInterrupted
	}

	if authErr != nil {
		return exitUsage
	}

	for _, res := range results {
		switch res.Status {
		case batch.Failed, batch.Interrupted, batch.SkippedMinutes:
			return exitFailed
		}
	}

	return exitOK
}
