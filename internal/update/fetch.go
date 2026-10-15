package update

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const errManifestTemplate = "latest.json: %w"

var errBadSignature = errors.New(
	"latest.json signature does not verify (a release may be in progress; try again in a few minutes)",
)

func isSignatureValid(pub ed25519.PublicKey, data, encoded []byte) bool {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return false
	}

	if len(sig) != ed25519.SignatureSize {
		return false
	}

	return ed25519.Verify(pub, data, sig)
}

// nothing is read before the signature verifies
func Fetch(ctx context.Context, hc *http.Client, base string, pub ed25519.PublicKey) (*Manifest, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, ErrNotConfigured
	}

	url := strings.TrimRight(base, "/") + manifestPath
	data, err := download(ctx, hc, url, maxManifest)
	if err != nil {
		return nil, err
	}

	encoded, err := download(ctx, hc, url+signatureExt, maxManifest)
	if err != nil {
		return nil, err
	}

	if !isSignatureValid(pub, data, encoded) {
		return nil, errBadSignature
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf(errManifestTemplate, err)
	}

	return &m, nil
}
