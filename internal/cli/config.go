package cli

import (
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/di"
)

func initDI(cmd *cli.Command) *do.RootScope {
	injector := di.Init()

	do.Provide(injector, func(_ do.Injector) (config.Config, error) {
		var level slog.Level

		lo.Must0(level.UnmarshalText([]byte(cmd.String(flags.FlagNameLogLevel))))

		return config.Config{
			HTTPPort_:           int(cmd.Int(flags.FlagNameServerPort)),
			FunctionName_:       cmd.String(flags.FlagNameFunctionName),
			FunctionInstanceID_: cmd.String(flags.FlagNameFunctionInstanceID),
			NATSURL_:            cmd.String(flags.FlagNameNATSURL),
			LogLevel_:           level,
		}, nil
	})

	return injector
}
