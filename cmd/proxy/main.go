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

func wait(shutdown func()) {
	defer shutdown()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs

	log.Info("Received signal: ", sig)
	log.Info("Shutting down...")
}

func main() {
	log.Info("Starting...")

	injector := di.New()

	httpserver.Register(injector)

	do.MustInvoke[*httpserver.Server](injector) // Start the service

	log.Info("Running...")
	wait(func() {
		log.Info("Cleaning up...")
		injector.Shutdown()
		log.Info("Exit.")
	})
}
