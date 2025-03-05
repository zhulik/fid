package cli

import (
	"github.com/urfave/cli/v3"
)

var healthcheckCMD = &cli.Command{
	Name:     "healthcheck",
	Aliases:  []string{"hc"},
	Usage:    "Run healthcheck.",
	Category: "Utility",
}
