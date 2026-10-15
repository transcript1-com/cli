package api

import (
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) Account(ctx context.Context) (*Account, error) {
	res, err := c.get(ctx, "/v1/account", nil, true)
	if err != nil {
		return nil, err
	}

	account := &Account{}
	if err := json.Unmarshal(res.Body, account); err != nil {
		return nil, fmt.Errorf("the server sent an unreadable account (HTTP %d): %w", res.Status, err)
	}

	return account, nil
}
