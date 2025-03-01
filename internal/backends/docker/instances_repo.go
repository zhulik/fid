package docker

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

// Key structure "<function-name>.<instance-uuid>"

type InstancesRepo struct {
	logger logrus.FieldLogger
	bucket core.KVBucket

	functionsRepo core.FunctionsRepo
}

func NewInstancesRepo(injector *do.Injector) (*InstancesRepo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	kv := do.MustInvoke[core.KV](injector)
	bucket := lo.Must(kv.Bucket(ctx, core.BucketNameInstances))

	functionsRepo := do.MustInvoke[core.FunctionsRepo](injector)

	return &InstancesRepo{
		logger: do.MustInvoke[logrus.FieldLogger](injector).
			WithField("component", "backends.docker.InstancesRepo"),
		bucket:        bucket,
		functionsRepo: functionsRepo,
	}, nil
}

func (r InstancesRepo) List(ctx context.Context, functionName string) ([]core.FunctionsInstance, error) {
	definition, err := r.functionsRepo.Get(ctx, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function definition: %w", err)
	}

	instances, err := r.bucket.All(ctx, key(functionName, "*"))
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	return lo.Map(instances, func(item core.KVEntry, _ int) core.FunctionsInstance {
		return NewFunctionInstance(item, definition)
	}), nil
}

func (r InstancesRepo) HealthCheck() error {
	return nil
}

func (r InstancesRepo) Shutdown() error {
	return nil
}

func (r InstancesRepo) Upsert(ctx context.Context, instance core.FunctionsInstance) error {
	err := r.bucket.Put(ctx, key(instance.Function().Name(), instance.ID()), serializeTime(instance.LastExecuted()))
	if err != nil {
		return fmt.Errorf("failed to store instance: %w", err)
	}

	return nil
}

func (r InstancesRepo) Get(ctx context.Context, id string) (core.FunctionsInstance, error) {
	list, err := r.bucket.All(ctx, key("*", id))
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return nil, core.ErrInstanceNotFound
		}

		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if len(list) == 0 {
		return nil, core.ErrInstanceNotFound
	}

	entry := list[0]

	functionName, _ := parseKey(entry.Key)

	definition, err := r.functionsRepo.Get(ctx, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get function definition: %w", err)
	}

	return NewFunctionInstance(entry, definition), nil
}

func (r InstancesRepo) Delete(ctx context.Context, id string) error {
	instance, err := r.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	err = r.bucket.Delete(ctx, key(instance.Function().Name(), id))
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return core.ErrInstanceNotFound
		}

		return fmt.Errorf("failed to delete instance: %w", err)
	}

	return nil
}

func key(functionName, instanceID string) string {
	return fmt.Sprintf("%s.%s", functionName, instanceID)
}

// parseKey parses a key of format <functionName>.<uuid> into function and instance UUID
// Panics if format is incorrect!
func parseKey(key string) (string, string) {
	parts := strings.Split(key, ".")
	if len(parts) != 2 { //nolint:mnd
		panic("incorrect key format")
	}

	return parts[0], parts[1]
}

// serializeTime packs time.Time to []byte as a unix timestamp in nanoseconds.
func serializeTime(t time.Time) []byte {
	nanos := uint64(t.UTC().UnixNano()) //nolint:gosec
	buf := make([]byte, 8)              //nolint:mnd
	binary.LittleEndian.PutUint64(buf, nanos)

	return buf
}

// deserializeTime extracts a unix timestamp in nanoseconds from []byte and returns it as time.Time.
func deserializeTime(data []byte) time.Time {
	nanos := int64(binary.LittleEndian.Uint64(data)) //nolint:gosec

	return time.Unix(0, nanos)
}
