package docker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/json"
)

type FunctionsRepo struct { //nolint:recvcheck
	Logger *slog.Logger
	KV     core.KV
	bucket core.KVBucket
}

func (r *FunctionsRepo) Init(ctx context.Context) error {
	bucket, err := r.KV.CreateBucket(ctx, core.BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create functions bucket: %w", err)
	}

	r.bucket = bucket

	return nil
}

func (r FunctionsRepo) Upsert(ctx context.Context, function core.FunctionDefinition) error {
	backendFunction := Function{
		Name_:    function.Name(),
		Image_:   function.Image(),
		Timeout_: function.Timeout(),
		MinScale: function.ScalingConfig().Min,
		MaxScale: function.ScalingConfig().Max,
		Env_:     function.Env(),
	}

	bytes, err := json.Marshal(backendFunction)
	if err != nil {
		return fmt.Errorf("failed to marshal function: %w", err)
	}

	err = r.bucket.Put(ctx, function.Name(), bytes)
	if err != nil {
		return fmt.Errorf("failed to store function template: %w", err)
	}

	r.Logger.With("function", function).Info("Function template stored")

	return nil
}

func (r FunctionsRepo) Get(ctx context.Context, name string) (core.FunctionDefinition, error) {
	r.Logger.With("function", name).Debug("Fetching function info")

	bytes, err := r.bucket.Get(ctx, name)
	if err != nil {
		if errors.Is(err, core.ErrKeyNotFound) {
			return nil, core.ErrFunctionNotFound
		}

		return nil, fmt.Errorf("failed to get function template: %w", err)
	}

	function, err := json.Unmarshal[Function](bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal function template: %w", err)
	}

	return function, nil
}

func (r FunctionsRepo) List(ctx context.Context) ([]core.FunctionDefinition, error) {
	r.Logger.Debug("Fetching function list")

	fns, err := r.bucket.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get function list: %w", err)
	}

	functions := make([]core.FunctionDefinition, len(fns))

	for i, fn := range fns {
		function, err := json.Unmarshal[Function](fn.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal function: %w", err)
		}

		functions[i] = function
	}

	return functions, nil
}

func (r FunctionsRepo) Delete(ctx context.Context, name string) error {
	err := r.bucket.Delete(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete function template: %w", err)
	}

	return nil
}
