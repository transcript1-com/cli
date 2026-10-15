package api

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) Delete(ctx context.Context, id string) error {
	r := &request{
		method:  http.MethodDelete,
		url:     c.buildURL("/v1/transcriptions/"+url.PathEscape(id), nil),
		timeout: defaultTimeout,
	}
	_, err := c.do(ctx, r, true)

	return err
}
