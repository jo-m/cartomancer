// Package utl provides various utility functions.
package utl

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T {
	return &v
}

// Ignore discards the error and returns the value.
func Ignore[T any](val T, _ error) T {
	return val
}

// Must panics if the error is not nil, and otherwise returns the value.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

// DownloadFile fetches the resource at httpUrl via HTTP GET and returns the response body.
// Returns an error if the request fails or the server returns a non-2xx status code.
func DownloadFile(httpURL string) ([]byte, error) {
	resp, err := http.Get(httpURL) //nolint:gosec // URL is caller-supplied intentionally.
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, httpURL)
	}

	return io.ReadAll(resp.Body)
}

// CopyFile copies src to dst, overwriting dst if it exists.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
