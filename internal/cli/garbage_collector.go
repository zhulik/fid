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

var garbageCollectorCMD = &cli.Command{
	Name:     core.ComponentNameGarbageCollector,
	Aliases:  []string{"gc"},
	Usage:    "Run global garbage collector.",
	Category: "Service",
	Flags:    append(flags.ForServer, flags.ForBackend...),

	Action: func(ctx context.Context, cmd *cli.Command) error {
		injector := initDI(cmd)

		logger := do.MustInvoke[logrus.FieldLogger](injector)

		logger.Info("Starting...")

		logger.Info("Running...")

		_, err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)

		return err
	},
}
