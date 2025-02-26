package main

import (
	"context"
	"errors"
	"fmt"
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

	err = createBuckets(ctx)
	if err != nil {
		logger.Fatalf("failed to create buckets: %v", err)
	}

	_, err = startGateway(ctx)
	if err != nil {
		if !errors.Is(err, core.ErrContainerAlreadyExists) {
			logger.Fatalf("failed to start info server: %v", err)
		}
	}

	_, err = startInfoServer(ctx)
	if err != nil {
		if !errors.Is(err, core.ErrContainerAlreadyExists) {
			logger.Fatalf("failed to start info server: %v", err)
		}
	}

	err = registerFunctions(ctx, functions)
	if err != nil {
		logger.Fatalf("failed to register function: %v", err)
	}
}

func createBuckets(ctx context.Context) error {
	_, err := kv.CreateBucket(ctx, core.BucketNameInstances, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameElections, config.ElectionsBucketTTL())
	if err != nil {
		return fmt.Errorf("failed to create or update elections bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update functions bucket: %w", err)
	}

	logger.Info("Buckets created or updated")

	return nil
}

func registerFunctions(ctx context.Context, functions map[string]*Function) error {
	logger.Infof("Registering %d functions...", len(functions))

	for _, function := range functions {
		logger := logger.WithField("function", function.Name())

		err := pubSuber.CreateOrUpdateFunctionStream(ctx, function)
		if err != nil {
			return fmt.Errorf("error creating or updating function stream %s: %w", function.Name(), err)
		}

		logger.Info("Elections bucket created")

		err = backend.Register(ctx, function)
		if err != nil {
			return fmt.Errorf("error registering function %s: %w", function.Name(), err)
		}
	}

	templates, err := backend.Functions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list functions: %w", err)
	}

	for _, template := range templates {
		_, exists := functions[template.Name()]
		if exists {
			continue
		}

		err := backend.Deregister(ctx, template.Name())
		if err != nil {
			return fmt.Errorf("failed to deregister function %s: %w", template.Name(), err)
		}
	}

	return nil
}

func startGateway(ctx context.Context) (string, error) {
	id, err := backend.StartGateway(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start gateway: %w", err)
	}

	return id, nil
}

func startInfoServer(ctx context.Context) (string, error) {
	id, err := backend.StartInfoServer(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start info server: %w", err)
	}

	return id, nil
}
