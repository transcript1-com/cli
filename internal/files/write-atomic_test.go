package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomic(t *testing.T) {
	t.Run("creates missing parent folders", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "a", "b", "t.txt")
		if err := WriteAtomic(path, []byte("hello")); err != nil {
			t.Fatal(err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("got %q, %v", got, err)
		}

		if string(got) != "hello" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	t.Run("replaces an existing file", func(t *testing.T) {
		path := touch(t, t.TempDir(), "t.txt")
		if err := WriteAtomic(path, []byte("new")); err != nil {
			t.Fatal(err)
		}

		got, _ := os.ReadFile(path)
		if string(got) != "new" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("leaves no temporary file", func(t *testing.T) {
		dir := t.TempDir()
		if err := WriteAtomic(filepath.Join(dir, "t.txt"), []byte("x")); err != nil {
			t.Fatal(err)
		}

		entries, _ := os.ReadDir(dir)
		if len(entries) != 1 {
			t.Fatalf("got %v", entries)
		}

		if entries[0].Name() != "t.txt" {
			t.Fatalf("got %v", entries)
		}
	})

	t.Run("makes a readable file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "t.txt")
		if err := WriteAtomic(path, []byte("x")); err != nil {
			t.Fatal(err)
		}

		info, _ := os.Stat(path)
		if info.Mode().Perm()&0o044 == 0 {
			t.Fatalf("mode %v", info.Mode())
		}
	})

	t.Run("names the path when the folder is read-only", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can write anywhere")
		}

		dir := t.TempDir()
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { os.Chmod(dir, 0o755) })
		path := filepath.Join(dir, "t.txt")
		err := WriteAtomic(path, []byte("x"))
		if err == nil {
			t.Fatalf("got %v", err)
		}

		if !strings.Contains(err.Error(), path) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("names the path when a parent cannot be created", func(t *testing.T) {
		blocker := touch(t, t.TempDir(), "file")
		path := filepath.Join(blocker, "sub", "t.txt")
		err := WriteAtomic(path, []byte("x"))
		if err == nil {
			t.Fatalf("got %v", err)
		}

		if !strings.Contains(err.Error(), path) {
			t.Fatalf("got %v", err)
		}
	})
}
