package main

import (
	"github.com/zhulik/fid/pkg/di"
	"syscall"

	"github.com/zhulik/fid/pkg/log"
)

var logger = log.Logger.WithField("component", "main")

func main() {
	logger.Info("Starting...")

	injector := di.New()

	logger.Info("Running...")
	err := injector.ShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)
	if err != nil {
		panic(err)
	}
}
