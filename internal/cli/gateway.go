package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
)

var gatewayCMD = &cli.Command{
	Name:     core.ComponentNameGateway,
	Aliases:  []string{"gw"},
	Usage:    "Run gateway server.",
	Category: "Service",
	Flags:    flags.ForServer,
	Action: func(ctx context.Context, cmd *cli.Command) error {
		p, err := initDI(ctx, cmd)
		if err != nil {
			return err
		}

		return p.Run(ctx)
	},
}
