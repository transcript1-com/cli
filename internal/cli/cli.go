package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/transcript1-com/cli/internal/version"
)

const (
	commandGet    = "get"
	commandLogin  = "login"
	commandLogout = "logout"
	commandUpdate = "update"

	errSeeHelpTemplate = "%w (see transcript1 --help)"
)

func Run(ctx context.Context, args []string, env Env) int {
	env = env.applyDefaults()
	if len(args) == 0 {
		fmt.Fprint(env.Stdout, helpText)
		return exitOK
	}

	switch args[0] {
	case commandGet:
		return get(ctx, args[1:], env)
	case commandLogin:
		return login(ctx, args[1:], env)
	case commandLogout:
		return logout(ctx, args[1:], env)
	case commandUpdate:
		return update(ctx, args[1:], env)
	}

	return transcribe(ctx, args, env)
}

func (e Env) failUsage(err error) int {
	fmt.Fprintf(e.Stderr, "transcript1: %s\n", err)
	return exitUsage
}

func (e Env) handleEarly(s *settings, err error) (code int, done bool) {
	switch {
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(e.Stdout, helpText)
		return exitOK, true
	case err != nil:
		return e.failUsage(fmt.Errorf(errSeeHelpTemplate, err)), true
	case s.version:
		fmt.Fprintln(e.Stdout, "transcript1 "+version.Version)
		return exitOK, true
	}

	return 0, false
}
