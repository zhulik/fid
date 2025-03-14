package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
	pubsubNats "github.com/zhulik/fid/internal/pubsub/nats"
)

func NewKV(injector do.Injector) (*KV, error) {
	return &KV{
		Nats: do.MustInvoke[*pubsubNats.Client](injector),
	}, nil
}

type KV struct {
	Nats *pubsubNats.Client
}

func (k KV) CreateBucket(ctx context.Context, name string, ttl time.Duration) (core.KVBucket, error) {
	bucket, err := k.Nats.JetStream.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: name,
		TTL:    ttl,
	})
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketExists) {
			return nil, fmt.Errorf("%w: %w", core.ErrBucketExists, err)
		}

		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return Bucket{bucket: bucket}, nil
}

func (k KV) Bucket(ctx context.Context, name string) (core.KVBucket, error) {
	bucket, err := k.Nats.JetStream.KeyValue(ctx, name)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			return nil, fmt.Errorf("%w: %w", core.ErrBucketNotFound, err)
		}

		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return Bucket{bucket: bucket}, nil
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

func (k KV) HealthCheck() error {
	return nil
}

func (k KV) Shutdown() error {
	return nil
}
