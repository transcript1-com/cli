package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *Client) UploadProgress(ctx context.Context, id string) (*UploadProgress, error) {
	res, err := c.get(ctx, "/v1/transcriptions/"+url.PathEscape(id)+"/upload", nil, true)
	if err != nil {
		return nil, err
	}

	progress := &UploadProgress{}
	if err := json.Unmarshal(res.Body, progress); err != nil {
		return nil, fmt.Errorf("the server sent unreadable upload progress (HTTP %d): %w", res.Status, err)
	}

	return progress, nil
}
