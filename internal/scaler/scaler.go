package scaler

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/elect"
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

// Run the scaler. When Shutdown is called, Run will return Stopped.
func (s Scaler) Run() error {
	s.logger.Info("Scaler started.")
	defer s.logger.Info("Scaler stopped.")

	for outcome := range s.elector.Start() {
		switch outcome.Status {
		case elect.Won:
			s.logger.Info("Elected as a leader")
		case elect.Lost:
			s.logger.Info("Lost leadership")
		case elect.Unknown:
			panic("should not happen")
		case elect.Error:
			return fmt.Errorf("elector error: %w", outcome.Error)
		case elect.Stopped:
			return Stopped
		}
	}

	panic("unreachable")
}

func (s Scaler) Shutdown() error {
	defer s.logger.Info("Scaler is shooting down")
	s.elector.Stop()

	return nil
}
