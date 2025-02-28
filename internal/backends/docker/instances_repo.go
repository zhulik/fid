package docker

import (
	"context"
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

func (r InstancesRepo) List(ctx context.Context, functionName string) ([]core.FunctionsInstance, error) {
	return nil, nil
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

func (r InstancesRepo) HealthCheck() error {
	return nil
}

func (r InstancesRepo) Shutdown() error {
	return nil
}

func (r InstancesRepo) Upsert(ctx context.Context, id core.FunctionsInstance) error {
	return nil
}

func (r InstancesRepo) Get(ctx context.Context, id string) (core.FunctionsInstance, error) {
	return nil, nil //nolint:nilnil
}

func (r InstancesRepo) Delete(ctx context.Context, id string) error {
	return nil
}
