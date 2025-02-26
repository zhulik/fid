package elect

import (
	"context"
	"errors"
)

var (
	ErrKeyExists    = errors.New("key already exists")
	ErrKeyNotFound  = errors.New("key not found")
	ErrAnotherError = errors.New("another error")
)

type KV interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Create(ctx context.Context, key string, value []byte) (uint64, error)
	Update(ctx context.Context, key string, value []byte, seq uint64) (uint64, error)
}
