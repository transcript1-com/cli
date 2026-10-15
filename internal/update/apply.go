package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	errNoBuildTemplate     = "transcript1 %s has no build for %s"
	errNotWritableTemplate = "cannot replace transcript1 in %s: %w; reinstall with: %s"
)

var errChecksum = errors.New("downloaded archive does not match its sha-256")

func Apply(ctx context.Context, hc *http.Client, m *Manifest, exe string) (err error) {
	asset, ok := m.Assets[Platform()]
	if !ok {
		return fmt.Errorf(errNoBuildTemplate, m.Version, Platform())
	}

	if asset.Size <= 0 {
		return fmt.Errorf(errNoBuildTemplate, m.Version, Platform())
	}

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".transcript1-update-*")
	if err != nil {
		return buildNotWritableError(dir, err)
	}

	defer func() {
		tmp.Close()
		if err != nil {
			os.Remove(tmp.Name())
		}
	}()

	bin, err := fetchBinary(ctx, hc, asset)
	if err != nil {
		return err
	}

	if err := writeExecutable(tmp, bin); err != nil {
		return err
	}

	if err := replace(tmp.Name(), exe); err != nil {
		return buildNotWritableError(dir, err)
	}

	return nil
}

func fetchBinary(ctx context.Context, hc *http.Client, asset Asset) ([]byte, error) {
	archive, err := download(ctx, hc, asset.URL, asset.Size)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(archive)
	if hex.EncodeToString(sum[:]) != strings.ToLower(asset.SHA256) {
		return nil, errChecksum
	}

	return extract(archive)
}

func writeExecutable(f *os.File, bin []byte) error {
	if _, err := f.Write(bin); err != nil {
		return err
	}

	if err := f.Chmod(0o755); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	return f.Close()
}

func buildNotWritableError(dir string, err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}

	return fmt.Errorf(errNotWritableTemplate, dir, err, InstallCommand)
}
