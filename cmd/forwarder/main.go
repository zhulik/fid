package main

import (
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/di"
	"github.com/zhulik/fid/pkg/forwarder"
)

func main() {
	injector := di.New()
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "main")

	logger.Info("Starting...")

	server := do.MustInvoke[*forwarder.Server](injector)

	go func() {
		err := server.Run()

		if errors.Is(err, http.ErrServerClosed) {
			return
		}

		logger.WithError(err).Fatal("Failed to run server")
	}()

	logger.Info("Running...")

	err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	if err != nil {
		logger.WithError(err).Fatal("Failed to shutdown")
	}
}
