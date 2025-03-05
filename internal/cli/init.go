package cli

import (
	"github.com/urfave/cli/v3"
)

var initCMD = &cli.Command{
	Name:     "init",
	Aliases:  []string{"s"},
	Usage:    "Start FID.",
	Category: "User",
}
