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
	"github.com/zhulik/fid/internal/runtimeapi"
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
		injector := initDI(cmd)
		server := do.MustInvoke[*runtimeapi.Server](injector)

		logger := di.Logger(injector)

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
