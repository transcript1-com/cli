package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/transcript1-com/cli/internal/config"
	selfupdate "github.com/transcript1-com/cli/internal/update"
	"github.com/transcript1-com/cli/internal/version"
)

const (
	updateTimeout = 10 * time.Minute

	errNoSelfUpdateTemplate = "this %s build does not update itself; install a release with: %s"
	msgUpToDate             = "transcript1 is up to date (%s).\n"
	msgUpdated              = "Updated transcript1 %s → %s.\n"
)

var errUpdateUsage = errors.New("update takes no arguments: transcript1 update")

func update(ctx context.Context, args []string, env Env) int {
	if len(args) != 0 {
		return env.failUsage(errUpdateUsage)
	}

	pub, err := selfupdate.PublicKey()
	if err != nil {
		return env.fail(err)
	}

	if !selfupdate.IsVersion(version.Version) {
		return env.fail(fmt.Errorf(errNoSelfUpdateTemplate, version.Version, selfupdate.InstallCommand))
	}

	exe, err := findExecutable()
	if err != nil {
		return env.fail(err)
	}

	hc := &http.Client{Timeout: updateTimeout}
	m, err := selfupdate.Fetch(ctx, hc, config.DownloadsURL(env.Getenv), pub)
	if err != nil {
		return env.fail(err)
	}

	if !selfupdate.Newer(m.Version, version.Version) {
		fmt.Fprintf(env.Stdout, msgUpToDate, version.Version)
		return exitOK
	}

	if err := selfupdate.Apply(ctx, hc, m, exe); err != nil {
		return env.fail(err)
	}

	fmt.Fprintf(env.Stdout, msgUpdated, version.Version, m.Version)

	return exitOK
}

func findExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(exe)
}

func (e Env) fail(err error) int {
	fmt.Fprintf(e.Stderr, "transcript1: %s\n", err)
	return exitFailed
}
