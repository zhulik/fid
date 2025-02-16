package scaler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/elect"
)

type kvWrapper struct {
	kv core.KV

	bucket string

	ttl time.Duration
}

func (j kvWrapper) TTL() time.Duration {
	return j.ttl
}

func (j kvWrapper) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := j.kv.Get(ctx, j.bucket, key)
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return nil, elect.ErrKeyNotFound
		}

		return nil, fmt.Errorf("%w: failed to get key %s: %w", elect.ErrAnotherError, key, err)
	}

	return entry, nil
}

func (j kvWrapper) Create(ctx context.Context, key string, value []byte) (uint64, error) {
	seq, err := j.kv.Create(ctx, j.bucket, key, value)
	if err != nil {
		if errors.Is(err, core.ErrKeyExists) {
			return 0, elect.ErrKeyExists
		}

		return 0, fmt.Errorf("%w: failed to create key %s: %w", elect.ErrAnotherError, key, err)
	}

	return seq, nil
}

func (j kvWrapper) Update(ctx context.Context, key string, value []byte, seq uint64) (uint64, error) {
	seq, err := j.kv.Update(ctx, j.bucket, key, value, seq)
	if err != nil {
		if errors.Is(err, core.ErrKeyExists) {
			return 0, elect.ErrKeyExists
		}

		return 0, fmt.Errorf("%w: failed to update key %s: %w", elect.ErrAnotherError, key, err)
	}

	return seq, nil
}
