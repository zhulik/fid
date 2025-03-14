package cli

import (
	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
)

func initDI(cmd *cli.Command) *do.Injector {
	injector := di.Init()

	do.Provide(injector, func(_ *do.Injector) (core.Config, error) {
		return config.Config{
			HTTPPort_:           int(cmd.Int(flagNameServerPort)),
			FunctionName_:       cmd.String(flagNameFunctionName),
			FunctionInstanceID_: cmd.String(flagNameFunctionInstanceID),
			NATSURL_:            cmd.String(flagNameNATSURL),
			LogLevel_:           cmd.String(flagNameLogLevel),
		}, nil
	})

	return injector
}
