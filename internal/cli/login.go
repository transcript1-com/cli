package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/transcript1/cli/internal/api"
	"github.com/transcript1/cli/internal/config"
	"github.com/transcript1/cli/internal/output"
)

const (
	errReadKeyTemplate = "cannot read the key: %w"
	msgSaveKeyFailed   = "transcript1: cannot save the key: %v\n"
	msgKeyPrompt       = "Create an API key at %s\nPaste your API key (input is hidden): "
	msgSignedIn        = "Signed in as %s. %s, %s %s left."
)

var (
	errLoginArgs    = errors.New("login takes no arguments: pipe the key in or type it when asked")
	errNoKeyEntered = errors.New("no key entered. Create one at " + tokensURL)
	errKeyRejected  = errors.New("that key was not accepted. Create one at " + tokensURL)
)

func login(ctx context.Context, args []string, env Env) int {
	if len(args) > 0 {
		return env.failUsage(errLoginArgs)
	}

	key, err := readKey(ctx, env)
	if ctx.Err() != nil {
		return exitInterrupted
	}

	if err != nil {
		return env.failUsage(fmt.Errorf(errReadKeyTemplate, err))
	}

	if key == "" {
		return env.failUsage(errNoKeyEntered)
	}

	account, err := api.New(config.APIURL(env.Getenv), key).Account(ctx)
	if api.IsCode(err, api.CodeUnauthorized, api.CodeInsufficientScope) {
		return env.failUsage(errKeyRejected)
	}

	if ctx.Err() != nil {
		return exitInterrupted
	}

	if err != nil {
		fmt.Fprintln(env.Stderr, output.ErrorLine("login", err))
		return exitFailed
	}

	if err := config.SaveKey(key); err != nil {
		fmt.Fprintf(env.Stderr, msgSaveKeyFailed, err)
		return exitFailed
	}

	plan := ""
	if account.Plan != nil {
		plan = account.Plan.Name
	}

	fmt.Fprintln(env.Stdout, buildSignedInLine(account.User.Email, plan, account.Minutes.Remaining))

	return exitOK
}

func canReadSecret(env Env) bool {
	if !env.StdinTTY {
		return false
	}

	return env.ReadSecret != nil
}

func isReadError(err error) bool {
	if err == nil {
		return false
	}

	return !errors.Is(err, io.EOF)
}

// the hidden read cannot be interrupted, so run it aside
func readSecret(ctx context.Context, env Env) (string, error) {
	type answer struct {
		key string
		err error
	}

	fmt.Fprintf(env.Stderr, msgKeyPrompt, tokensURL)
	done := make(chan answer, 1)
	go func() {
		key, err := env.ReadSecret()
		done <- answer{key, err}
	}()

	defer fmt.Fprintln(env.Stderr)
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case a := <-done:
		return strings.TrimSpace(a.key), a.err
	}
}

func readKey(ctx context.Context, env Env) (string, error) {
	if canReadSecret(env) {
		return readSecret(ctx, env)
	}

	line, err := bufio.NewReader(env.Stdin).ReadString('\n')
	if isReadError(err) {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func buildSignedInLine(email, plan string, minutes int) string {
	planText := "No plan"
	if plan != "" {
		planText = plan + " plan"
	}

	unit := "minutes"
	if minutes == 1 {
		unit = "minute"
	}

	return fmt.Sprintf(msgSignedIn, email, planText, formatThousands(minutes), unit)
}
