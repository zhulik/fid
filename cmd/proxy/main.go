package main

import (
	"github.com/samber/do"
	"github.com/zhulik/fid/pkg/di"
	"github.com/zhulik/fid/pkg/httpserver"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhulik/fid/pkg/log"
)

var logger = log.Logger.WithField("component", "main")

func wait(shutdown func()) {
	defer shutdown()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs

	logger.Info("Received signal: ", sig)
	logger.Info("Shutting down...")
}

func main() {
	logger.Info("Starting...")

	injector := di.New()

	httpserver.Register(injector)

	do.MustInvoke[*httpserver.Server](injector) // Start the service

	logger.Info("Running...")
	wait(func() {
		logger.Info("Cleaning up...")
		injector.Shutdown()
		logger.Info("Exit.")
	})
}
