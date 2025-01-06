package main

import (
	"errors"
	"net/http"
	"syscall"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/backends"
	"github.com/zhulik/fid/pkg/core"
	"github.com/zhulik/fid/pkg/di"
	"github.com/zhulik/fid/pkg/log"
	"github.com/zhulik/fid/pkg/proxyserver"
)

var logger = log.Logger.WithField("component", "main")

func main() {
	logger.Info("Starting...")

	injector := di.New()

	proxyserver.Register(injector)
	backends.Register(injector)

	do.MustInvoke[core.ContainerBackend](injector)

	server := do.MustInvoke[*proxyserver.Server](injector)

	for service, err := range injector.HealthCheck() {
		if err != nil {
			logger.WithFields(logrus.Fields{
				"component": service,
			}).WithError(err).Fatal("Health check failed")
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
