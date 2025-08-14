package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/fidfile"
	"github.com/zhulik/pal"
)

var startCMD = &cli.Command{
	Name:     "start",
	Aliases:  []string{"s"},
	Usage:    "Start FID.",
	Category: "User",
	Flags: []cli.Flag{
		flags.NatsURL,
		flags.LogLevel,
		&cli.StringFlag{
			Name:    "fidfile",
			Aliases: []string{"f"},
			Value:   core.FilenameFidfile,
			Usage:   "Load Fidfile.yaml from `FILE`",
			Sources: cli.EnvVars("FIDFILE"),
		},
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		p, err := initDI(ctx, cmd)
		if err != nil {
			return err
		}

		logger := lo.Must(pal.Invoke[*slog.Logger](ctx, p))

		fidFilePath := cmd.String("fidfile")
		logger.Info("Starting...")
		logger.Info("Loading", "fidfile", fidFilePath)

		fidFile, err := fidfile.ParseFile(fidFilePath)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", fidFilePath, err)
		}

		backend := lo.Must(pal.Invoke[core.ContainerBackend](ctx, p))

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

		return registerFunctions(ctx, p, backend, fidFile.Functions)
	},
}

func registerFunctions(
	ctx context.Context,
	p *pal.Pal,
	backend core.ContainerBackend,
	functions map[string]*fidfile.Function,
) error {
	pubSuber := lo.Must(pal.Invoke[core.PubSuber](ctx, p))
	functionsRepo := lo.Must(pal.Invoke[core.FunctionsRepo](ctx, p))
	logger := lo.Must(pal.Invoke[*slog.Logger](ctx, p))

	logger.Info("Registering functions", "count", len(functions))

	for _, function := range functions {
		logger := logger.With("function", function.Name())

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
