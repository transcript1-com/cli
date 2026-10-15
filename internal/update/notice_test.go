package update

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNoticeOncePerDay(t *testing.T) {
	pub, priv := newKey(t)
	useKey(t, pub)
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	notice := func(d *fakeDownloads, dir string, at time.Time) string {
		return Notice(context.Background(), http.DefaultClient, d.URL, "1.0.0", dir, at)
	}

	t.Run("announces a newer version once a day", func(t *testing.T) {
		d := newDownloads(t)
		d.publish(t, priv, "1.2.0", []byte("archive"))
		dir := t.TempDir()
		if got := notice(d, dir, now); got != "transcript1 1.2.0 is available. Run transcript1 update." {
			t.Fatalf("notice %q", got)
		}

		before := d.requests.Load()
		if got := notice(d, dir, now.Add(23*time.Hour)); got != "" {
			t.Fatalf("notice %q", got)
		}

		if d.requests.Load() != before {
			t.Fatal("checked twice within 24 hours")
		}

		if got := notice(d, dir, now.Add(25*time.Hour)); got == "" {
			t.Fatal("no notice the next day")
		}
	})

	t.Run("stays quiet on the same version", func(t *testing.T) {
		d := newDownloads(t)
		d.publish(t, priv, "1.0.0", []byte("archive"))
		if got := notice(d, t.TempDir(), now); got != "" {
			t.Fatalf("notice %q", got)
		}
	})

	t.Run("stays quiet on a manifest signed by another key", func(t *testing.T) {
		d := newDownloads(t)
		_, other := newKey(t)
		d.publish(t, other, "1.2.0", []byte("archive"))
		if got := notice(d, t.TempDir(), now); got != "" {
			t.Fatalf("notice %q", got)
		}
	})

	t.Run("records a failed check so offline runs are not slowed", func(t *testing.T) {
		d := newDownloads(t)
		d.Close()
		dir := t.TempDir()
		if got := notice(d, dir, now); got != "" {
			t.Fatalf("notice %q", got)
		}

		if _, err := os.Stat(filepath.Join(dir, "update-check.json")); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("makes no request when updates are not configured", func(t *testing.T) {
		d := newDownloads(t)
		d.publish(t, priv, "1.2.0", []byte("archive"))
		releaseKey = "unset\n"
		got := notice(d, t.TempDir(), now)
		if got != "" {
			t.Fatalf("notice %q requests %d", got, d.requests.Load())
		}

		if d.requests.Load() != 0 {
			t.Fatalf("notice %q requests %d", got, d.requests.Load())
		}
	})
}
