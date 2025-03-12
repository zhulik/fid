package cli

import (
	"context"
	"fmt"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
)

var initCMD = &cli.Command{
	Name:     "init",
	Aliases:  []string{"s"},
	Usage:    "Init FID. Does not start any services.",
	Category: "User",
	Flags: []cli.Flag{
		flagNatsURL,
		flagLogLevel,
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		registerConfig(cmd)

		err := createBuckets(ctx)
		if err != nil {
			return fmt.Errorf("failed to create buckets: %w", err)
		}

		return nil
	},
}

func createBuckets(ctx context.Context) error {
	kv := do.MustInvoke[core.KV](nil)

	_, err := kv.CreateBucket(ctx, core.BucketNameInstances, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameElections, di.Config().ElectionsBucketTTL())
	if err != nil {
		return fmt.Errorf("failed to create or update elections bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update functions bucket: %w", err)
	}

	di.Logger().Info("Buckets created or updated")

	return nil
}
