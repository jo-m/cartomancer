// Package utl provides various utility functions.
package utl

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
