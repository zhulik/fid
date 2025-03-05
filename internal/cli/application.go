package cli

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/di"
)

const VERSION = "0.1.0"

var cmd = &cli.Command{
	Name:    "fid",
	Usage:   "Function in docker cli.",
	Version: VERSION,
}

func Run() {
	cmd.Commands = []*cli.Command{
		gatewayCMD,
		infoserverCMD,
		runtimeapiCMD,
		scalerCMD,
		healthcheckCMD,
		startCMD,
		initCMD,
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		di.Logger().WithError(err).Fatal("failed to run command")
	}
}
