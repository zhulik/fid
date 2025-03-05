package cli

import (
	"context"
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/fid/internal/scaler"
)

var scalerCMD = &cli.Command{
	Name:     "scaler",
	Aliases:  []string{"sc"},
	Usage:    "Run scaler.",
	Category: "Function",
	Action: func(ctx context.Context, command *cli.Command) error {
		server := do.MustInvoke[*scaler.Server](nil)

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
