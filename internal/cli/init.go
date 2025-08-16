package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/pal"
)

type Initer struct {
	Logger *slog.Logger
	KV     core.KV
	Config *config.Config
}

func (s *Initer) Run(ctx context.Context) error {
	_, err := s.KV.CreateBucket(ctx, core.BucketNameInstances, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = s.KV.CreateBucket(ctx, core.BucketNameElections, s.Config.ElectionsBucketTTL)
	if err != nil {
		return fmt.Errorf("failed to create or update elections bucket: %w", err)
	}

	_, err = s.KV.CreateBucket(ctx, core.BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update functions bucket: %w", err)
	}

	s.Logger.Info("Buckets created or updated")

	return nil
}

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
		return runApp(ctx, cmd, pal.Provide(&Initer{}))
	},
}
