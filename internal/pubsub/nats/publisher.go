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

var replyStreamConfig = jetstream.StreamConfig{ //nolint:gochecknoglobals
	Name:      core.ReplyStreamName,
	Subjects:  []string{core.ReplyStreamName + ".*"},
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

	err = publisher.createOrUpdateStreams(context.Background(), invocationStreamConfig, replyStreamConfig)
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
	requestID := header[core.RequestIDHeaderName][0]
	replySubject := fmt.Sprintf("%s.%s", core.ReplySubjectBase, requestID)

	replyCtx, cancel := context.WithTimeout(ctx, replyTimeout)
	defer cancel()

	replChan := lo.Async2(func() ([]byte, error) { return p.awaitReply(replyCtx, replySubject) })

	data, ok := payload.([]byte)
	if !ok {
		var err error

		data, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	msg := nats.NewMsg(subject)
	msg.Data = data
	msg.Reply = replySubject
	msg.Header = header

	_, err := p.nats.jetStream.PublishMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish msg: %w", err)
	}

	p.logger.WithField("subject", subject).Debugf("Message sent, awaiting reply")

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to consume reply: %w", ctx.Err())
	case reply := <-replChan:
		return reply.Unpack()
	}
}

func (p Publisher) awaitReply(ctx context.Context, subject string) ([]byte, error) {
	reply, err := p.subscriber.Next(ctx, core.ReplySubjectBase, "", subject)
	if err != nil {
		return nil, fmt.Errorf("failed to read reply: %w", err)
	}

	if err = reply.Ack(); err != nil {
		return nil, fmt.Errorf("failed to ack reply: %w", err)
	}

	return reply.Data(), nil
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
