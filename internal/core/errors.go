package core

import (
	"errors"
)

var (
	// General errors.
	ErrContainerAlreadyExists = errors.New("container already exists")

	// Function errors.
	ErrFunctionNotFound     = errors.New("function not found")
	ErrFunctionErrored      = errors.New("function returned an error")
	ErrFunctionNameNotGiven = errors.New("function name is not provided as env FUNCTION_NAME")

	ErrInstanceNotFound      = errors.New("function instance not found")
	ErrInstanceAlreadyExists = errors.New("function instance already exists")

	// KV errors.
	ErrKeyNotFound    = errors.New("key not found")
	ErrBucketNotFound = errors.New("bucket not found")
	ErrBucketExists   = errors.New("bucket already exists")
	ErrKeyExists      = errors.New("key already exists")
	ErrWrongOperation = errors.New("wrong operation or revision")
)
