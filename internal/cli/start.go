package cli

import (
	"github.com/urfave/cli/v3"
)

var startCMD = &cli.Command{
	Name:     "start",
	Aliases:  []string{"s"},
	Usage:    "Start FID.",
	Category: "User",
}
