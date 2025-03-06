package cli

import (
	"log"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
)

func registerConfig(cmd *cli.Command) {
	do.Provide[core.Config](nil, func(injector *do.Injector) (core.Config, error) {
		log.Printf("%+v", cmd.String(flagNameLogLevel))

		return config.Config{
			HTTPPort_:           int(cmd.Int(flagNameServerPort)),
			FunctionName_:       cmd.String(flagNameFunctionName),
			FunctionInstanceID_: cmd.String(flagNameFunctionInstanceID),
			NATSURL_:            cmd.String(flagNameNATSURL),
			LogLevel_:           cmd.String(flagNameLogLevel),
		}, nil
	})
}
