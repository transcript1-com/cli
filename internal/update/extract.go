package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"
)

const errUnsafeEntryTemplate = "archive entry %q is outside the archive root"

var (
	zipMagic       = []byte("PK\x03\x04")
	errNoBinary    = errors.New("archive has no transcript1 binary")
	errBinaryLarge = errors.New("binary in the archive is too large")
)

func extract(archive []byte) ([]byte, error) {
	if bytes.HasPrefix(archive, zipMagic) {
		return extractZip(archive)
	}

	return extractTarGz(archive)
}

func isBinaryEntry(regular bool, name string) bool {
	if !regular {
		return false
	}

	return trimEntryName(name) == getBinaryFile()
}

func extractTarGz(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, errNoBinary
		}

		if err != nil {
			return nil, err
		}

		if err := checkEntry(hdr.Name); err != nil {
			return nil, err
		}

		if isBinaryEntry(hdr.Typeflag == tar.TypeReg, hdr.Name) {
			return readCapped(tr)
		}
	}
}

func extractZip(archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}

	for _, f := range zr.File {
		if err := checkEntry(f.Name); err != nil {
			return nil, err
		}

		if !isBinaryEntry(f.Mode().IsRegular(), f.Name) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		defer rc.Close()

		return readCapped(rc)
	}

	return nil, errNoBinary
}

func readCapped(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBinary+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxBinary {
		return nil, errBinaryLarge
	}

	return data, nil
}

func isAbsolute(name string) bool {
	if strings.HasPrefix(name, "/") {
		return true
	}

	if strings.HasPrefix(name, `\`) {
		return true
	}

	return strings.Contains(name, ":")
}

func isSeparator(r rune) bool {
	return slices.Contains([]rune{'/', '\\'}, r)
}

func isUnsafeEntry(name string) bool {
	if name == "" {
		return true
	}

	if isAbsolute(name) {
		return true
	}

	return slices.Contains(strings.FieldsFunc(name, isSeparator), "..")
}

func checkEntry(name string) error {
	if isUnsafeEntry(name) {
		return fmt.Errorf(errUnsafeEntryTemplate, name)
	}

	return nil
}

func trimEntryName(name string) string {
	return strings.TrimPrefix(name, "./")
}

func getBinaryFile() string {
	if runtime.GOOS == "windows" {
		return binaryName + ".exe"
	}

	return binaryName
}
