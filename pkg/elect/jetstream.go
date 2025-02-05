package elect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type JetStreamKV struct {
	KV jetstream.KeyValue

	Ttl time.Duration //nolint:stylecheck
}

func (j JetStreamKV) TTL() time.Duration {
	return j.Ttl
}

func (j JetStreamKV) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := j.KV.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, ErrKeyNotFound
		}

		return nil, fmt.Errorf("%w: failed to get key %s: %w", ErrAnotherError, key, err)
	}

	return entry.Value(), nil
}

func (j JetStreamKV) Create(ctx context.Context, key string, value []byte) (uint64, error) {
	seq, err := j.KV.Create(ctx, key, value)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyExists) {
			return 0, ErrKeyExists
		}

		return 0, fmt.Errorf("%w: failed to create key %s: %w", ErrAnotherError, key, err)
	}

	return seq, nil
}

func (j JetStreamKV) Update(ctx context.Context, key string, value []byte, seq uint64) (uint64, error) {
	seq, err := j.KV.Update(ctx, key, value, seq)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyExists) {
			return 0, ErrKeyExists
		}

		return 0, fmt.Errorf("%w: failed to update key %s: %w", ErrAnotherError, key, err)
	}

	return seq, nil
}
