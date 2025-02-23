package main

import (
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	_ "github.com/zhulik/fid/internal/di"
	"github.com/zhulik/fid/internal/runtimeapi"
)

func main() {
	logger := do.MustInvoke[logrus.FieldLogger](nil).WithField("component", "main")
	server := do.MustInvoke[*runtimeapi.Server](nil)

	logger.Info("Starting...")

	go func() {
		err := server.Run()

		if errors.Is(err, http.ErrServerClosed) {
			return
		}

		logger.WithError(err).Fatal("Failed to run server")
	}()

	logger.Info("Running...")

	err := do.DefaultInjector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	if err != nil {
		logger.WithError(err).Fatal("Failed to shutdown")
	}
}
