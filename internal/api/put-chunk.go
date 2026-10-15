package api

import (
	"context"
	"io"
	"net/http"
)

func (c *Client) PutChunk(ctx context.Context, chunkURL, token string, body *io.SectionReader) error {
	r := &request{
		method: http.MethodPut,
		url:    chunkURL,
		header: http.Header{
			"Content-Type":   {"application/octet-stream"},
			"X-Upload-Token": {token},
		},
		// fresh reader per attempt, a canceled put may still read
		body:    func() (io.Reader, error) { return io.NewSectionReader(body, 0, body.Size()), nil },
		length:  body.Size(),
		timeout: chunkTimeout,
		noAuth:  true,
	}

	_, err := c.do(ctx, r, true)

	return err
}
