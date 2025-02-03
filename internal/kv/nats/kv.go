package nats

import (
	"context"
	"errors"
	"fmt"

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
		nats: nats,
	}, nil
}

type KV struct {
	nats *pubsubNats.Client
}

func (k KV) CreateBucket(ctx context.Context, name string) error {
	_, err := k.nats.JetStream.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: name,
	})
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketExists) {
			return fmt.Errorf("%w: %w", core.ErrBucketExists, err)
		}

		return fmt.Errorf("failed to create bucket: %w", err)
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

func (k KV) Create(ctx context.Context, bucket, key string, value []byte) error {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return err
	}

	_, err = kv.Create(ctx, key, value)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyExists) {
			return fmt.Errorf("%w: %w", core.ErrKeyExists, err)
		}

		return fmt.Errorf("failed to put value: %w", err)
	}

	return nil
}

func (k KV) WaitCreated(ctx context.Context, bucket, key string) ([]byte, error) {
	kv, err := k.getBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}

	watch, err := kv.Watch(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to watch value: %w", err)
	}

	err = watch.Stop()
	if err != nil {
		return nil, fmt.Errorf("failed to stop watch: %w", err)
	}

	select {
	case entry := <-watch.Updates():
		if entry.Operation() == jetstream.KeyValuePut && entry.Revision() == 0 {
			return entry.Value(), nil
		} else {
			return nil, core.ErrWrongOperation
		}

	case <-ctx.Done():
		return nil, fmt.Errorf("context done: %w", ctx.Err())
	}
}

func (k KV) Delete(ctx context.Context, bucket, key string) error {
	kv, err := k.nats.JetStream.KeyValue(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %w", err)
	}

	err = kv.Delete(ctx, key)
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

func (k KV) getBucket(ctx context.Context, bucket string) (jetstream.KeyValue, error) { //nolint:ireturn
	kv, err := k.nats.JetStream.KeyValue(ctx, bucket)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return nil, fmt.Errorf("%w: %w", core.ErrBucketNotFound, err)
		}

		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return kv, nil
}
