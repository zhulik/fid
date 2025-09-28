package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
)

// Responsibilities:
// Monitor pods: if function or runtime api crash - delete the rest and the network.
// Call the scaler or do it here?
// What if scaler crashes?
var functionGarbageCollectorCMD = &cli.Command{
	Name:     core.ComponentNameFunctionGarbageCollector,
	Aliases:  []string{"fngc"},
	Usage:    "Run function garbage collector.",
	Category: "Service",
	Flags: append(append(
		flags.ForServer,
		flags.FunctionName),
		flags.ForBackend...,
	),

	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runApp(ctx, cmd)
	},
}
