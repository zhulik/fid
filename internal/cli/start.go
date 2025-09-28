package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/fidfile"
	"github.com/zhulik/pal"
)

type Starter struct {
	Logger        *slog.Logger
	Backend       core.ContainerBackend
	PubSuber      core.PubSuber
	FunctionsRepo core.FunctionsRepo
	KV            core.KV
	Config        *config.Config

	CMD *cli.Command `pal:"name=command"`
}

func (s *Starter) Run(ctx context.Context) error {
	initOnly := s.CMD.Bool(flags.FlagNameInitOnly)

	fidFilePath := s.Config.FidfilePath

	s.Logger.Info("Starting...", "init-only", initOnly)

	err := s.createKVBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to create KV buckets %w", err)
	}

	s.Logger.Info("Loading", "fidfile", fidFilePath)

	fidFile, err := fidfile.ParseFile(fidFilePath)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", fidFilePath, err)
	}

	err = s.createOrUpdateFunctionStreams(ctx, fidFile.Functions)
	if err != nil {
		return fmt.Errorf("failed to create or update function streams %s: %w", fidFilePath, err)
	}

	if initOnly {
		return nil
	}

	_, err = s.startGateway(ctx)
	if err != nil {
		if !errors.Is(err, core.ErrContainerAlreadyExists) {
			return fmt.Errorf("failed to start gateway: %w", err)
		}
	}

	if fidFile.InfoServer != nil {
		_, err = s.startInfoServer(ctx)
		if err != nil {
			if !errors.Is(err, core.ErrContainerAlreadyExists) {
				return fmt.Errorf("failed to start info server: %w", err)
			}
		}
	}

	return s.registerFunctions(ctx, fidFile.Functions)
}

func (s *Starter) createKVBuckets(ctx context.Context) error {
	_, err := s.KV.CreateBucket(ctx, core.BucketNameInstances)
	if err != nil {
		return fmt.Errorf("failed to create or update instances bucket: %w", err)
	}

	_, err = s.KV.CreateBucket(ctx, core.BucketNameFunctions)
	if err != nil {
		return fmt.Errorf("failed to create or update functions bucket: %w", err)
	}

	s.Logger.Info("KV buckets created or updated")

	return nil
}

func (s *Starter) createOrUpdateFunctionStreams(ctx context.Context, functions map[string]*fidfile.Function) error {
	s.Logger.Info("Creating function streams", "count", len(functions))

	for _, function := range functions {
		err := s.PubSuber.CreateOrUpdateFunctionStream(ctx, function)
		if err != nil {
			return fmt.Errorf("error creating or updating function stream %s: %w", function.Name(), err)
		}
	}

	return nil
}

func (s *Starter) registerFunctions(ctx context.Context, functions map[string]*fidfile.Function) error {
	s.Logger.Info("Registering functions", "count", len(functions))

	for _, function := range functions {
		err := s.Backend.Register(ctx, function)
		if err != nil {
			return fmt.Errorf("error registering function %s: %w", function.Name(), err)
		}
	}

	templates, err := s.FunctionsRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list functions: %w", err)
	}

	for _, template := range templates {
		_, exists := functions[template.Name()]
		if exists {
			continue
		}

		err := s.Backend.Deregister(ctx, template)
		if err != nil {
			return fmt.Errorf("failed to deregister function %s: %w", template, err)
		}
	}

	return nil
}

func (s *Starter) startGateway(ctx context.Context) (string, error) {
	id, err := s.Backend.StartGateway(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start gateway: %w", err)
	}

	return id, nil
}

func (s *Starter) startInfoServer(ctx context.Context) (string, error) {
	id, err := s.Backend.StartInfoServer(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start info server: %w", err)
	}

	return id, nil
}

var startCMD = &cli.Command{
	Name:     "start",
	Aliases:  []string{"s"},
	Usage:    "Start FID. Reads Fidfile.yaml and starts the required services.",
	Category: "User",
	Flags: []cli.Flag{
		flags.NatsURL,
		flags.LogLevel,
		&cli.BoolFlag{
			Name:    flags.FlagNameInitOnly,
			Aliases: []string{"i"},
			Usage:   "If specified, only streams and buckets will be created, no services will start",
		},
		&cli.StringFlag{
			Name:    flags.FlagNameFIDFile,
			Aliases: []string{"f"},
			Value:   core.FilenameFidfile,
			Usage:   "Load Fidfile.yaml from `FILE`",
			Sources: cli.EnvVars("FIDFILE"),
		},
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runApp(ctx, cmd,
			pal.Provide(&Starter{}),
		)
	},
}
