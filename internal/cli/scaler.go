package cli

import (
	"context"
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do/v2"
	"github.com/sirupsen/logrus"
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
		injector := initDI(cmd)

		server := do.MustInvoke[*scaler.Server](injector)
		logger := do.MustInvoke[logrus.FieldLogger](injector)

		logger.Info("Starting...")

		go func() {
			err := server.Run()

			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			logger.WithError(err).Fatal("Failed to run server")
		}()

		logger.Info("Running...")

		_, err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)

		return err
	},
}
