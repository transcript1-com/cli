package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/transcript1-com/cli/internal/version"
)

const (
	maxAttempts    = 3
	connectTimeout = 30 * time.Second
	waitTimeout    = 90 * time.Second
	chunkTimeout   = 5 * time.Minute
	defaultTimeout = 60 * time.Second
	// slot waits use the full retry-after, plain retries cap here
	maxRetryAfter = 60 * time.Second
)

type Client struct {
	BaseURL   string
	Key       string
	UserAgent string
	HTTP      *http.Client
	Sleep     func(context.Context, time.Duration) error
}

type request struct {
	method  string
	url     string
	header  http.Header
	body    func() (io.Reader, error)
	length  int64
	timeout time.Duration
	noAuth  bool
}

type response struct {
	Status int
	Header http.Header
	Body   []byte
}

func New(baseURL, key string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := &net.Dialer{
		Timeout:   connectTimeout,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = dialer.DialContext

	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		Key:       key,
		UserAgent: version.UserAgent(),
		HTTP:      &http.Client{Transport: transport},
		Sleep:     sleep,
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) do(ctx context.Context, r *request, retry429 bool) (*response, error) {
	for attempt := 1; ; attempt++ {
		res, err := c.attempt(ctx, r)
		if isSuccess(res, err) {
			return res, nil
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if !shouldRetry(res, err, retry429) {
			return nil, buildAttemptError(res, err)
		}

		if attempt == maxAttempts {
			return nil, buildAttemptError(res, err)
		}

		if err := c.Sleep(ctx, getRetryDelay(attempt, res)); err != nil {
			return nil, err
		}
	}
}

func isSuccess(res *response, err error) bool {
	if err != nil {
		return false
	}

	return res.Status < http.StatusBadRequest
}

func shouldRetry(res *response, err error, retry429 bool) bool {
	if err != nil {
		return true
	}

	if !isRetryableStatus(res.Status) {
		return false
	}

	if res.Status == http.StatusTooManyRequests {
		return retry429
	}

	return true
}

func buildAttemptError(res *response, err error) error {
	if err != nil {
		return err
	}

	return buildResponseError(res)
}

func getRetryDelay(attempt int, res *response) time.Duration {
	if res != nil {
		if wait := parseRetryAfter(res.Header); wait > 0 {
			return min(wait, maxRetryAfter)
		}
	}

	base := time.Second << (attempt - 1)

	return base + time.Duration(rand.Int64N(int64(base/4)))
}

func (c *Client) attempt(ctx context.Context, r *request) (*response, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	req, err := c.build(ctx, r)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	resp := &response{
		Status: res.StatusCode,
		Header: res.Header,
		Body:   body,
	}

	return resp, nil
}

func (c *Client) build(ctx context.Context, r *request) (*http.Request, error) {
	var body io.Reader
	if r.body != nil {
		var err error
		if body, err = r.body(); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, r.method, r.url, body)
	if err != nil {
		return nil, err
	}

	for name, values := range r.header {
		req.Header[name] = values
	}

	req.Header.Set("User-Agent", c.UserAgent)
	if !r.noAuth {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}

	if body != nil {
		req.ContentLength = r.length
		if r.length == 0 {
			req.Body = http.NoBody
		}
	}

	return req, nil
}

func (c *Client) buildURL(path string, query url.Values) string {
	target := c.BaseURL + path
	if len(query) == 0 {
		return target
	}

	return target + "?" + query.Encode()
}

func getTimeout(query url.Values) time.Duration {
	if wait, _ := strconv.Atoi(query.Get("wait")); wait > 0 {
		return waitTimeout
	}

	return defaultTimeout
}

func (c *Client) get(ctx context.Context, path string, query url.Values, retry429 bool) (*response, error) {
	r := &request{
		method:  http.MethodGet,
		url:     c.buildURL(path, query),
		timeout: getTimeout(query),
	}

	return c.do(ctx, r, retry429)
}

func (c *Client) postJSON(ctx context.Context, path string, query url.Values, idemKey string, payload any) (*response, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	r := &request{
		method: http.MethodPost,
		url:    c.buildURL(path, query),
		header: http.Header{
			"Content-Type":    {"application/json"},
			"Idempotency-Key": {idemKey},
		},
		body:    func() (io.Reader, error) { return bytes.NewReader(raw), nil },
		length:  int64(len(raw)),
		timeout: getTimeout(query),
	}

	return c.do(ctx, r, false)
}

func (c *Client) postPublicJSON(ctx context.Context, path string, payload any, retry429 bool) (*response, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	r := &request{
		method:  http.MethodPost,
		url:     c.buildURL(path, nil),
		header:  http.Header{"Content-Type": {"application/json"}},
		body:    func() (io.Reader, error) { return bytes.NewReader(raw), nil },
		length:  int64(len(raw)),
		timeout: defaultTimeout,
		noAuth:  true,
	}

	return c.do(ctx, r, retry429)
}

func decodeTranscription(res *response) (*Transcription, error) {
	tr := &Transcription{}
	if err := json.Unmarshal(res.Body, tr); err != nil {
		return nil, fmt.Errorf("the server sent an unreadable transcription (HTTP %d): %w", res.Status, err)
	}

	tr.Raw = json.RawMessage(res.Body)

	return tr, nil
}
