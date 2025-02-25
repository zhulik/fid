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
	function core.Function
	logger   logrus.FieldLogger

	backend  core.ContainerBackend
	pubSuber core.PubSuber

	elector *elect.Elect
}

func NewScaler(function core.Function, injector *do.Injector) (*Scaler, error) {
	electID := uuid.NewString()

	config := do.MustInvoke[core.Config](injector)
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithFields(map[string]interface{}{
		"component": "scaler.Scaler",
		"function":  function.Name(),
		"electID":   electID,
	})
	backend := do.MustInvoke[core.ContainerBackend](injector)
	pubSuber := do.MustInvoke[core.PubSuber](injector)
	kv := do.MustInvoke[core.KV](injector)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	bucket, err := kv.Bucket(ctx, function.Name()+"-elections")
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	kvWrap := kvWrapper{
		bucket: bucket,
		ttl:    config.ElectionsBucketTTL(),
	}

	elector, err := elect.New(kvWrap, "scaler-leader", electID)
	if err != nil {
		return nil, fmt.Errorf("failed to create elector: %w", err)
	}

	logger.Infof("Scaler created")

	return &Scaler{
		function: function,
		logger:   logger,
		backend:  backend,
		pubSuber: pubSuber,
		elector:  elector,
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
		s.function.Name()+"-scaler",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	return sub, nil
}

func (s Scaler) Shutdown() error {
	defer s.logger.Info("Scaler is shooting down")
	s.elector.Stop()

	return nil
}

func (s Scaler) runScaler(sub core.Subscription) error {
	for msg := range sub.C() {
		s.logger.Debugf("Scaler received message: %s", msg.Data())

		req, err := json.Unmarshal[core.ScalingRequest](msg.Data())
		if err != nil {
			return fmt.Errorf("failed to scaling request: %w", err)
		}

		switch req.Type {
		case core.ScalingRequestTypeScaleUp:
			if req.Count == 0 {
				s.logger.Warn("Scaling up with 0 instances")

				msg.Ack()

				continue
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()

				// TODO: reply to scale request with instance IDs
				err = s.scaleUp(ctx, req)
				if err != nil {
					s.logger.Errorf("Failed to scale up %+v", err)

					return
				}

				msg.Ack()
			}()
		case core.ScalingRequestTypeScaleDown:
			if len(req.InstanceIDs) == 0 {
				s.logger.Warn("Killing 0 instances")

				msg.Ack()

				continue
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), s.function.Timeout())
				defer cancel()

				// TODO: reply to scale request when deleted
				err = s.killInstances(ctx, req)
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

func (s Scaler) killInstances(ctx context.Context, req core.ScalingRequest) error {
	for _, instanceID := range req.InstanceIDs {
		err := s.backend.KillInstance(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("failed to kill instance: %w", err)
		}
	}

	return nil
}

func (s Scaler) scaleUp(ctx context.Context, req core.ScalingRequest) error {
	s.logger.Infof("Scaling up with %d instances", req.Count)
	instances := make([]string, req.Count)

	for i := range req.Count {
		instanceID, err := s.backend.AddInstance(ctx, s.function)
		if err != nil {
			return fmt.Errorf("failed to add instance: %w", err)
		}

		instances[i] = instanceID
	}

	return nil
}
