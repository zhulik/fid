package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/pkg/json"
)

const (
	maxBytes = 10 * 1024 * 1024 // 10MB
	maxMsgs  = 1000
	maxAge   = 72 * time.Hour
)

var invocationStreamConfig = jetstream.StreamConfig{ //nolint:gochecknoglobals
	Name:      core.InvocationStreamName,
	Subjects:  []string{core.InvokeSubjectBase + ".*"},
	Storage:   jetstream.FileStorage,
	Retention: jetstream.WorkQueuePolicy,
	MaxAge:    maxAge,
	MaxMsgs:   maxMsgs,
	MaxBytes:  maxBytes,
	Replicas:  1,
}

var responseStreamConfig = jetstream.StreamConfig{ //nolint:gochecknoglobals
	Name:      core.ResponseStreamName,
	Subjects:  []string{core.ResponseStreamName + ".*"},
	Storage:   jetstream.FileStorage,
	Retention: jetstream.WorkQueuePolicy,
	MaxAge:    maxAge,
	MaxMsgs:   maxMsgs,
	MaxBytes:  maxBytes,
	Replicas:  1,
}

type Publisher struct {
	nats *Client

	logger     logrus.FieldLogger
	subscriber core.Subscriber
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

	subscriber, err := do.Invoke[core.Subscriber](injector)
	if err != nil {
		return nil, err
	}

	publisher := &Publisher{
		nats:       natsClient,
		logger:     logger,
		subscriber: subscriber,
	}

	err = publisher.createOrUpdateStreams(context.Background(), invocationStreamConfig, responseStreamConfig)
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

func (p Publisher) Publish(ctx context.Context, msg core.Msg) error {
	data, ok := msg.Data.([]byte)
	if !ok {
		var err error

		data, err = json.Marshal(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	message := nats.NewMsg(msg.Subject)
	message.Data = data
	message.Header = msg.Header

	_, err := p.nats.jetStream.PublishMsg(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// PublishWaitResponse Publishes a message to "subject", awaits for response on "subject.response".
// If payload is []byte, publishes as is, otherwise marshals to JSON.
func (p Publisher) PublishWaitResponse(ctx context.Context, msg core.Msg, responseInput core.PublishWaitResponseInput) ([]byte, error) { //nolint:lll
	replChan := lo.Async2(func() ([]byte, error) { return p.awaitResponse(ctx, responseInput) })

	if err := p.Publish(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to publish msg: %w", err)
	}

	p.logger.WithField("subject", msg.Subject).Debugf("Message sent, awaiting response")

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to consume response: %w", ctx.Err())
	case response := <-replChan:
		return response.Unpack()
	}
}

func (p Publisher) awaitResponse(ctx context.Context, responseInput core.PublishWaitResponseInput) ([]byte, error) {
	responseCtx, cancel := context.WithTimeout(ctx, responseInput.Timeout)
	defer cancel()

	response, err := p.subscriber.Next(responseCtx, responseInput.Stream, "", responseInput.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err = response.Ack(); err != nil {
		return nil, fmt.Errorf("failed to ack response: %w", err)
	}

	return response.Data(), nil
}

func (p Publisher) createOrUpdateStreams(ctx context.Context, streams ...jetstream.StreamConfig) error {
	for _, stream := range streams {
		logger := p.logger.WithField("streamName", stream.Name)

		_, err := p.nats.jetStream.CreateOrUpdateStream(ctx, stream)
		if err != nil {
			return fmt.Errorf("failed to create or update stream: %w", err)
		}

		logger.Info("Stream created or updated")
	}

	return nil
}
