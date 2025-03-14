package db

import "fmt"

func EnsureOneRowChanged(n int64, err error) error {
	if err != nil {
		return err
	}

	if n != 1 {
		return fmt.Errorf("expected exactly one changed row, but got %d", n)
	}

	return nil
}
