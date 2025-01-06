package main

import (
	"errors"
	"net/http"
	"runtime/debug"
	"syscall"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/backends"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/di"
	"github.com/zhulik/fid/pkg/infoserver"
)

func main() {
	injector := di.New()

	logger := do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "main")

	logger.Info("Starting...")

	infoserver.Register(injector)
	backends.Register(injector)

	do.MustInvoke[core.ContainerBackend](injector)

	server := do.MustInvoke[*infoserver.Server](injector)

	for service, err := range injector.HealthCheck() {
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":     err,
				"component": service,
			}).Fatal("Application panicked:", string(debug.Stack()))
		}
	}

	go func() {
		// Start the service
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
