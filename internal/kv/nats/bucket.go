package nats

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/zhulik/fid/internal/core"
)

type Bucket struct {
	bucket jetstream.KeyValue
}

func (b Bucket) All(ctx context.Context) ([]core.KVEntry, error) {
	lister, err := b.bucket.ListKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	return b.listKeys(ctx, lister)
}

func (b Bucket) AllFiltered(ctx context.Context, filters ...string) ([]core.KVEntry, error) {
	lister, err := b.bucket.ListKeysFiltered(ctx, filters...)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	return b.listKeys(ctx, lister)
}

func (b Bucket) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := b.bucket.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, fmt.Errorf("%w: %w", core.ErrKeyNotFound, err)
		}

		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return entry.Value(), nil
}

func (b Bucket) Create(ctx context.Context, key string, value []byte) (uint64, error) {
	seq, err := b.bucket.Create(ctx, key, value)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyExists) {
			return 0, fmt.Errorf("%w: %w", core.ErrKeyExists, err)
		}

		return 0, fmt.Errorf("failed to put value: %w", err)
	}

	return seq, nil
}

func (b Bucket) Put(ctx context.Context, key string, value []byte) error {
	_, err := b.bucket.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to put value: %w", err)
	}

	return nil
}

func (b Bucket) Update(ctx context.Context, key string, value []byte, seq uint64) (uint64, error) {
	seq, err := b.bucket.Update(ctx, key, value, seq)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return 0, fmt.Errorf("%w: %w", core.ErrKeyNotFound, err)
		}

		return 0, fmt.Errorf("failed to put value: %w", err)
	}

	return seq, nil
}

func (b Bucket) Delete(ctx context.Context, key string) error {
	err := b.bucket.Purge(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}

func (b Bucket) Incr(ctx context.Context, key string, n int64) (int64, error) { //nolint:cyclop
	for {
		// Try to get the value
		entry, err := b.bucket.Get(ctx, key)
		if err != nil { //nolint:nestif
			if errors.Is(err, jetstream.ErrKeyNotFound) {
				// It does not exist, create it.
				_, err := b.bucket.Create(ctx, key, []byte(strconv.FormatInt(n, 10)))
				if err != nil {
					// Value was created concurrently, try again.
					if errors.Is(err, jetstream.ErrKeyExists) {
						continue
					}

					return 0, fmt.Errorf("failed to create value: %w", err)
				}

				return n, nil
			}

			return 0, fmt.Errorf("failed to get value: %w", err)
		}

		value, err := strconv.ParseInt(string(entry.Value()), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse counter value: %w", err)
		}

		value += n

		// Try updating the value
		_, err = b.bucket.Update(ctx, key, []byte(strconv.FormatInt(value, 10)), entry.Revision())
		if err == nil {
			return value, nil
		}

		if errors.Is(err, jetstream.ErrKeyExists) ||
			errors.Is(err, jetstream.ErrKeyDeleted) ||
			errors.Is(err, jetstream.ErrKeyNotFound) {
			// It was updated or delete concurrently, try again.
			continue
		}

		return 0, fmt.Errorf("failed to update value: %w", err)
	}
}

func (b Bucket) Decr(ctx context.Context, key string, n int64) (int64, error) {
	return b.Incr(ctx, key, -n)
}

func (b Bucket) Name() string {
	return b.bucket.Bucket()
}

func (b Bucket) listKeys(ctx context.Context, lister jetstream.KeyLister) ([]core.KVEntry, error) {
	var entries []core.KVEntry //nolint:prealloc

	for key := range lister.Keys() {
		value, err := b.Get(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get value: %w", err)
		}

		entries = append(entries, core.KVEntry{
			Key:   key,
			Value: value,
		})
	}

	return entries, nil
}
