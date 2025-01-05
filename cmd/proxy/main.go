package main

import (
	"errors"
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/di"
	"github.com/zhulik/fid/pkg/httpserver"
	"net/http"
	"syscall"

	"github.com/zhulik/fid/pkg/log"
)

var logger = log.Logger.WithField("component", "main")

func main() {
	logger.Info("Starting...")

	injector := di.New()

	httpserver.Register(injector)

	go func() {
		server := do.MustInvoke[*httpserver.Server](injector) // Start the service
		err := server.Run()

		if errors.Is(err, http.ErrServerClosed) {
			return
		}

		panic(err)
	}()

	logger.Info("Running...")
	err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	if err != nil {
		panic(err)
	}
}
