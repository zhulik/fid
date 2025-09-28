package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/infoserver"
)

var infoServerCMD = &cli.Command{
	Name:     core.ComponentNameInfoServer,
	Aliases:  []string{"is"},
	Usage:    "Info server is a component that provides information about functions, instances, execution and various metrics.", //nolint:lll
	Category: "Service",
	Flags: append(
		flags.ForServer,
		flags.ForBackend...,
	),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runApp(ctx, cmd, infoserver.Provide())
	},
}
