package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/zhulik/fid/internal/core"
	pubsubNats "github.com/zhulik/fid/internal/pubsub/nats"
)

func NewKV(injector *do.Injector) (*KV, error) {
	nats, err := do.Invoke[*pubsubNats.Client](injector)
	if err != nil {
		return nil, err
	}

	return &KV{
		Nats: nats,
	}, nil
}

type KV struct {
	Nats *pubsubNats.Client
}

func (k KV) CreateBucket(ctx context.Context, name string, ttl time.Duration) error {
	_, err := k.Nats.JetStream.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: name,
		TTL:    ttl,
	})
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketExists) {
			return fmt.Errorf("%w: %w", core.ErrBucketExists, err)
		}

		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

func (k KV) DeleteBucket(ctx context.Context, name string) error {
	err := k.Nats.JetStream.DeleteKeyValue(ctx, name)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return fmt.Errorf("%w: %w", core.ErrBucketNotFound, err)
		}

		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

func (k KV) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}

	entry, err := kv.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, fmt.Errorf("%w: %w", core.ErrKeyNotFound, err)
		}

		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return entry.Value(), nil
}

func (k KV) Put(ctx context.Context, bucket, key string, value []byte) error {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	_, err = kv.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to put value: %w", err)
	}

	return nil
}

func (k KV) Create(ctx context.Context, bucket, key string, value []byte) (uint64, error) {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return 0, err
	}

	seq, err := kv.Create(ctx, key, value)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyExists) {
			return 0, fmt.Errorf("%w: %w", core.ErrKeyExists, err)
		}

		return 0, fmt.Errorf("failed to put value: %w", err)
	}

	return seq, nil
}

func (k KV) Update(ctx context.Context, bucket, key string, value []byte, seq uint64) (uint64, error) {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return 0, err
	}

	seq, err = kv.Update(ctx, key, value, seq)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return 0, fmt.Errorf("%w: %w", core.ErrKeyNotFound, err)
		}

		return 0, fmt.Errorf("failed to put value: %w", err)
	}

	return seq, nil
}

func (k KV) WaitCreated(ctx context.Context, bucket, key string) ([]byte, error) {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}

	value, err := k.Get(ctx, bucket, key)
	if err == nil {
		return value, nil
	}

	if !errors.Is(err, core.ErrKeyNotFound) {
		return nil, err
	}

	watch, err := kv.Watch(ctx, key, jetstream.UpdatesOnly(), jetstream.IgnoreDeletes())
	if err != nil {
		return nil, fmt.Errorf("failed to watch value: %w", err)
	}

	defer watch.Stop() //nolint:errcheck

	updates := watch.Updates()

	for {
		select {
		case entry := <-updates:
			return entry.Value(), nil

		case <-ctx.Done():
			return nil, fmt.Errorf("context done: %w", ctx.Err())
		}
	}
}

func (k KV) Delete(ctx context.Context, bucket, key string) error {
	kv, err := k.Nats.JetStream.KeyValue(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %w", err)
	}

	err = kv.Purge(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	return nil
}

func (k KV) HealthCheck() error {
	return nil
}

func (k KV) Shutdown() error {
	return nil
}

func (k KV) getBucket(ctx context.Context, bucket string) (jetstream.KeyValue, error) {
	kv, err := k.Nats.JetStream.KeyValue(ctx, bucket)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return nil, fmt.Errorf("%w: %w", core.ErrBucketNotFound, err)
		}

		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return kv, nil
}
