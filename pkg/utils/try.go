package utils

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
)

var ErrPanicked = errors.New("panic")

func Try[T any](fun func() (T, error)) (res T, err error) { //nolint:nonamedreturns
	defer func() {
		if r := recover(); r != nil {
			res = lo.Empty[T]()
			err = fmt.Errorf("%w: %v", ErrPanicked, r)
		}
	}()

	return fun()
}
