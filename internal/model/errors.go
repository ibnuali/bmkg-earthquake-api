package model

import "errors"

var (
	ErrNotFound       = errors.New("data not found")
	ErrUpstream       = errors.New("upstream service error")
	ErrRateLimited    = errors.New("rate limit exceeded")
	ErrInvalidRequest = errors.New("invalid request")
)
