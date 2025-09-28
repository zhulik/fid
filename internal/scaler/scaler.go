package scaler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
)

type Scaler struct { //nolint:recvcheck
	Logger        *slog.Logger
	FunctionsRepo core.FunctionsRepo
	Config        *config.Config
	Backend       core.ContainerBackend
	InstancesRepo core.InstancesRepo

	function core.FunctionDefinition
}

func (s *Scaler) Init(ctx context.Context) error {
	function, err := s.FunctionsRepo.Get(ctx, s.Config.FunctionName)
	if err != nil {
		return fmt.Errorf("failed to get function: %w", err)
	}

	s.Logger = s.Logger.With(
		"function", function,
	)

	s.Logger.Info("Scaler created")

	s.function = function

	return nil
}

// Run the scaler. Never returns nil error. When Shutdown is called, returns Stopped.
func (s Scaler) Run(ctx context.Context) error {
	<-ctx.Done()

	return nil
}

func (s Scaler) rescaleToConfig(ctx context.Context) error { //nolint:unused
	instances, err := s.InstancesRepo.Count(ctx, s.function)
	if err != nil {
		return fmt.Errorf("failed to get instances: %w", err)
	}

	switch {
	case instances < s.function.ScalingConfig().Min:
		toCreate := s.function.ScalingConfig().Min - instances
		s.Logger.Info("No need to rescale to config",
			"instances", instances,
			"min", s.function.ScalingConfig().Min,
			"toCreate", toCreate,
		)

		for range toCreate {
			ctx, cancel := context.WithTimeout(ctx, time.Second)

			_, err = s.scaleUp(ctx)

			cancel()

			if err != nil {
				return fmt.Errorf("failed to scale up: %w", err)
			}
		}
	case instances > s.function.ScalingConfig().Max:
		// TODO: implement
		toKill := instances - s.function.ScalingConfig().Max
		s.Logger.Info("No need to rescale to config",
			"instances", instances,
			"max", s.function.ScalingConfig().Max,
			"toKill", toKill,
		)
	default:
		s.Logger.Info("No need to rescale to config",
			"instances", instances,
			"min", s.function.ScalingConfig().Min,
			"max", s.function.ScalingConfig().Max,
		)
	}

	return nil
}

func (s Scaler) scaleUp(ctx context.Context) (string, error) { //nolint:unused
	s.Logger.Info("Scaling up")

	instanceID, err := s.Backend.AddInstance(ctx, s.function)
	if err != nil {
		return "", fmt.Errorf("failed to add instance: %w", err)
	}

	s.Logger.Info("Instance added", "instanceID", instanceID)

	return instanceID, nil
}
