package main

import (
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

	logger.Info("Running...")
	wait(func() {
		logger.Info("Exit.")
	})
}
