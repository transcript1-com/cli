package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type LoginRequest struct {
	RequestID       string `json:"request_id"`
	UserCode        string `json:"user_code"`
	PollSecret      string `json:"poll_secret"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type LoginPoll struct {
	Status   string `json:"status"`
	Interval int    `json:"interval"`
	APIKey   string `json:"api_key"`
	TokenID  string `json:"token_id"`
	Email    string `json:"email"`
}

type createLoginBody struct {
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	CLIVersion string `json:"cli_version"`
}

type pollSecretBody struct {
	PollSecret string `json:"poll_secret"`
}

func (c *Client) CreateLoginRequest(ctx context.Context, hostname, goos, cliVersion string) (*LoginRequest, error) {
	body := createLoginBody{
		Hostname:   hostname,
		OS:         goos,
		CLIVersion: cliVersion,
	}
	res, err := c.postPublicJSON(ctx, "/cli/login-requests", body, false)
	if err != nil {
		return nil, err
	}

	login := &LoginRequest{}
	if err := json.Unmarshal(res.Body, login); err != nil {
		return nil, fmt.Errorf("the server sent an unreadable login request (HTTP %d): %w", res.Status, err)
	}

	return login, nil
}

func (c *Client) PollLoginRequest(ctx context.Context, id, secret string) (*LoginPoll, error) {
	path := "/cli/login-requests/" + url.PathEscape(id) + "/poll"
	res, err := c.postPublicJSON(ctx, path, pollSecretBody{PollSecret: secret}, false)
	if err != nil {
		return nil, err
	}

	poll := &LoginPoll{}
	if err := json.Unmarshal(res.Body, poll); err != nil {
		return nil, fmt.Errorf("the server sent an unreadable login reply (HTTP %d): %w", res.Status, err)
	}

	return poll, nil
}

func (c *Client) CancelLoginRequest(ctx context.Context, id, secret string) error {
	path := "/cli/login-requests/" + url.PathEscape(id) + "/cancel"
	_, err := c.postPublicJSON(ctx, path, pollSecretBody{PollSecret: secret}, false)

	return err
}

func (c *Client) RevokeToken(ctx context.Context, tokenID string) error {
	r := &request{
		method:  http.MethodDelete,
		url:     c.buildURL("/user/tokens/"+url.PathEscape(tokenID), nil),
		timeout: defaultTimeout,
	}
	_, err := c.do(ctx, r, true)

	return err
}
