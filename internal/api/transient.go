package api

import (
	"errors"
	"net/url"
)

func Transient(err error) bool {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return isRetryableStatus(apiErr.Status)
	}

	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
