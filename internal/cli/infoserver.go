package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
)

var infoServerCMD = &cli.Command{
	Name:     core.ComponentNameInfoServer,
	Aliases:  []string{"is"},
	Usage:    "Run info server.",
	Category: "Service",
	Flags: append(
		flags.ForServer,
		flags.ForBackend...,
	),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		p, err := initDI(ctx, cmd)
		if err != nil {
			return err
		}

		return p.Run(ctx)
	},
}
