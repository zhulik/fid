package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
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
		// TODO: if already exists, use a custom error
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

func (k KV) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	kv, err := k.nats.JetStream.KeyValue(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	entry, err := kv.Get(ctx, key)
	if err != nil {
		// TODO: if not found, use a custom error
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	return entry.Value(), nil
}

func (k KV) Put(ctx context.Context, bucket, key string, value []byte) error {
	kv, err := k.nats.JetStream.KeyValue(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %w", err)
	}

	_, err = kv.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to put value: %w", err)
	}

	return nil
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
