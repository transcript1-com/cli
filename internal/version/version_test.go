package version

import (
	"runtime"
	"testing"
)

func TestUserAgent(t *testing.T) {
	t.Run("names the version, os and arch", func(t *testing.T) {
		want := "transcript1-cli/dev (" + runtime.GOOS + "/" + runtime.GOARCH + ")"
		if got := UserAgent(); got != want {
			t.Fatalf("UserAgent() = %q, want %q", got, want)
		}
	})
}
