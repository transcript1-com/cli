package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/config"
	"github.com/transcript1-com/cli/internal/output"
)

const (
	errReadKeyTemplate = "cannot read the key: %w"
	msgSaveKeyFailed   = "transcript1: cannot save the key: %v\n"
	msgKeyPrompt       = "Create an API key at %s\nPaste your API key (input is hidden): "
	msgLoggedIn        = "Logged in as %s. %s, %s %s left."
	msgEnvKeyWins      = "%s is set in this shell and takes precedence over the saved key.\n"

	maxPipedBytes = 64 * 1024
	revokeTimeout = 10 * time.Second
)

var (
	errLoginArgs    = errors.New("login takes no arguments: use --with-key to paste a key")
	errNoKeyEntered = errors.New("no key entered. Create one at " + tokensURL)
	errKeyRejected  = errors.New("that key was not accepted. Create one at " + tokensURL)
)

func login(ctx context.Context, args []string, env Env) int {
	flags := flag.NewFlagSet(commandLogin, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	withKey := flags.Bool("with-key", false, "")
	noBrowser := flags.Bool("no-browser", false, "")
	err := flags.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(env.Stdout, helpText)
		return exitOK
	}

	if err != nil {
		return env.failUsage(err)
	}

	if flags.NArg() > 0 {
		return env.failUsage(errLoginArgs)
	}

	if *withKey {
		return loginPastingKey(ctx, env)
	}

	if env.StdinTTY {
		return loginWithBrowser(ctx, env, *noBrowser)
	}

	key, err := readPipedKey(env)
	if err != nil {
		return env.failUsage(fmt.Errorf(errReadKeyTemplate, err))
	}

	if key == "" {
		return loginWithBrowser(ctx, env, *noBrowser)
	}

	return loginWithKey(ctx, env, key)
}

func loginPastingKey(ctx context.Context, env Env) int {
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

	return loginWithKey(ctx, env, key)
}

func loginWithKey(ctx context.Context, env Env, key string) int {
	account, err := newLoginClient(env, key).Account(ctx)
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

	old := loadPreviousFile()
	file := config.KeyFile{APIKey: key}
	if old.APIKey == key {
		file.TokenID = old.TokenID
	}

	if err := config.SaveKeyFile(file); err != nil {
		fmt.Fprintf(env.Stderr, msgSaveKeyFailed, err)
		return exitFailed
	}

	return finishLogin(ctx, env, old, file, account)
}

func newLoginClient(env Env, key string) *api.Client {
	client := api.New(config.APIURL(env.Getenv), key)
	client.Sleep = env.Sleep

	return client
}

func loadPreviousFile() config.KeyFile {
	file, err := config.LoadKeyFile()
	if err != nil {
		return config.KeyFile{}
	}

	return file
}

func finishLogin(ctx context.Context, env Env, old, file config.KeyFile, account *api.Account) int {
	revokePrevious(ctx, env, old, file)
	plan := ""
	if account.Plan != nil {
		plan = account.Plan.Name
	}

	fmt.Fprintln(env.Stdout, buildLoggedInLine(account.User.Email, plan, account.Minutes.Remaining))
	if env.Getenv(config.EnvAPIKey) != "" {
		fmt.Fprintf(env.Stderr, msgEnvKeyWins, config.EnvAPIKey)
	}

	return exitOK
}

// best effort: a failed revoke is never worth a word
func revokePrevious(ctx context.Context, env Env, old, file config.KeyFile) {
	if old.TokenID == "" {
		return
	}

	if old.TokenID == file.TokenID {
		return
	}

	if old.APIKey == file.APIKey {
		return
	}

	revokeCtx, cancel := context.WithTimeout(ctx, revokeTimeout)
	defer cancel()
	newLoginClient(env, old.APIKey).RevokeToken(revokeCtx, old.TokenID)
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

// returns the first non-empty line of piped stdin
func readPipedKey(env Env) (string, error) {
	data, err := io.ReadAll(io.LimitReader(env.Stdin, maxPipedBytes))
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		key := strings.TrimSpace(line)
		if key != "" {
			return key, nil
		}
	}

	return "", nil
}

func buildLoggedInLine(email, plan string, minutes int) string {
	planText := "No plan"
	if plan != "" {
		planText = plan + " plan"
	}

	unit := "minutes"
	if minutes == 1 {
		unit = "minute"
	}

	return fmt.Sprintf(msgLoggedIn, email, planText, formatThousands(minutes), unit)
}
