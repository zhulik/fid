package cli

import (
	"context"
	"syscall"

	"github.com/samber/do/v2"
	"github.com/sirupsen/logrus"
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
		injector := initDI(cmd)

		logger := do.MustInvoke[logrus.FieldLogger](injector)

		logger.Info("Starting...")

		logger.Info("Running...")

		_, err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)

		return err
	},
}
