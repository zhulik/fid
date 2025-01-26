package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/json"
)

const (
	InvocationStreamName = "invocation"

	maxBytes = 10 * 1024 * 1024 // 10MB
	maxMsgs  = 1000
	maxAge   = 72 * time.Hour
)

var invocationStreamConfig = jetstream.StreamConfig{ //nolint:gochecknoglobals
	Name:      InvocationStreamName,
	Subjects:  []string{core.InvokeSubjectBase, core.InvokeSubjectBase + ".*"},
	Storage:   jetstream.FileStorage,
	Retention: jetstream.WorkQueuePolicy,
	MaxAge:    maxAge,
	MaxMsgs:   maxMsgs,
	MaxBytes:  maxBytes,
	Replicas:  1,
}

type Publisher struct {
	nats *Client

	logger logrus.FieldLogger
}

func NewPublisher(injector *do.Injector) (*Publisher, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "pubsub.nats.Publisher")

	natsClient, err := do.Invoke[*Client](injector)
	if err != nil {
		return nil, err
	}

	publisher := &Publisher{
		nats:   natsClient,
		logger: logger,
	}

	err = publisher.createOrUpdateStreams(context.Background(), invocationStreamConfig)
	if err != nil {
		return nil, err
	}

	return publisher, nil
}

func (p Publisher) HealthCheck() error {
	p.logger.Debug("Publisher health check...")

	err := p.nats.HealthCheck()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (p Publisher) Shutdown() error {
	return nil
}

func (p Publisher) Publish(ctx context.Context, subject string, msg any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = p.nats.jetStream.Publish(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// PublishWaitReply Publishes a message to "subject", awaits for response on "subject.reply".
// If payload is []byte, publishes as is, otherwise marshals to JSON.
func (p Publisher) PublishWaitReply(ctx context.Context, subject string, payload any, header map[string][]string, replyTimeout time.Duration) ([]byte, error) { //nolint:lll
	consumerName := uuid.New().String()
	replySubject := subject + ".reply"

	cons, err := p.nats.jetStream.CreateConsumer(ctx, InvocationStreamName, jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: replySubject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer for subject=%s: %w", replySubject, err)
	}

	p.logger.WithField("subject", replySubject).Debug("NATS Consumer created")

	defer func() { //nolint:contextcheck
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := p.nats.jetStream.DeleteConsumer(ctx, InvocationStreamName, consumerName); err != nil {
			p.logger.WithError(err).Errorf("Failed to delete consumer stream=%s consumer=%s", InvocationStreamName, consumerName)
		}

		p.logger.WithField("subject", replySubject).Debug("NATS Consumer for deleted")
	}()

	done, errChan := awaitReply(cons, replyTimeout)

	var data []byte

	var ok bool

	if data, ok = payload.([]byte); !ok {
		data, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	msg := nats.NewMsg(subject)
	msg.Data = data
	msg.Reply = replySubject
	msg.Header = header

	_, err = p.nats.jetStream.PublishMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish msg: %w", err)
	}

	p.logger.WithField("subject", subject).Debugf("Message sent, awaiting reply")

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to consume reply: %w", ctx.Err())
	case reply := <-done:
		return reply, nil
	case err = <-errChan:
		return nil, err
	}
}

func awaitReply(cons jetstream.Consumer, timeout time.Duration) (chan []byte, chan error) {
	done := make(chan []byte)
	errChan := make(chan error)

	go func() {
		reply, err := cons.Next(jetstream.FetchMaxWait(timeout))
		if err != nil {
			errChan <- fmt.Errorf("failed to consume reply: %w", err)

			return
		}
		done <- reply.Data()

		err = reply.Ack()
		if err != nil {
			errChan <- fmt.Errorf("failed to ack reply: %w", err)
		}
	}()

	return done, errChan
}

func (p Publisher) createOrUpdateStreams(ctx context.Context, streams ...jetstream.StreamConfig) error {
	for _, stream := range streams {
		logger := p.logger.WithField("streamName", stream.Name)

		_, err := p.nats.jetStream.CreateStream(ctx, stream)
		if err != nil {
			if errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
				_, err = p.nats.jetStream.UpdateStream(ctx, stream)
				if err != nil {
					return fmt.Errorf("failed to update stream: %w", err)
				}

				logger.Info("Stream updated")

				return nil
			}

			return fmt.Errorf("failed to create stream: %w", err)
		}

		logger.Info("Stream created")
	}

	return nil
}
