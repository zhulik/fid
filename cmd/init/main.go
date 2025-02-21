package main

import (
	"context"
	"os"
	"time"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
)

const (
	DefaultFileName     = "functions.yaml"
	RegistrationTimeout = 10 * time.Second
)

func main() {
	injector := di.New()
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "main")

	fileName := DefaultFileName
	if len(os.Args) > 1 {
		fileName = os.Args[1]
	}

	logger.Info("Starting...")
	logger.Infof("Loading %s...", fileName)

	functions, err := ParseFile(fileName)
	if err != nil {
		logger.Fatalf("error: %v", err)
	}

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		logger.Fatalf("error instantiating backend: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), RegistrationTimeout)
	defer cancel()

	for _, function := range functions {
		err = backend.Register(ctx, function)
		if err != nil {
			logger.Fatalf("error registering function %s: %v", function.Name(), err)
		}
	}
	// Create or update all necessary JetStream resources
	// Start gateway
	// Start scaler per function
	// Wait until they are healhy
	// Exit
}
