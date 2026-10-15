package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/transcript1/cli/internal/update"
)

var errPackageUsage = errors.New("usage: release-tool package --version <x.y.z> --dist <dir> [--root <dir>]")

func pack(args []string, stdout io.Writer) error {
	fs := newFlags("package")
	version := fs.String("version", "", "")
	dist := fs.String("dist", "", "")
	root := fs.String("root", ".", "")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if !update.IsVersion(*version) {
		return errPackageUsage
	}

	if *dist == "" {
		return errPackageUsage
	}

	if fs.NArg() != 0 {
		return errPackageUsage
	}

	docs, err := readDocs(*root)
	if err != nil {
		return err
	}

	sums := map[string]string{}
	names := make([]string, 0, len(targets))
	for _, t := range targets {
		name := t.getArchive(*version)
		data, err := packTarget(*dist, t, docs)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(*dist, name), data, 0o644); err != nil {
			return err
		}

		sum := sha256.Sum256(data)
		sums[name] = hex.EncodeToString(sum[:])
		names = append(names, name)
		fmt.Fprintln(stdout, "wrote", name)
	}

	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		b.WriteString(sums[name] + "  " + name + "\n")
	}

	return os.WriteFile(filepath.Join(*dist, "checksums.txt"), []byte(b.String()), 0o644)
}

func readDocs(root string) ([]entry, error) {
	var docs []entry
	for _, name := range []string{"README.md", "LICENSE"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}

		doc := entry{
			name: name,
			mode: 0o644,
			data: data,
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

func packTarget(dist string, t target, docs []entry) ([]byte, error) {
	path := filepath.Join(dist, "bin", t.getKey(), t.getBinary())
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	bin, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	binEntry := entry{
		name: t.getBinary(),
		mode: 0o755,
		data: bin,
	}
	entries := append([]entry{binEntry}, docs...)
	mtime := info.ModTime().UTC().Truncate(time.Second)
	if t.os == "windows" {
		return buildZip(entries, mtime)
	}

	return buildTarGz(entries, mtime)
}
