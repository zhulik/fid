package scaler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/elect"
	"github.com/zhulik/fid/pkg/json"
)

var Stopped = errors.New("stopped") //nolint:errname,gochecknoglobals,stylecheck

type Scaler struct {
	function core.FunctionDefinition
	logger   logrus.FieldLogger

	backend       core.ContainerBackend
	pubSuber      core.PubSuber
	instancesRepo core.InstancesRepo

	elector *elect.Elect
}

func NewScaler(function core.FunctionDefinition, injector *do.Injector) (*Scaler, error) {
	electID := uuid.NewString()

	config := do.MustInvoke[core.Config](injector)
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithFields(map[string]interface{}{
		"component": "scaler.Scaler",
		"function":  function,
		"electID":   electID,
	})
	backend := do.MustInvoke[core.ContainerBackend](injector)
	pubSuber := do.MustInvoke[core.PubSuber](injector)
	kv := do.MustInvoke[core.KV](injector)
	instancesRepo := do.MustInvoke[core.InstancesRepo](injector)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	bucket, err := kv.Bucket(ctx, core.BucketNameElections)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	kvWrap := kvWrapper{
		bucket: bucket,
	}

	keyName := function.Name() + "-scaler-leader"

	elector, err := elect.New(kvWrap, config.ElectionsBucketTTL(), keyName, electID)
	if err != nil {
		return nil, fmt.Errorf("failed to create elector: %w", err)
	}

	logger.Infof("Scaler created")

	return &Scaler{
		function:      function,
		logger:        logger,
		backend:       backend,
		pubSuber:      pubSuber,
		elector:       elector,
		instancesRepo: instancesRepo,
	}, nil
}

// Run the scaler. Never returns nil error. When Shutdown is called, returns Stopped.
func (s Scaler) Run() error { //nolint:cyclop,funlen
	s.logger.Info("Scaler started.")
	defer s.logger.Info("Scaler stopped.")

	var sub core.Subscription

	var err error

	stop := func() {
		if sub != nil {
			sub.Stop()
			sub = nil
		}

		s.logger.Info("Scaler unsubscribed")
	}

	errCh := make(chan error)
	defer close(errCh)

	defer stop()

	outcomeCh := s.elector.Start()

	for {
		select {
		case outcome := <-outcomeCh:
			switch outcome.Status {
			case elect.Won:
				s.logger.Info("Elected as a leader")

				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				sub, err = s.subscribe(ctx)

				cancel()

				if err != nil {
					return fmt.Errorf("failed to subscribe: %w", err)
				}

				go func() {
					err := s.runScaler(sub)
					if err != nil {
						errCh <- err
					}
				}()
				s.logger.Info("Scaler subscribed")
			case elect.Lost:
				s.logger.Info("Lost leadership")
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
	sub, err := s.pubSuber.Subscribe(ctx,
		s.pubSuber.FunctionStreamName(s.function),
		[]string{s.pubSuber.ScaleSubjectName(s.function)},
		s.subscriberName(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return sub, nil
}

func (s Scaler) subscriberName() string {
	return s.function.Name() + "-scaler"
}

func (s Scaler) Shutdown() error {
	defer s.logger.Info("Scaler is shooting down")
	s.elector.Stop()

	return nil
}

func (s Scaler) runScaler(sub core.Subscription) error {
	err := s.rescaleToConfig()
	if err != nil {
		return err
	}

	// TODO: subscribe to definition change and scale accordingly
	for msg := range sub.C() {
		s.logger.Debugf("Scaler received message: %s", msg.Data())

		req, err := json.Unmarshal[core.ScalingRequest](msg.Data())
		if err != nil {
			return fmt.Errorf("failed to scaling request: %w", err)
		}

		switch req.Type {
		case core.ScalingRequestTypeScaleUp:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				// TODO: reply to scale request with instance IDs
				_, err = s.scaleUp(ctx)
				if err != nil {
					s.logger.Errorf("Failed to scale up %+v", err)

					return
				}

				msg.Ack()
			}()
		case core.ScalingRequestTypeScaleDown:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), s.function.Timeout())
				defer cancel()

				// TODO: reply to scale request when deleted
				err = s.killInstance(ctx, req.InstanceID)
				if err != nil {
					s.logger.Errorf("Failed to scale down %+v", err)

					return
				}

				msg.Ack()
			}()

		default:
			s.logger.Warnf("Unknown scaling request type: %d", req.Type)
		}
	}

	return nil
}

func (s Scaler) rescaleToConfig() error {
	instances, err := s.instancesRepo.Count(context.Background(), s.function)
	if err != nil {
		return fmt.Errorf("failed to get instances: %w", err)
	}

	if instances < s.function.ScalingConfig().Min {
		toCreate := s.function.ScalingConfig().Min - instances
		s.logger.Info("%d instances running, minimum is %d, creating %d", instances, s.function.ScalingConfig().Min, toCreate)

		for range toCreate {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

			_, err = s.scaleUp(ctx)

			cancel()

			if err != nil {
				return fmt.Errorf("failed to scale up: %w", err)
			}
		}
	}

	if instances > s.function.ScalingConfig().Max {
		// TODO: implement
		toKill := instances - s.function.ScalingConfig().Max
		s.logger.Info("%d instances running, maximum is %d, killing %d", instances, s.function.ScalingConfig().Max, toKill)
	}

	return nil
}

func (s Scaler) killInstance(ctx context.Context, instanceID string) error {
	count, err := s.instancesRepo.Count(ctx, s.function)
	if err != nil {
		return fmt.Errorf("failed to get instance count: %w", err)
	}

	if count-1 <= s.function.ScalingConfig().Min {
		s.logger.Infof("cannot kill instances %s: too few instances left", instanceID)

		return nil
	}

	err = s.backend.KillInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to kill instance: %w", err)
	}

	err = s.instancesRepo.Delete(ctx, s.function, instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance record: %w", err)
	}

	return nil
}

func (s Scaler) scaleUp(ctx context.Context) (string, error) { //nolint:unparam
	s.logger.Info("Scaling up")

	instanceID, err := s.backend.AddInstance(ctx, s.function)
	if err != nil {
		return "", fmt.Errorf("failed to add instance: %w", err)
	}

	s.logger.Infof("Instance added %s", instanceID)

	return instanceID, nil
}
