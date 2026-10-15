package api

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) CompleteUpload(ctx context.Context, id string) (*Transcription, error) {
	r := &request{
		method:  http.MethodPost,
		url:     c.buildURL("/v1/transcriptions/"+url.PathEscape(id)+"/upload/complete", nil),
		timeout: defaultTimeout,
	}
	res, err := c.do(ctx, r, true)
	if err != nil {
		return nil, err
	}

	return decodeTranscription(res)
}
