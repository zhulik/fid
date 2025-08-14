package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
)

var runtimeapiCMD = &cli.Command{
	Name:     core.ComponentNameRuntimeAPI,
	Aliases:  []string{"ra"},
	Usage:    "Run runtime api server.",
	Category: "Function",
	Flags: append(
		flags.ForServer,
		flags.FunctionName,
		flags.FunctionInstanceID,
	),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		p, err := initDI(ctx, cmd)
		if err != nil {
			return err
		}

		return p.Run(ctx)
	},
}
