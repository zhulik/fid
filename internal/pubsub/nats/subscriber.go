package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

type msgWrapper struct {
	msg jetstream.Msg
}

func (m msgWrapper) Headers() map[string][]string {
	return m.msg.Headers()
}

func (m msgWrapper) Data() []byte {
	return m.msg.Data()
}

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

func (s Subscriber) Fetch(ctx context.Context, consumerName, subject string) (core.Message, error) { //nolint:ireturn
	cons, err := s.nats.jetStream.CreateOrUpdateConsumer(ctx, InvocationStreamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: subject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// TODO: respect ctx cancellation
	msg, err := cons.Next()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	return msgWrapper{msg}, nil
}
