package cli

import (
	"github.com/urfave/cli/v3"
)

var scalerCMD = &cli.Command{
	Name:     "scaler",
	Aliases:  []string{"sc"},
	Usage:    "Run scaler.",
	Category: "Function",
}
