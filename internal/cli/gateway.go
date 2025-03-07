package cli

import (
	"context"
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/fid/internal/gateway"
)

var gatewayCMD = &cli.Command{
	Name:     core.ComponentNameGateway,
	Aliases:  []string{"gw"},
	Usage:    "Run gateway server.",
	Category: "Service",
	Flags:    flagsServer,
	Action: func(ctx context.Context, cmd *cli.Command) error {
		registerConfig(cmd)
		server := do.MustInvoke[*gateway.Server](nil)

		di.Logger().Info("Starting...")

		go func() {
			err := server.Run()

			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			di.Logger().WithError(err).Fatal("Failed to run server")
		}()

		di.Logger().Info("Running...")

		return do.DefaultInjector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	},
}
