package inputs

import "fmt"

func newLink(raw string) (Input, error) {
	id, canonical, ok := ParseYouTube(raw)
	if !ok {
		return Input{}, fmt.Errorf("%s: Only YouTube links and files are supported for now", raw)
	}

	in := Input{
		Raw:     raw,
		Kind:    Link,
		URL:     canonical,
		VideoID: id,
	}

	return in, nil
}
