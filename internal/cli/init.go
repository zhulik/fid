package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/samber/do"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/di"
	"github.com/zhulik/fid/internal/fidfile"
)

var initCMD = &cli.Command{
	Name:     "init",
	Aliases:  []string{"i"},
	Usage:    "Initialize FID.",
	Category: "User",

	Action: func(ctx context.Context, command *cli.Command) error {
		fileName := core.FilenameFidfile
		if len(os.Args) > 1 {
			fileName = os.Args[1]
		}

		di.Logger().Info("Starting...")
		di.Logger().Infof("Loading %s...", fileName)

		fidFile, err := fidfile.ParseFile(fileName)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", fileName, err)
		}

		err = createBuckets(ctx)
		if err != nil {
			return fmt.Errorf("failed to create buckets: %w", err)
		}

		backend := do.MustInvoke[core.ContainerBackend](nil)

		_, err = startGateway(ctx, backend)
		if err != nil {
			if !errors.Is(err, core.ErrContainerAlreadyExists) {
				return fmt.Errorf("failed to start gateway: %w", err)
			}
		}

		if fidFile.InfoServer != nil {
			_, err = startInfoServer(ctx, backend)
			if err != nil {
				if !errors.Is(err, core.ErrContainerAlreadyExists) {
					return fmt.Errorf("failed to start info server: %w", err)
				}
			}
		}

		return registerFunctions(ctx, backend, fidFile.Functions)
	},
}

func createBuckets(ctx context.Context) error {
	kv := do.MustInvoke[core.KV](nil)

	_, err := kv.CreateBucket(ctx, core.BucketNameInstances, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameElections, di.Config().ElectionsBucketTTL())
	if err != nil {
		return fmt.Errorf("failed to create or update elections bucket: %w", err)
	}

	_, err = kv.CreateBucket(ctx, core.BucketNameFunctions, 0)
	if err != nil {
		return fmt.Errorf("failed to create or update functions bucket: %w", err)
	}

	di.Logger().Info("Buckets created or updated")

	return nil
}

func registerFunctions(
	ctx context.Context,
	backend core.ContainerBackend,
	functions map[string]*fidfile.Function,
) error {
	pubSuber := do.MustInvoke[core.PubSuber](nil)
	functionsRepo := do.MustInvoke[core.FunctionsRepo](nil)

	di.Logger().Infof("Registering %d functions...", len(functions))

	for _, function := range functions {
		logger := di.Logger().WithField("function", function.Name())

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

	templates, err := functionsRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list functions: %w", err)
	}

	for _, template := range templates {
		_, exists := functions[template.Name()]
		if exists {
			continue
		}

		err := backend.Deregister(ctx, template)
		if err != nil {
			return fmt.Errorf("failed to deregister function %s: %w", template, err)
		}
	}

	return nil
}

func startGateway(ctx context.Context, backend core.ContainerBackend) (string, error) {
	id, err := backend.StartGateway(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start gateway: %w", err)
	}

	return id, nil
}

func startInfoServer(ctx context.Context, backend core.ContainerBackend) (string, error) {
	id, err := backend.StartInfoServer(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start info server: %w", err)
	}

	return id, nil
}
