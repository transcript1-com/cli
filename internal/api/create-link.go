package api

import (
	"context"
	"net/url"
	"strconv"
)

type createLinkBody struct {
	URL      string `json:"url"`
	Source   string `json:"source"`
	Language string `json:"language"`
	Diarize  bool   `json:"diarize"`
}

func (c *Client) CreateLink(ctx context.Context, idemKey, link string, opt CreateOptions, wait int, reuse bool) (*Transcription, error) {
	query := url.Values{"wait": {strconv.Itoa(wait)}}
	if reuse {
		query.Set("reuse_recent", "true")
	}

	body := createLinkBody{
		URL:      link,
		Source:   opt.getSource(),
		Language: opt.getLanguage(),
		Diarize:  opt.Diarize,
	}
	res, err := c.postJSON(ctx, "/v1/transcriptions", query, idemKey, body)
	if err != nil {
		return nil, err
	}

	return decodeTranscription(res)
}
