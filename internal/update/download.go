package update

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/transcript1-com/cli/internal/version"
)

const (
	errStatusTemplate   = "download %s: HTTP %d"
	errTooLargeTemplate = "download %s: larger than %d bytes"
	errReadTemplate     = "download %s: %w"
)

func download(ctx context.Context, hc *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(userAgent, version.UserAgent())
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(errStatusTemplate, url, resp.StatusCode)
	}

	if resp.ContentLength > limit {
		return nil, fmt.Errorf(errTooLargeTemplate, url, limit)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf(errReadTemplate, url, err)
	}

	if int64(len(data)) > limit {
		return nil, fmt.Errorf(errTooLargeTemplate, url, limit)
	}

	return data, nil
}
