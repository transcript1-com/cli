package config

const (
	EnvAPIKey        = "TRANSCRIPT1_API_KEY"
	EnvAPIURL        = "TRANSCRIPT1_API_URL"
	EnvDownloadsURL  = "TRANSCRIPT1_DOWNLOADS_URL"
	EnvNoUpdateCheck = "TRANSCRIPT1_NO_UPDATE_CHECK"

	DefaultAPIURL       = "https://api.transcript1.com"
	DefaultDownloadsURL = "https://downloads.transcript1.com"

	dirName  = "transcript1"
	fileName = "config.json"
)

type KeyFile struct {
	APIKey  string `json:"api_key"`
	TokenID string `json:"token_id,omitempty"`
}
