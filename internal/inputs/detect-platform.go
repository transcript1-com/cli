package inputs

import (
	"net/url"
	"slices"
	"strings"
)

const PlatformYouTube = "youtube"

// same host lists as the api; paths are left to the api
var socialHosts = map[string]string{
	"x.com":               "x",
	"www.x.com":           "x",
	"mobile.x.com":        "x",
	"twitter.com":         "x",
	"www.twitter.com":     "x",
	"mobile.twitter.com":  "x",
	"tiktok.com":          "tiktok",
	"www.tiktok.com":      "tiktok",
	"m.tiktok.com":        "tiktok",
	"vm.tiktok.com":       "tiktok",
	"vt.tiktok.com":       "tiktok",
	"instagram.com":       "instagram",
	"www.instagram.com":   "instagram",
	"m.instagram.com":     "instagram",
	"facebook.com":        "facebook",
	"www.facebook.com":    "facebook",
	"m.facebook.com":      "facebook",
	"web.facebook.com":    "facebook",
	"mbasic.facebook.com": "facebook",
	"fb.watch":            "facebook",
	"bsky.app":            "bluesky",
}

func DetectPlatform(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}

	if !slices.Contains([]string{"http", "https"}, u.Scheme) {
		return "", false
	}

	host := strings.ToLower(u.Hostname())
	if _, ok := readYouTubeHost(host); ok {
		return PlatformYouTube, true
	}

	platform, ok := socialHosts[host]

	return platform, ok
}
