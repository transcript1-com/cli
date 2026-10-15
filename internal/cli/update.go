package cli

import (
	"context"
	"fmt"
)

func update(_ context.Context, _ []string, env Env) int {
	fmt.Fprintln(env.Stderr, "updates are not configured in this build")
	return exitFailed
}
