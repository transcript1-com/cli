package api

import (
	"context"
	"net/url"
)

func hasTimestamps(format string, timestamps bool) bool {
	if format != "text" {
		return false
	}

	return timestamps
}

func (c *Client) Render(ctx context.Context, id, format string, timestamps bool) ([]byte, error) {
	query := url.Values{"format": {format}}
	if hasTimestamps(format, timestamps) {
		query.Set("timestamps", "true")
	}

	res, err := c.get(ctx, "/v1/transcriptions/"+url.PathEscape(id), query, true)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}
