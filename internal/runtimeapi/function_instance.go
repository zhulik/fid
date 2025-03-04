package runtimeapi

import (
	"context"
	"time"

	"github.com/zhulik/fid/internal/core"
)

type functionInstance struct {
	core.FunctionDefinition
	id            string
	instancesRepo core.InstancesRepo
}

func (fi functionInstance) add(ctx context.Context) error {
	return fi.instancesRepo.Add(ctx, fi, fi.id) //nolint:wrapcheck
}

func (fi functionInstance) delete(ctx context.Context) error {
	return fi.instancesRepo.Delete(ctx, fi, fi.id) //nolint:wrapcheck
}

func (fi functionInstance) busy(ctx context.Context, busy bool) error {
	return fi.instancesRepo.SetBusy(ctx, fi, fi.id, busy) //nolint:wrapcheck
}

func (fi functionInstance) executed(ctx context.Context) error {
	return fi.instancesRepo.SetLastExecuted(ctx, fi, fi.id, time.Now()) //nolint:wrapcheck
}
