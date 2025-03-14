package cli

import (
	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
)

func initDI(cmd *cli.Command) *do.Injector {
	injector := di.Init()

	do.Provide(injector, func(_ *do.Injector) (core.Config, error) {
		return config.Config{
			HTTPPort_:           int(cmd.Int(flags.FlagNameServerPort)),
			FunctionName_:       cmd.String(flags.FlagNameFunctionName),
			FunctionInstanceID_: cmd.String(flags.FlagNameFunctionInstanceID),
			NATSURL_:            cmd.String(flags.FlagNameNATSURL),
			LogLevel_:           cmd.String(flags.FlagNameLogLevel),
		}, nil
	})

	return injector
}
