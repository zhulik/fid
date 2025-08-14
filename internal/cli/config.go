package cli

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/pal"
)

func initDI(ctx context.Context, cmd *cli.Command) (*pal.Pal, error) {
	var level slog.Level

	lo.Must0(level.UnmarshalText([]byte(cmd.String(flags.FlagNameLogLevel))))

	cfg := &config.Config{
		HTTPPort:           int(cmd.Int(flags.FlagNameServerPort)),
		FunctionName:       cmd.String(flags.FlagNameFunctionName),
		FunctionInstanceID: cmd.String(flags.FlagNameFunctionInstanceID),
		NATSURL:            cmd.String(flags.FlagNameNATSURL),
		LogLevel:           level,
		ElectionsBucketTTL: lo.Must(time.ParseDuration("2s")), // TODO: use const everywhere
	}

	p, err := di.InitPal(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pal: %w", err)
	}

	return p, nil
}
