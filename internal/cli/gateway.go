package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/gateway"
)

var gatewayCMD = &cli.Command{
	Name:     core.ComponentNameGateway,
	Aliases:  []string{"gw"},
	Usage:    "Run gateway server.",
	Category: "Service",
	Flags:    flags.ForServer,
	Action: func(ctx context.Context, cmd *cli.Command) error {
		p, err := initDI(ctx, cmd, gateway.Provide())
		if err != nil {
			return err
		}

		return p.Run(ctx)
	},
}
