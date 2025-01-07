package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
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

func (s Subscriber) Subscribe(
	ctx context.Context, consumerName, subject string, handler func(payload []byte, unsubscribe func()) error,
) error {
	cons, err := s.nats.jetStream.CreateConsumer(ctx, InvocationStreamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: subject,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	defer func() {
		if err := s.nats.jetStream.DeleteConsumer(ctx, InvocationStreamName, consumerName); err != nil {
			s.logger.WithError(err).Error("failed to delete consumer")
		}
	}()

	var sub jetstream.ConsumeContext

	// TODO: timeout
	sub, err = cons.Consume(func(msg jetstream.Msg) {
		result := msg.Data()

		err := handler(result, sub.Stop)
		// TODO: if handler panics, it should be recovered and the message should be nacked.
		if err != nil {
			// TODO: handler error somehow
			lo.Must0(msg.Nak())

			return
		}

		// TODO: handler error somehow
		lo.Must0(msg.Ack())
	})
	if err != nil {
		return fmt.Errorf("failed to consume: %w", err)
	}
	defer sub.Stop()

	<-sub.Closed()

	return nil
}
