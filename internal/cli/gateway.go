package cli

import (
	"github.com/urfave/cli/v3"
)

var gatewayCMD = &cli.Command{
	Name:     "gateway",
	Aliases:  []string{"gw"},
	Usage:    "Run gateway server.",
	Category: "Service",
}
