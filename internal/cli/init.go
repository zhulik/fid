package cli

import (
	"context"
	"fmt"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
)

var initCMD = &cli.Command{
	Name:     "init",
	Aliases:  []string{"s"},
	Usage:    "Init FID. Does not start any services.",
	Category: "User",
	Flags: []cli.Flag{
		flags.NatsURL,
		flags.LogLevel,
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		injector := initDI(cmd)

		err := createBuckets(ctx, injector)
		if err != nil {
			return fmt.Errorf("failed to create buckets: %w", err)
		}

		return nil
	},
}

func createBuckets(ctx context.Context, injector *do.Injector) error {
	logger := di.Logger(injector)
	kv := do.MustInvoke[core.KV](injector)

	_, err := kv.CreateBucket(ctx, core.BucketNameInstances, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameElections, di.Config(injector).ElectionsBucketTTL())
	if err != nil {
		return fmt.Errorf("failed to create or update elections bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update functions bucket: %w", err)
	}

	logger.Info("Buckets created or updated")

	return nil
}
