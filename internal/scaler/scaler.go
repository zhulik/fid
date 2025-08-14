package scaler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/elect"
	"github.com/zhulik/fid/pkg/json"
)

var Stopped = errors.New("stopped") //nolint:errname,gochecknoglobals,staticcheck

type Scaler struct {
	Function core.FunctionDefinition
	Logger   *slog.Logger

	Backend       core.ContainerBackend
	PubSuber      core.PubSuber
	InstancesRepo core.InstancesRepo

	Elector *elect.Elect
}

func NewScaler(ctx context.Context, injector do.Injector, function core.FunctionDefinition) (*Scaler, error) {
	electID := uuid.NewString()

	config := do.MustInvoke[config.Config](injector)
	logger := do.MustInvoke[*slog.Logger](injector).With(
		"component", "scaler.Scaler",
		"function", function,
		"electID", electID,
	)

	backend := do.MustInvoke[core.ContainerBackend](injector)
	pubSuber := do.MustInvoke[core.PubSuber](injector)
	kv := do.MustInvoke[core.KV](injector)
	instancesRepo := do.MustInvoke[core.InstancesRepo](injector)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	bucket, err := kv.Bucket(ctx, core.BucketNameElections)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	kvWrap := kvWrapper{
		bucket: bucket,
	}

	keyName := function.Name() + "-scaler-leader"

	elector, err := elect.New(kvWrap, config.ElectionsBucketTTL, keyName, electID)
	if err != nil {
		return nil, fmt.Errorf("failed to create elector: %w", err)
	}

	logger.Info("Scaler created")

	return &Scaler{
		Function:      function,
		Logger:        logger,
		Backend:       backend,
		PubSuber:      pubSuber,
		Elector:       elector,
		InstancesRepo: instancesRepo,
	}, nil
}

// Run the scaler. Never returns nil error. When Shutdown is called, returns Stopped.
func (s Scaler) Run() error { //nolint:cyclop,funlen
	s.Logger.Info("Scaler started.")
	defer s.Logger.Info("Scaler stopped.")

	var sub core.Subscription

	var err error

	stop := func() {
		if sub != nil {
			sub.Stop()
			sub = nil
		}

		s.Logger.Info("Scaler unsubscribed")
	}

	errCh := make(chan error)
	defer close(errCh)

	defer stop()

	outcomeCh := s.Elector.Start()

	for {
		select {
		case outcome := <-outcomeCh:
			switch outcome.Status {
			case elect.Won:
				s.Logger.Info("Elected as a leader")

				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				sub, err = s.subscribe(ctx)

				cancel()

				if err != nil {
					return fmt.Errorf("failed to subscribe: %w", err)
				}

				go func() {
					err := s.runScaler(context.Background(), sub)
					if err != nil {
						errCh <- err
					}
				}()
				s.Logger.Info("Scaler subscribed")
			case elect.Lost:
				s.Logger.Info("Lost leadership")
				stop()
			case elect.Error:
				return fmt.Errorf("elector error: %w", outcome.Error)
			case elect.Stopped:
				return Stopped
			case elect.Unknown:
				panic("should not happen")
			}

		case err := <-errCh:
			return fmt.Errorf("scaler failed: %w", err)
		}
	}
}

func (s Scaler) subscribe(ctx context.Context) (core.Subscription, error) {
	sub, err := s.PubSuber.Subscribe(ctx,
		s.PubSuber.FunctionStreamName(s.Function),
		[]string{s.PubSuber.ScaleSubjectName(s.Function)},
		s.subscriberName(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return sub, nil
}

func (s Scaler) subscriberName() string {
	return fmt.Sprintf("%s-%s", s.Function.Name(), core.ComponentNameScaler)
}

func (s Scaler) Shutdown() error {
	defer s.Logger.Info("Scaler is shooting down")
	s.Elector.Stop()

	return nil
}

func (s Scaler) runScaler(ctx context.Context, sub core.Subscription) error {
	err := s.rescaleToConfig(ctx)
	if err != nil {
		return err
	}

	// TODO: subscribe to definition change and scale accordingly
	for msg := range sub.C() {
		s.Logger.Debug("Scaler received message", "message", msg.Data())

		req, err := json.Unmarshal[core.ScalingRequest](msg.Data())
		if err != nil {
			return fmt.Errorf("failed to scaling request: %w", err)
		}

		switch req.Type {
		case core.ScalingRequestTypeScaleUp:
			go func() {
				ctx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()

				// TODO: reply to scale request with instance IDs
				_, err = s.scaleUp(ctx)
				if err != nil {
					s.Logger.Error("Failed to scale up", "error", err)

					return
				}

				msg.Ack()
			}()
		case core.ScalingRequestTypeScaleDown:
			go func() {
				ctx, cancel := context.WithTimeout(ctx, s.Function.Timeout())
				defer cancel()

				// TODO: reply to scale request when deleted
				err = s.stopInstance(ctx, req.InstanceID)
				if err != nil {
					s.Logger.Error("Failed to scale down", "error", err)

					return
				}

				msg.Ack()
			}()

		default:
			s.Logger.Warn("Unknown scaling request type", "type", req.Type)
		}
	}

	return nil
}

func (s Scaler) rescaleToConfig(ctx context.Context) error {
	instances, err := s.InstancesRepo.Count(ctx, s.Function)
	if err != nil {
		return fmt.Errorf("failed to get instances: %w", err)
	}

	switch {
	case instances < s.Function.ScalingConfig().Min:
		toCreate := s.Function.ScalingConfig().Min - instances
		s.Logger.Info("No need to rescale to config",
			"instances", instances,
			"min", s.Function.ScalingConfig().Min,
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
	case instances > s.Function.ScalingConfig().Max:
		// TODO: implement
		toKill := instances - s.Function.ScalingConfig().Max
		s.Logger.Info("No need to rescale to config",
			"instances", instances,
			"max", s.Function.ScalingConfig().Max,
			"toKill", toKill,
		)
	default:
		s.Logger.Info("No need to rescale to config",
			"instances", instances,
			"min", s.Function.ScalingConfig().Min,
			"max", s.Function.ScalingConfig().Max,
		)
	}

	return nil
}

func (s Scaler) stopInstance(ctx context.Context, instanceID string) error {
	count, err := s.InstancesRepo.Count(ctx, s.Function)
	if err != nil {
		return fmt.Errorf("failed to get instance count: %w", err)
	}

	if count-1 <= s.Function.ScalingConfig().Min {
		s.Logger.Info("cannot kill instances, too few instances left", "instanceID", instanceID)

		return nil
	}

	err = s.Backend.StopInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to kill instance: %w", err)
	}

	err = s.InstancesRepo.Delete(ctx, s.Function, instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance record: %w", err)
	}

	return nil
}

func (s Scaler) scaleUp(ctx context.Context) (string, error) { //nolint:unparam
	s.Logger.Info("Scaling up")

	instanceID, err := s.Backend.AddInstance(ctx, s.Function)
	if err != nil {
		return "", fmt.Errorf("failed to add instance: %w", err)
	}

	s.Logger.Info("Instance added", "instanceID", instanceID)

	return instanceID, nil
}
