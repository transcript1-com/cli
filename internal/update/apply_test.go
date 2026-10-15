//go:build !windows

package update

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "transcript1")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	return exe
}

func assertOld(t *testing.T, exe string) {
	t.Helper()
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("exe %q %v", data, err)
	}

	if string(data) != "old" {
		t.Fatalf("exe %q %v", data, err)
	}

	entries, _ := os.ReadDir(filepath.Dir(exe))
	if len(entries) != 1 {
		t.Fatalf("left behind %v", entries)
	}
}

func TestApplyReplaces(t *testing.T) {
	_, priv := newKey(t)

	t.Run("replaces the executable with the new binary", func(t *testing.T) {
		d := newDownloads(t)
		m := d.publish(t, priv, "1.2.0", buildTarGz(t, "transcript1", "new"))
		exe := newExe(t)
		if err := Apply(context.Background(), http.DefaultClient, m, exe); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(exe)
		if err != nil {
			t.Fatalf("exe %q %v", data, err)
		}

		if string(data) != "new" {
			t.Fatalf("exe %q %v", data, err)
		}

		info, _ := os.Stat(exe)
		if info.Mode().Perm() != 0o755 {
			t.Fatalf("mode %v", info.Mode())
		}
	})

	t.Run("keeps the old binary on a wrong sha-256", func(t *testing.T) {
		d := newDownloads(t)
		m := d.publish(t, priv, "1.2.0", buildTarGz(t, "transcript1", "new"))
		a := m.Assets[Platform()]
		a.SHA256 = strings.Repeat("0", 64)
		m.Assets[Platform()] = a
		exe := newExe(t)
		if err := Apply(context.Background(), http.DefaultClient, m, exe); err == nil {
			t.Fatal("no error")
		}

		assertOld(t, exe)
	})

	t.Run("refuses an archive larger than the manifest says", func(t *testing.T) {
		d := newDownloads(t)
		m := d.publish(t, priv, "1.2.0", buildTarGz(t, "transcript1", "new"))
		a := m.Assets[Platform()]
		a.Size--
		m.Assets[Platform()] = a
		exe := newExe(t)
		if err := Apply(context.Background(), http.DefaultClient, m, exe); err == nil {
			t.Fatal("no error")
		}

		assertOld(t, exe)
	})

	t.Run("refuses an entry outside the archive root", func(t *testing.T) {
		d := newDownloads(t)
		m := d.publish(t, priv, "1.2.0", buildTarGz(t, "../transcript1", "new"))
		exe := newExe(t)
		if err := Apply(context.Background(), http.DefaultClient, m, exe); err == nil {
			t.Fatal("no error")
		}

		assertOld(t, exe)
	})

	t.Run("reports a missing build for this platform", func(t *testing.T) {
		exe := newExe(t)
		m := &Manifest{Version: "1.2.0", Assets: map[string]Asset{}}
		err := Apply(context.Background(), http.DefaultClient, m, exe)
		if err == nil {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), Platform()) {
			t.Fatalf("err %v", err)
		}
	})

	t.Run("prints the install command for a folder it cannot write", func(t *testing.T) {
		d := newDownloads(t)
		m := d.publish(t, priv, "1.2.0", buildTarGz(t, "transcript1", "new"))
		exe := newExe(t)
		dir := filepath.Dir(exe)
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { os.Chmod(dir, 0o700) })
		err := Apply(context.Background(), http.DefaultClient, m, exe)
		if err == nil {
			t.Fatalf("err %v", err)
		}

		if !strings.Contains(err.Error(), "curl -fsSL https://transcript1.com/install.sh | sh") {
			t.Fatalf("err %v", err)
		}

		if d.requests.Load() != 0 {
			t.Fatalf("downloaded before checking the folder")
		}
	})
}

func TestCleanupOld(t *testing.T) {
	t.Run("removes the previous executable", func(t *testing.T) {
		exe := newExe(t)
		if err := os.WriteFile(exe+".old", []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}

		CleanupOld(exe)
		if _, err := os.Stat(exe + ".old"); !os.IsNotExist(err) {
			t.Fatalf("still there: %v", err)
		}
	})

	t.Run("does nothing when there is none", func(t *testing.T) {
		CleanupOld(newExe(t))
	})
}
