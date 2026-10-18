package inputs

import (
	"fmt"
	"strings"
)

const errUnsupportedTemplate = "%s: Only YouTube, X, TikTok, Instagram, Facebook and Bluesky links and files are supported"

func newLink(raw string) (Input, error) {
	link := strings.TrimSpace(raw)
	platform, ok := DetectPlatform(link)
	if !ok {
		return Input{}, fmt.Errorf(errUnsupportedTemplate, raw)
	}

	in := Input{
		Raw:      raw,
		Kind:     Link,
		Platform: platform,
		URL:      link,
	}
	if platform != PlatformYouTube {
		return in, nil
	}

	id, canonical, ok := ParseYouTube(link)
	if !ok {
		return Input{}, fmt.Errorf(errUnsupportedTemplate, raw)
	}

	in.URL = canonical
	in.VideoID = id

	return in, nil
}
