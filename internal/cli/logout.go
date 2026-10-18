package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/config"
)

const (
	msgNoSavedKey    = "No saved key."
	msgLoggedOut     = "Logged out."
	msgKeyRevoked    = "Logged out. The key was revoked."
	msgKeyStillWorks = "Logged out. The key still works until you revoke it on the API keys page."
	msgRevokeFailed  = "Could not revoke the key on the server. Revoke \"%s\" on the API keys page.\n"
	msgRemoveFailed  = "transcript1: cannot remove %s: %v\n"
	msgKeyStillInEnv = "%s is still set in this shell.\n"
)

type revokeResult int

const (
	revokeSkipped revokeResult = iota
	revokeDone
	revokeFailed
)

var errLogoutArgs = errors.New("logout takes no arguments")

func logout(ctx context.Context, args []string, env Env) int {
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

	result := revokeSavedKey(ctx, env, loadPreviousFile())
	if err := config.DeleteKey(); err != nil {
		fmt.Fprintf(env.Stderr, msgRemoveFailed, path, err)
		return exitFailed
	}

	if result == revokeFailed {
		fmt.Fprintf(env.Stderr, msgRevokeFailed, buildKeyName(getHostname()))
		if ctx.Err() != nil {
			return exitInterrupted
		}
	}

	fmt.Fprintln(env.Stdout, buildLogoutLine(result))
	if env.Getenv(config.EnvAPIKey) != "" {
		fmt.Fprintf(env.Stderr, msgKeyStillInEnv, config.EnvAPIKey)
	}

	return exitOK
}

func revokeSavedKey(ctx context.Context, env Env, file config.KeyFile) revokeResult {
	if file.TokenID == "" {
		return revokeSkipped
	}

	revokeCtx, cancel := context.WithTimeout(ctx, revokeTimeout)
	defer cancel()
	err := newLoginClient(env, file.APIKey).RevokeToken(revokeCtx, file.TokenID)
	if isRevoked(err) {
		return revokeDone
	}

	return revokeFailed
}

func buildLogoutLine(result revokeResult) string {
	if result == revokeDone {
		return msgKeyRevoked
	}

	if result == revokeSkipped {
		return msgKeyStillWorks
	}

	return msgLoggedOut
}

// a key the server no longer accepts counts as revoked
func isRevoked(err error) bool {
	if err == nil {
		return true
	}

	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return false
	}

	if apiErr.Status == http.StatusUnauthorized {
		return true
	}

	if apiErr.Status == http.StatusNotFound {
		return true
	}

	return apiErr.Code == api.CodeAuthInvalidSession
}
