package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/pal"
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
		p, err := initDI(ctx, cmd)
		if err != nil {
			return err
		}

		err = createBuckets(ctx, p)
		if err != nil {
			return fmt.Errorf("failed to create buckets: %w", err)
		}

		return nil
	},
}

func createBuckets(ctx context.Context, p *pal.Pal) error {
	logger := lo.Must(pal.Invoke[*slog.Logger](ctx, p))
	kv := lo.Must(pal.Invoke[core.KV](ctx, p))
	cfg := lo.Must(pal.Invoke[config.Config](ctx, p))

	_, err := kv.CreateBucket(ctx, core.BucketNameInstances, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameElections, cfg.ElectionsBucketTTL)
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
