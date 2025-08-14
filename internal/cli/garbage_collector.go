package cli

import (
	"context"
	"log/slog"
	"syscall"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
)

// Responsibilities:
// Delete all resources and containers if a function is deleted.
// ???
var garbageCollectorCMD = &cli.Command{
	Name:     core.ComponentNameGarbageCollector,
	Aliases:  []string{"gc"},
	Usage:    "Run global garbage collector.",
	Category: "Service",
	Flags:    append(flags.ForServer, flags.ForBackend...),

	Action: func(ctx context.Context, cmd *cli.Command) error {
		injector := initDI(cmd)

		logger := do.MustInvoke[*slog.Logger](injector)

		logger.Info("Starting...")

		logger.Info("Running...")

		_, err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)

		return err
	},
}
