package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/scaler"
)

var scalerCMD = &cli.Command{
	Name:     core.ComponentNameScaler,
	Aliases:  []string{"sc"},
	Usage:    "Run scaler.",
	Category: "Function",
	Flags: append(append(
		flags.ForServer,
		flags.FunctionName),
		flags.ForBackend...,
	),

	Action: func(ctx context.Context, cmd *cli.Command) error {
		p, err := initDI(ctx, cmd, scaler.Provide())
		if err != nil {
			return err
		}

		return p.Run(ctx)
	},
}
