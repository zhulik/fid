package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

type Subscriber struct {
	nats *Client

	logger logrus.FieldLogger
}

func NewSubscriber(injector *do.Injector) (*Subscriber, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "pubsub.nats.Subscriber")

	natsClient, err := do.Invoke[*Client](injector)
	if err != nil {
		return nil, err
	}

	subscriber := &Subscriber{
		nats:   natsClient,
		logger: logger,
	}

	return subscriber, nil
}

func (s Subscriber) HealthCheck() error {
	s.logger.Debug("Publisher health check...")

	err := s.nats.HealthCheck()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (s Subscriber) Shutdown() error {
	return nil
}

func (s Subscriber) Next(ctx context.Context, //nolint:ireturn
	streamName, consumerName, subject string,
) (core.Message, error) {
	cons, err := s.nats.jetStream.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: subject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	defer func() {
		logger := s.logger.WithFields(logrus.Fields{
			"stream":   streamName,
			"consumer": consumerName,
			"subject":  subject,
		})
		if err := s.nats.jetStream.DeleteConsumer(ctx, streamName, consumerName); err != nil {
			logger.WithError(err).Error("Failed to delete consumer")
		}

		logger.Debug("NATS Consumer deleted")
	}()

	// TODO: respect ctx cancellation https://github.com/nats-io/nats.go/issues/1772
	msg, err := cons.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	return messageWrapper{msg}, nil
}
