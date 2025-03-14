package cli

import (
	"context"
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/gateway"
)

var gatewayCMD = &cli.Command{
	Name:     core.ComponentNameGateway,
	Aliases:  []string{"gw"},
	Usage:    "Run gateway server.",
	Category: "Service",
	Flags:    flags.ForServer,
	Action: func(ctx context.Context, cmd *cli.Command) error {
		injector := initDI(cmd)
		server := do.MustInvoke[*gateway.Server](injector)

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

		return do.DefaultInjector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	},
}
