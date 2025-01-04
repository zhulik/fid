package main

import (
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

	log.Info("Running...")
	wait(func() {
		log.Info("Exit.")
	})
}
