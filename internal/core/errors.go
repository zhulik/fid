package core

import (
	"errors"
)

var (
	ErrFunctionNotFound = errors.New("function not found")
	ErrFunctionErrored  = errors.New("function returned an error")
)
