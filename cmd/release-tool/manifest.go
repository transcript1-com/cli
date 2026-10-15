package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/transcript1/cli/internal/update"
)

var errManifestUsage = errors.New(
	"usage: release-tool manifest --version <x.y.z> --dist <dir> --base-url <url> --key <file> [--pub <file>]")

func writeManifest(args []string, stdout io.Writer) error {
	fs := newFlags("manifest")
	version := fs.String("version", "", "")
	dist := fs.String("dist", "", "")
	base := fs.String("base-url", "", "")
	keyPath := fs.String("key", "", "")
	pubPath := fs.String("pub", defaultPub, "")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if !update.IsVersion(*version) {
		return errManifestUsage
	}

	if *dist == "" {
		return errManifestUsage
	}

	if *base == "" {
		return errManifestUsage
	}

	if *keyPath == "" {
		return errManifestUsage
	}

	if fs.NArg() != 0 {
		return errManifestUsage
	}

	priv, err := loadMatchingKey(*keyPath, *pubPath)
	if err != nil {
		return err
	}

	m, err := buildManifest(*version, *dist, strings.TrimRight(*base, "/"))
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(*dist, "latest.json"), data, 0o644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(*dist, "latest.json.sig"), sign(priv, data), 0o644); err != nil {
		return err
	}

	fmt.Fprintln(stdout, "wrote latest.json and latest.json.sig for", *version)

	return nil
}

func buildManifest(version, dist, base string) (*update.Manifest, error) {
	m := &update.Manifest{
		Version:     version,
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
		Assets:      map[string]update.Asset{},
	}
	for _, t := range targets {
		name := t.getArchive(version)
		data, err := os.ReadFile(filepath.Join(dist, name))
		if err != nil {
			return nil, err
		}

		sum := sha256.Sum256(data)
		m.Assets[t.getKey()] = update.Asset{
			URL:    base + "/cli/releases/" + version + "/" + name,
			SHA256: hex.EncodeToString(sum[:]),
			Size:   int64(len(data)),
		}
	}

	return m, nil
}
