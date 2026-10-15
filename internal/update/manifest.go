package update

import "errors"

const (
	InstallCommand = "curl -fsSL https://transcript1.com/install.sh | sh"

	binaryName   = "transcript1"
	manifestPath = "/cli/latest.json"
	signatureExt = ".sig"
	maxManifest  = 1 << 20
	maxBinary    = 512 << 20
	userAgent    = "User-Agent"
)

var ErrNotConfigured = errors.New("updates are not configured in this build")

// fields in key order so latest.json gets sorted keys
type Manifest struct {
	Assets      map[string]Asset `json:"assets"`
	PublishedAt string           `json:"published_at"`
	Version     string           `json:"version"`
}

type Asset struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	URL    string `json:"url"`
}
