package api

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

const indexPlaceholder = "{index}"

var errNoChunkTemplate = errors.New("the upload plan has no chunk url template")

func (c *Client) ChunkURL(plan *UploadPlan, index int) (string, error) {
	if plan == nil {
		return "", errNoChunkTemplate
	}

	if !strings.Contains(plan.ChunkURLTemplate, indexPlaceholder) {
		return "", errNoChunkTemplate
	}

	chunkURL := strings.ReplaceAll(plan.ChunkURLTemplate, indexPlaceholder, strconv.Itoa(index))
	if !strings.HasPrefix(chunkURL, "/") {
		return chunkURL, nil
	}

	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return "", err
	}

	return base.Scheme + "://" + base.Host + chunkURL, nil
}
