package scaler

import (
	"context"
	"errors"
	"fmt"

	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/elect"
)

type kvWrapper struct {
	bucket core.KVBucket
}

func (j kvWrapper) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := j.bucket.Get(ctx, key)
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return nil, elect.ErrKeyNotFound
		}

		return nil, fmt.Errorf("%w: failed to get key %s: %w", elect.ErrAnotherError, key, err)
	}

	return entry, nil
}

func (j kvWrapper) Create(ctx context.Context, key string, value []byte) (uint64, error) {
	seq, err := j.bucket.Create(ctx, key, value)
	if err != nil {
		if errors.Is(err, core.ErrKeyExists) {
			return 0, elect.ErrKeyExists
		}

		return 0, fmt.Errorf("%w: failed to create key %s: %w", elect.ErrAnotherError, key, err)
	}

	return seq, nil
}

func (j kvWrapper) Update(ctx context.Context, key string, value []byte, seq uint64) (uint64, error) {
	seq, err := j.bucket.Update(ctx, key, value, seq)
	if err != nil {
		if errors.Is(err, core.ErrKeyExists) {
			return 0, elect.ErrKeyExists
		}

		return 0, fmt.Errorf("%w: failed to update key %s: %w", elect.ErrAnotherError, key, err)
	}

	return seq, nil
}
