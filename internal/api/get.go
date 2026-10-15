package api

import (
	"context"
	"net/url"
	"strconv"
)

func (c *Client) Get(ctx context.Context, id string, wait int) (*Transcription, error) {
	query := url.Values{}
	if wait > 0 {
		query.Set("wait", strconv.Itoa(wait))
	}

	res, err := c.get(ctx, "/v1/transcriptions/"+url.PathEscape(id), query, true)
	if err != nil {
		return nil, err
	}

	return decodeTranscription(res)
}
