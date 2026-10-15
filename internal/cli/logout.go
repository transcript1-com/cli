package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/transcript1/cli/internal/config"
)

const (
	msgNoSavedKey    = "No saved key."
	msgSignedOut     = "Signed out."
	msgRemoveFailed  = "transcript1: cannot remove %s: %v\n"
	msgKeyStillInEnv = "%s is still set in this shell.\n"
)

var errLogoutArgs = errors.New("logout takes no arguments")

func logout(args []string, env Env) int {
	if len(args) > 0 {
		return env.failUsage(errLogoutArgs)
	}

	path, err := config.KeyPath()
	if err != nil {
		return env.failUsage(err)
	}

	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintln(env.Stdout, msgNoSavedKey)
		return exitOK
	}

	if err := config.DeleteKey(); err != nil {
		fmt.Fprintf(env.Stderr, msgRemoveFailed, path, err)
		return exitFailed
	}

	fmt.Fprintln(env.Stdout, msgSignedOut)
	if env.Getenv(config.EnvAPIKey) != "" {
		fmt.Fprintf(env.Stderr, msgKeyStillInEnv, config.EnvAPIKey)
	}

	return exitOK
}
