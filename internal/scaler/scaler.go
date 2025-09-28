package scaler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/elect"
	"github.com/zhulik/fid/pkg/json"
)

var Stopped = errors.New("stopped") //nolint:errname,gochecknoglobals,staticcheck

type Scaler struct { //nolint:recvcheck
	Logger        *slog.Logger
	FunctionsRepo core.FunctionsRepo
	Config        *config.Config
	Backend       core.ContainerBackend
	PubSuber      core.PubSuber
	InstancesRepo core.InstancesRepo
	KV            core.KV
	elector       *elect.Elect

	function core.FunctionDefinition
}

func (s *Scaler) Init(ctx context.Context) error {
	electID := uuid.NewString()

	function, err := s.FunctionsRepo.Get(ctx, s.Config.FunctionName)
	if err != nil {
		return fmt.Errorf("failed to get function: %w", err)
	}

	s.Logger = s.Logger.With(
		"function", function,
		"electID", electID,
	)

	bucket, err := s.KV.Bucket(ctx, core.BucketNameElections)
	if err != nil {
		return fmt.Errorf("failed to get bucket: %w", err)
	}

	kvWrap := kvWrapper{
		bucket: bucket,
	}

	keyName := function.Name() + "-scaler-leader"

	elector, err := elect.New(kvWrap, s.Config.ElectionsBucketTTL, keyName, electID)
	if err != nil {
		return fmt.Errorf("failed to create elector: %w", err)
	}

	s.Logger.Info("Scaler created")

	s.function = function
	s.elector = elector

	return nil
}

// Run the scaler. Never returns nil error. When Shutdown is called, returns Stopped.
func (s Scaler) Run(ctx context.Context) error { //nolint:cyclop,funlen
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

	outcomeCh := s.elector.Start(ctx)

	for {
		select {
		case outcome := <-outcomeCh:
			switch outcome.Status {
			case elect.Won:
				s.Logger.Info("Elected as a leader")

				ctx, cancel := context.WithTimeout(ctx, time.Second)
				sub, err = s.subscribe(ctx)

				cancel()

				if err != nil {
					return fmt.Errorf("failed to subscribe: %w", err)
				}

				go func() {
					err := s.runScaler(ctx, sub)
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
		s.PubSuber.FunctionStreamName(s.function),
		[]string{s.PubSuber.ScaleSubjectName(s.function)},
		s.subscriberName(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return sub, nil
}

func (s Scaler) subscriberName() string {
	return fmt.Sprintf("%s-%s", s.function.Name(), core.ComponentNameScaler)
}

func (s Scaler) Shutdown(_ context.Context) error {
	defer s.Logger.Info("Scaler is shooting down")
	s.elector.Stop()

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
				ctx, cancel := context.WithTimeout(ctx, s.function.Timeout())
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

func (s Scaler) stopInstance(ctx context.Context, instanceID string) error {
	count, err := s.InstancesRepo.Count(ctx, s.function)
	if err != nil {
		return fmt.Errorf("failed to get instance count: %w", err)
	}

	if count-1 <= s.function.ScalingConfig().Min {
		s.Logger.Info("cannot kill instances, too few instances left", "instanceID", instanceID)

		return nil
	}

	err = s.Backend.StopInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to kill instance: %w", err)
	}

	err = s.InstancesRepo.Delete(ctx, s.function, instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance record: %w", err)
	}

	return nil
}

func (s Scaler) scaleUp(ctx context.Context) (string, error) { //nolint:unparam
	s.Logger.Info("Scaling up")

	instanceID, err := s.Backend.AddInstance(ctx, s.function)
	if err != nil {
		return "", fmt.Errorf("failed to add instance: %w", err)
	}

	s.Logger.Info("Instance added", "instanceID", instanceID)

	return instanceID, nil
}
