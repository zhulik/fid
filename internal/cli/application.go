package cli

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/pal"
)

const VERSION = "0.1.0"

var cmd = &cli.Command{
	Name:    "fid",
	Usage:   "Function in docker cli.",
	Version: VERSION,
	Commands: []*cli.Command{
		gatewayCMD,
		infoServerCMD,
		runtimeapiCMD,
		scalerCMD,
		healthcheckCMD,
		startCMD,
	},
}

func Run() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runApp(ctx context.Context, cmd *cli.Command, services ...pal.ServiceDef) error {
	var level slog.Level

	lo.Must0(level.UnmarshalText([]byte(cmd.String(flags.FlagNameLogLevel))))

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))

	services = append(services, pal.ProvideNamed("command", cmd))

	slog.SetDefault(logger)

	cfg := &config.Config{
		HTTPPort:           int(cmd.Int(flags.FlagNameServerPort)),
		FunctionName:       cmd.String(flags.FlagNameFunctionName),
		FunctionInstanceID: cmd.String(flags.FlagNameFunctionInstanceID),
		NATSURL:            cmd.String(flags.FlagNameNATSURL),
		LogLevel:           level,
		FidfilePath:        cmd.String(flags.FlagNameFIDFile),
	}

	return di.Run(ctx, cfg, services...) //nolint:wrapcheck
}
