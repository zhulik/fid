package cli

import (
	"github.com/urfave/cli/v3"
)

var infoserverCMD = &cli.Command{
	Name:     "infoserver",
	Aliases:  []string{"is"},
	Usage:    "Run info server.",
	Category: "Service",
}
