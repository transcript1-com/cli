package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/transcript1/cli/internal/config"
	selfupdate "github.com/transcript1/cli/internal/update"
	"github.com/transcript1/cli/internal/version"
)

const (
	noticeTimeout = 3 * time.Second
	devVersion    = "dev"
	cacheDirName  = "transcript1"
)

var checkUpdate = selfupdate.Notice

type notice struct {
	line     chan string
	deadline time.Time
}

func shouldCheckUpdate(env Env, s *settings) bool {
	if !env.StderrTTY {
		return false
	}

	if s.json {
		return false
	}

	if s.quiet {
		return false
	}

	if env.Getenv(config.EnvNoUpdateCheck) == "1" {
		return false
	}

	return version.Version != devVersion
}

func startNotice(ctx context.Context, env Env, s *settings) *notice {
	if !shouldCheckUpdate(env, s) {
		return nil
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		return nil
	}

	n := &notice{line: make(chan string, 1), deadline: time.Now().Add(noticeTimeout)}
	base := config.DownloadsURL(env.Getenv)
	current := version.Version
	dir := filepath.Join(cache, cacheDirName)
	go func() {
		ctx, cancel := context.WithDeadline(ctx, n.deadline)
		defer cancel()
		n.line <- checkUpdate(ctx, &http.Client{}, base, current, dir, time.Now())
	}()

	return n
}

func (n *notice) print(ctx context.Context, env Env) {
	if n == nil {
		return
	}

	if ctx.Err() != nil {
		return
	}

	timer := time.NewTimer(time.Until(n.deadline))
	defer timer.Stop()
	select {
	case line := <-n.line:
		if line != "" {
			fmt.Fprintln(env.Stderr, line)
		}
	case <-timer.C:
	case <-ctx.Done():
	}
}
