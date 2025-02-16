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
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	electID := uuid.NewString()

	logger = logger.WithFields(map[string]interface{}{
		"component": "scaler.Scaler",
		"function":  function.Name(),
		"electID":   electID,
	})

	backend, err := do.Invoke[core.ContainerBackend](injector)
	if err != nil {
		return nil, err
	}

	pubSuber, err := do.Invoke[core.PubSuber](injector)
	if err != nil {
		return nil, err
	}

	kv, err := do.Invoke[core.KV](injector)
	if err != nil {
		return nil, err
	}

	kvWrap := kvWrapper{
		kv:     kv,
		bucket: function.Name() + "-elections",
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

func (s Scaler) subscribe(ctx context.Context) (core.Subscription, error) { //nolint:ireturn
	sub, err := s.pubSuber.Subscribe(ctx,
		s.pubSuber.FunctionStreamName(s.function.Name()),
		[]string{s.pubSuber.ScaleSubjectName(s.function.Name())},
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
		s.logger.Infof("Scaler received message: %s", msg.Data())

		msg.Ack() // TODO: move to the end

		req, err := json.Unmarshal[core.ScalingRequest](msg.Data())
		if err != nil {
			return fmt.Errorf("failed to scaling request: %w", err)
		}

		switch req.Type {
		case core.ScalingRequestTypeScaleUp:
			if req.Count == 0 {
				s.logger.Warn("Scaling up with 0 instances")

				continue
			}

			s.logger.Infof("Scaling up with %d instances", req.Count)
		case core.ScalingRequestTypeScaleDown:
			if req.Count == 0 {
				s.logger.Warn("Scaling up with 0 instances")

				continue
			}

			s.logger.Infof("Killing instances: %+v", req.InstanceIDs)

		default:
			s.logger.Warnf("Unknown scaling request type: %d", req.Type)
		}
	}

	return nil
}
