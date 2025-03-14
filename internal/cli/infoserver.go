package cli

import (
	"context"
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/fid/internal/infoserver"
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
		injector := initDI(cmd)

		logger := di.Logger(injector)

		logger.Info("Starting...")
		server := do.MustInvoke[*infoserver.Server](injector)

		logger.Info("Starting...")

		go func() {
			err := server.Run()

			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			logger.WithError(err).Fatal("Failed to run server")
		}()

		logger.Info("Running...")

		return do.DefaultInjector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	},
}
