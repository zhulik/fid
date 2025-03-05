package cli

import (
	"github.com/urfave/cli/v3"
)

var runtimeapiCMD = &cli.Command{
	Name:     "runtimeapi",
	Aliases:  []string{"ra"},
	Usage:    "Run runtime api server.",
	Category: "Function",
}
