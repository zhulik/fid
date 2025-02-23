package main

import (
	"context"
	"os"
	"time"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	_ "github.com/zhulik/fid/internal/di"
)

const (
	DefaultFileName     = "functions.yaml"
	RegistrationTimeout = 10 * time.Second
)

var (
	logger   logrus.FieldLogger    //nolint:gochecknoglobals
	config   core.Config           //nolint:gochecknoglobals
	backend  core.ContainerBackend //nolint:gochecknoglobals
	pubSuber core.PubSuber         //nolint:gochecknoglobals
	kv       core.KV               //nolint:gochecknoglobals
)

func init() { //nolint:gochecknoinits
	config = do.MustInvoke[core.Config](nil)
	logger = do.MustInvoke[logrus.FieldLogger](nil).WithField("component", "main")
	backend = do.MustInvoke[core.ContainerBackend](nil)
	pubSuber = do.MustInvoke[core.PubSuber](nil)
	kv = do.MustInvoke[core.KV](nil)
}

func main() {
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

	ctx, cancel := context.WithTimeout(context.Background(), RegistrationTimeout)
	defer cancel()

	for _, function := range functions {
		logger := logger.WithField("function", function.Name())

		err := pubSuber.CreateOrUpdateFunctionStream(ctx, function)
		if err != nil {
			logger.Fatalf("error creating or updating function stream %s: %v", function.Name(), err)
		}

		// TODO: better place for this and for bucket naming?
		err = kv.CreateBucket(ctx, function.Name()+"-elections", config.ElectionsBucketTTL())
		if err != nil {
			logger.Fatalf("failed to create or update function elections bucket: %v", err)
		}

		logger.Info("Elections bucket created")

		err = backend.Register(ctx, function)
		if err != nil {
			logger.Fatalf("error registering function %s: %v", function.Name(), err)
		}
	}

	// TODO: delete functions that are not in the list

	if len(functions) == 0 {
		// TODO: stop gateway
		return
	}
	// Start gateway
	// Start scaler per function
	// Wait until they are healhy
	// Exit
	return //nolint:gosimple
}
