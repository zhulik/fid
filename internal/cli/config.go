package cli

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/pal"
)

func runApp(ctx context.Context, cmd *cli.Command, services ...pal.ServiceDef) error {
	var level slog.Level

	lo.Must0(level.UnmarshalText([]byte(cmd.String(flags.FlagNameLogLevel))))

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	slog.SetDefault(logger)

	cfg := &config.Config{
		HTTPPort:           int(cmd.Int(flags.FlagNameServerPort)),
		FunctionName:       cmd.String(flags.FlagNameFunctionName),
		FunctionInstanceID: cmd.String(flags.FlagNameFunctionInstanceID),
		NATSURL:            cmd.String(flags.FlagNameNATSURL),
		LogLevel:           level,
		ElectionsBucketTTL: lo.Must(time.ParseDuration("2s")), // TODO: use const everywhere
		FidfilePath:        cmd.String(cmd.String("fidfile")),
	}

	p, err := di.InitPal(ctx, cfg, services...)
	if err != nil {
		return err //nolint:wrapcheck
	}

	return p.Run(ctx) //nolint:wrapcheck
}
