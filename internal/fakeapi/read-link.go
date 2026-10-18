package fakeapi

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/transcript1-com/cli/internal/api"
	"github.com/transcript1-com/cli/internal/inputs"
)

const msgUnsupportedLink = "Not a supported link. Use a YouTube, X, TikTok, Instagram, Facebook or Bluesky link."

var (
	platformNames = map[string]string{
		"youtube":   "YouTube",
		"x":         "X",
		"tiktok":    "TikTok",
		"instagram": "Instagram",
		"facebook":  "Facebook",
		"bluesky":   "Bluesky",
	}
	platformIDs = []string{"youtube", "x", "tiktok", "instagram", "facebook", "bluesky"}
	digitsOnly  = regexp.MustCompile(`^[0-9]+$`)
)

func (s *Server) checkPlatform(w http.ResponseWriter, body createBody) (string, bool) {
	detected, ok := inputs.DetectPlatform(body.URL)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidURL, msgUnsupportedLink, nil)
		return "", false
	}

	if body.Platform == "" {
		return detected, true
	}

	if _, known := platformNames[body.Platform]; !known {
		writeError(w, http.StatusUnprocessableEntity, api.CodeUnsupportedPlatform, "Unsupported platform",
			map[string]any{"supported_platforms": platformIDs})
		return "", false
	}

	if body.Platform != detected {
		message := "This link is from " + platformNames[detected] + ", not " + platformNames[body.Platform] + "."
		writeError(w, http.StatusUnprocessableEntity, api.CodeInvalidURL, message, nil)
		return "", false
	}

	return detected, true
}

// the fake keeps the link as sent and takes a numeric last segment as the media id
func readLink(platform, raw string) (link, mediaID string, ok bool) {
	if platform == inputs.PlatformYouTube {
		id, canonical, found := inputs.ParseYouTube(raw)
		return canonical, id, found
	}

	link = strings.TrimSpace(raw)
	u, err := url.Parse(link)
	if err != nil {
		return "", "", false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	last := segments[len(segments)-1]
	if digitsOnly.MatchString(last) {
		return link, last, true
	}

	return link, "", true
}
