package cli

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
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
		initCMD,
		garbageCollectorCMD,
		functionGarbageCollectorCMD,
	},
}

func Run() {
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
