package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/browser"
	"github.com/transcript1-com/cli/internal/config"
	"github.com/transcript1-com/cli/internal/output"
	"github.com/transcript1-com/cli/internal/version"
)

const (
	unknownHost     = "unknown host"
	maxPolls        = 400
	defaultInterval = 2 * time.Second
	maxInterval     = 60 * time.Second
	cancelTimeout   = 2 * time.Second
)

type loginFailureType string

func (e loginFailureType) Error() string {
	return string(e)
}

var (
	errLoginDenied   error = loginFailureType("Login was denied in the browser.")
	errLoginExpired  error = loginFailureType("The code expired. Run transcript1 login again.")
	errLoginInvalid  error = loginFailureType("This login request is no longer valid. Run transcript1 login again.")
	errTooManyLogins error = loginFailureType("Too many login attempts. Try again in a few minutes.")
	errKeyLimit      error = loginFailureType("You have 25 API keys. Revoke one at " + tokensURL + ", then run transcript1 login again.")
	errLoginStatus   error = loginFailureType("the server sent an unexpected login status. Run transcript1 login again.")

	errLoginIncomplete = errors.New("the server sent an incomplete login request")
	errLoginNoKey      = errors.New("the server approved the login without a key")
)

var loginFailures = map[string]error{
	api.CodeCliLoginDenied:    errLoginDenied,
	api.CodeCliLoginExpired:   errLoginExpired,
	api.CodeCliLoginConsumed:  errLoginInvalid,
	api.CodeCliLoginNotFound:  errLoginInvalid,
	api.CodePatLimitReached:   errKeyLimit,
	api.CodeRateLimitExceeded: errTooManyLogins,
}

func loginWithBrowser(ctx context.Context, env Env, noBrowser bool) int {
	client := newLoginClient(env, "")
	req, err := client.CreateLoginRequest(ctx, getHostname(), runtime.GOOS, version.Version)
	if ctx.Err() != nil {
		return exitInterrupted
	}

	if err != nil {
		return failLogin(env, err)
	}

	if !isCompleteRequest(req) {
		return failLogin(env, errLoginIncomplete)
	}

	created := env.Now()
	fmt.Fprintf(env.Stderr, msgOpenURL, req.VerificationURL, formatUserCode(env, req.UserCode))
	if tryOpenBrowser(env, noBrowser, req.VerificationURL) {
		fmt.Fprint(env.Stderr, msgOpenedBrowser)
	}

	wait := startWaitLine(env)
	poll, err := waitForApproval(ctx, env, client, req, created)
	wait.end(env.Stderr)
	if ctx.Err() != nil {
		cancelLoginRequest(client, req)
		return exitInterrupted
	}

	if err != nil {
		return failLogin(env, err)
	}

	return saveApprovedLogin(ctx, env, poll)
}

func isCompleteRequest(req *api.LoginRequest) bool {
	if req.RequestID == "" {
		return false
	}

	if req.PollSecret == "" {
		return false
	}

	if req.UserCode == "" {
		return false
	}

	return req.VerificationURL != ""
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return unknownHost
	}

	if strings.TrimSpace(name) == "" {
		return unknownHost
	}

	return name
}

func isWebURL(link string) bool {
	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	if parsed.Host == "" {
		return false
	}

	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func tryOpenBrowser(env Env, noBrowser bool, link string) bool {
	if env.OpenBrowser == nil {
		return false
	}

	if !browser.ShouldOpen(runtime.GOOS, env.Getenv, noBrowser) {
		return false
	}

	if !isWebURL(link) {
		return false
	}

	return env.OpenBrowser(link) == nil
}

func readInterval(seconds int) time.Duration {
	if seconds < 1 {
		return defaultInterval
	}

	return min(time.Duration(seconds)*time.Second, maxInterval)
}

func findLoginFailure(err error) error {
	var apiErr *api.Error
	if !errors.As(err, &apiErr) {
		return nil
	}

	if apiErr.Status == http.StatusTooManyRequests {
		return errTooManyLogins
	}

	return loginFailures[apiErr.Code]
}

func waitForApproval(ctx context.Context, env Env, client *api.Client, req *api.LoginRequest, created time.Time) (*api.LoginPoll, error) {
	interval := readInterval(req.Interval)
	deadline := created.Add(time.Duration(req.ExpiresIn) * time.Second)
	var lastErr error
	for range maxPolls {
		if err := env.Sleep(ctx, interval); err != nil {
			return nil, err
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if env.Now().After(deadline) {
			return nil, buildExpiredError(lastErr)
		}

		poll, err := client.PollLoginRequest(ctx, req.RequestID, req.PollSecret)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if failure := findLoginFailure(err); failure != nil {
			return nil, failure
		}

		if err != nil {
			if !api.Transient(err) {
				return nil, err
			}

			lastErr = err
			continue
		}

		lastErr = nil
		if poll.Status == api.LoginApproved {
			return poll, nil
		}

		if !slices.Contains([]string{api.LoginPending, api.LoginSlowDown}, poll.Status) {
			return nil, errLoginStatus
		}

		interval = readInterval(poll.Interval)
	}

	return nil, buildExpiredError(lastErr)
}

func buildExpiredError(lastErr error) error {
	if lastErr != nil {
		return lastErr
	}

	return errLoginExpired
}

func cancelLoginRequest(client *api.Client, req *api.LoginRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), cancelTimeout)
	defer cancel()
	client.CancelLoginRequest(ctx, req.RequestID, req.PollSecret)
}

func failLogin(env Env, err error) int {
	var failure loginFailureType
	if errors.As(err, &failure) {
		return env.failUsage(failure)
	}

	if failure := findLoginFailure(err); failure != nil {
		return env.failUsage(failure)
	}

	fmt.Fprintln(env.Stderr, output.ErrorLine("login", err))

	return exitFailed
}

func saveApprovedLogin(ctx context.Context, env Env, poll *api.LoginPoll) int {
	if poll.APIKey == "" {
		return failLogin(env, errLoginNoKey)
	}

	old := loadPreviousFile()
	file := config.KeyFile{
		APIKey:  poll.APIKey,
		TokenID: poll.TokenID,
	}
	if err := config.SaveKeyFile(file); err != nil {
		fmt.Fprintf(env.Stderr, msgSaveKeyFailed, err)
		return exitFailed
	}

	account, err := newLoginClient(env, file.APIKey).Account(ctx)
	if ctx.Err() != nil {
		return exitInterrupted
	}

	if err != nil {
		fmt.Fprintln(env.Stderr, output.ErrorLine("login", err))
		return exitFailed
	}

	return finishLogin(ctx, env, old, file, account)
}
