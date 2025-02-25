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

func (k KV) Get(ctx context.Context, bucketName, key string) ([]byte, error) {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return bucket.Get(ctx, key) //nolint:wrapcheck
}

// All reads the entire bucket, make sure to use it only for small buckets.
func (k KV) All(ctx context.Context, bucketName string) ([]core.KVEntry, error) {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return bucket.All(ctx) //nolint:wrapcheck
}

func (k KV) AllFiltered(ctx context.Context, bucketName string, filters ...string) ([]core.KVEntry, error) {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return nil, err
	}

	return bucket.AllFiltered(ctx, filters...) //nolint:wrapcheck
}

func (k KV) Put(ctx context.Context, bucketName, key string, value []byte) error {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	return bucket.Put(ctx, key, value) //nolint:wrapcheck
}

func (k KV) Create(ctx context.Context, bucketName, key string, value []byte) (uint64, error) {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return 0, err
	}

	return bucket.Create(ctx, key, value) //nolint:wrapcheck
}

func (k KV) Update(ctx context.Context, bucketName, key string, value []byte, seq uint64) (uint64, error) {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return 0, err
	}

	return bucket.Update(ctx, key, value, seq) //nolint:wrapcheck
}

func (k KV) Incr(ctx context.Context, bucketName, key string, n int64) (int64, error) {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return 0, err
	}

	return bucket.Incr(ctx, key, n) //nolint:wrapcheck
}

func (k KV) Decr(ctx context.Context, bucket, key string, n int64) (int64, error) {
	return k.Incr(ctx, bucket, key, -n)
}

func (k KV) Delete(ctx context.Context, bucketName, key string) error {
	bucket, err := k.Bucket(ctx, bucketName)
	if err != nil {
		return err
	}

	return bucket.Delete(ctx, key) //nolint:wrapcheck
}

func (k KV) HealthCheck() error {
	return nil
}

func (k KV) Shutdown() error {
	return nil
}
