// Package client provides a pre-configured HTTP client for external requests.
//
// All code that makes outbound HTTP calls should use [New] instead of
// http.DefaultClient so that every request has a reasonable timeout and does
// not block indefinitely when an upstream server is unresponsive.
package client

import (
	"net/http"
	"time"
)

const (
	// defaultTimeout is the end-to-end timeout applied to every request
	// performed by the client returned from [New].
	defaultTimeout = 2 * time.Minute
)

// New returns an *http.Client with sensible defaults for outbound requests.
// The returned client enforces an end-to-end timeout so that a hanging
// upstream cannot block a goroutine indefinitely.
func New() *http.Client {
	return &http.Client{
		Timeout: defaultTimeout,
	}
}
