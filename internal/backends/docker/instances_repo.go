package docker

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

// Key structure "<function-name>.<instance-uuid>"

type InstancesRepo struct {
	logger logrus.FieldLogger
	bucket core.KVBucket
}

func NewInstancesRepo(ctx context.Context, injector do.Injector) (*InstancesRepo, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	kv := do.MustInvoke[core.KV](injector)
	bucket := lo.Must(kv.Bucket(ctx, core.BucketNameInstances))

	return &InstancesRepo{
		logger: do.MustInvoke[logrus.FieldLogger](injector).
			WithField("component", "backends.docker.InstancesRepo"),
		bucket: bucket,
	}, nil
}

func (r InstancesRepo) List(ctx context.Context, function core.FunctionDefinition) ([]core.FunctionInstance, error) {
	list, err := r.bucket.All(ctx, allKeys(function.Name(), "*"))
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	grouped := lo.GroupBy(list, func(item core.KVEntry) string {
		_, id := parseKey(item.Key)

		return id
	})

	instances := make([]core.FunctionInstance, 0, len(grouped))

	for id, items := range grouped {
		instances = append(instances, NewFunctionInstance(id, function, groupByKey(items)))
	}

	return instances, nil
}

func (r InstancesRepo) Count(ctx context.Context, function core.FunctionDefinition) (int, error) {
	count, err := r.bucket.Count(ctx, presenceKey(function.Name(), "*"))
	if err != nil {
		return 0, fmt.Errorf("failed to count functions instances: %w", err)
	}

	return count, nil
}

func (r InstancesRepo) HealthCheck() error {
	return nil
}

func (r InstancesRepo) Shutdown() error {
	return nil
}

func (r InstancesRepo) Get(
	ctx context.Context,
	function core.FunctionDefinition,
	id string,
) (core.FunctionInstance, error) {
	list, err := r.bucket.All(ctx, allKeys(function.Name(), id))
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return nil, core.ErrInstanceNotFound
		}

		return nil, fmt.Errorf("failed to get instance info: %w", err)
	}

	if len(list) == 0 {
		return nil, core.ErrInstanceNotFound
	}

	indexedRecords := groupByKey(list)

	return NewFunctionInstance(id, function, indexedRecords), nil
}

func (r InstancesRepo) Delete(ctx context.Context, function core.FunctionDefinition, id string) error {
	list, err := r.bucket.All(ctx, allKeys(function.Name(), id))
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	if len(list) == 0 {
		return core.ErrInstanceNotFound
	}

	// TODO: parallel?
	for _, item := range list {
		err = r.bucket.Delete(ctx, item.Key)
		if err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}
	}

	return nil
}

func (r InstancesRepo) Add(ctx context.Context, function core.FunctionDefinition, id string) error {
	_, err := r.bucket.Create(ctx, presenceKey(function.Name(), id), []byte{}) // TODO: put something in?
	if err != nil {
		if errors.Is(err, core.ErrKeyExists) {
			return fmt.Errorf("%w: %s", core.ErrInstanceAlreadyExists, id)
		}

		return fmt.Errorf("failed to create instance: %w", err)
	}

	return r.SetBusy(ctx, function, id, false)
}

func (r InstancesRepo) SetLastExecuted(
	ctx context.Context,
	function core.FunctionDefinition,
	id string,
	timestamp time.Time,
) error {
	err := r.bucket.Put(ctx, lastExecutedKey(function.Name(), id), serializeTime(timestamp))
	if err != nil {
		return fmt.Errorf("failed to update last executed time: %w", err)
	}

	return nil
}

func (r InstancesRepo) SetBusy(ctx context.Context, function core.FunctionDefinition, id string, busy bool) error {
	var err error
	if busy {
		err = r.bucket.Delete(ctx, idleKey(function.Name(), id))
	} else {
		err = r.bucket.Put(ctx, idleKey(function.Name(), id), []byte{}) // TODO: put something in?
	}

	if err != nil {
		return fmt.Errorf("failed to update busy status: %w", err)
	}

	return nil
}

func (r InstancesRepo) CountIdle(ctx context.Context, function core.FunctionDefinition) (int, error) {
	count, err := r.bucket.Count(ctx, idleKey(function.Name(), "*"))
	if err != nil {
		return 0, fmt.Errorf("failed to count idle instances: %w", err)
	}

	return count, nil
}

func lastExecutedKey(functionName, instanceID string) string {
	return fmt.Sprintf("%s.%s.lastExecuted", functionName, instanceID)
}

func idleKey(functionName, instanceID string) string {
	return fmt.Sprintf("%s.%s.idle", functionName, instanceID)
}

func presenceKey(functionName, instanceID string) string {
	return fmt.Sprintf("%s.%s.presence", functionName, instanceID)
}

func allKeys(functionName, instanceID string) string {
	return fmt.Sprintf("%s.%s.*", functionName, instanceID)
}

// parseKey parses a lastExecutedKey of format <functionName>.<uuid> into function and instance UUID
// Panics if format is incorrect!
func parseKey(key string) (string, string) {
	parts := strings.Split(key, ".")

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

func groupByKey(list []core.KVEntry) map[string]core.KVEntry {
	result := make(map[string]core.KVEntry)
	for _, item := range list {
		result[item.Key] = item
	}

	return result
}
