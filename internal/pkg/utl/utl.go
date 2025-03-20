package utl

func Ptr[T any](v T) *T {
	return &v
}

func Ignore[T any](val T, _err error) T {
	return val
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
