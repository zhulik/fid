package nats

import (
	"context"
	"errors"
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
	maxBytes    = 10 * 1024 * 1024 // 10MB
	maxMsgs     = 1000
	maxAge      = 72 * time.Hour
	nextTimeout = 30 * time.Second
)

type PubSuber struct {
	nats *Client

	logger logrus.FieldLogger
}

func NewPubSuber(injector *do.Injector) (*PubSuber, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "pubsub.nats.PubSuber")

	natsClient, err := do.Invoke[*Client](injector)
	if err != nil {
		return nil, err
	}

	pubSuber := &PubSuber{
		nats:   natsClient,
		logger: logger,
	}

	return pubSuber, nil
}

func (p PubSuber) HealthCheck() error {
	p.logger.Debug("PubSuber health check...")

	err := p.nats.HealthCheck()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (p PubSuber) Shutdown() error {
	return nil
}

func (p PubSuber) Publish(ctx context.Context, msg core.Msg) error {
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
func (p PubSuber) PublishWaitResponse(ctx context.Context, input core.PublishWaitResponseInput) ([]byte, error) { //nolint:lll
	replChan := lo.Async2(func() ([]byte, error) { return p.awaitResponse(ctx, input) })

	if err := p.Publish(ctx, input.Msg); err != nil {
		return nil, fmt.Errorf("failed to publish msg: %w", err)
	}

	p.logger.WithField("subject", input.Msg.Subject).Debug("Message sent, awaiting response")

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("failed to consume response: %w", ctx.Err())
	case response := <-replChan:
		return response.Unpack()
	}
}

func (p PubSuber) awaitResponse(ctx context.Context, input core.PublishWaitResponseInput) ([]byte, error) {
	responseCtx, cancel := context.WithTimeout(ctx, input.Timeout)
	defer cancel()

	response, err := p.Next(responseCtx, input.Stream, input.Subject, "")
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err = response.Ack(); err != nil {
		return nil, fmt.Errorf("failed to ack response: %w", err)
	}

	return response.Data(), nil
}

// CreateOrUpdateFunctionStream creates or updates a stream for function invocation.
// TODO: something more universal, for any kind of streams?
func (p PubSuber) CreateOrUpdateFunctionStream(ctx context.Context, functionName string) error {
	streamName := p.FunctionStreamName(functionName)
	logger := p.logger.WithField("streamName", streamName)

	cfg := jetstream.StreamConfig{
		Name: streamName,
		Subjects: []string{
			p.InvokeSubjectName(functionName),
			p.ResponseSubjectName(functionName, "*"),
			p.ErrorSubjectName(functionName, "*"),
		},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    maxAge,
		MaxMsgs:   maxMsgs,
		MaxBytes:  maxBytes,
		Replicas:  1,
	}

	_, err := p.nats.jetStream.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create or update stream: %w", err)
	}

	logger.Info("Stream created or updated")

	return nil
}

// Next returns the next message from the stream, **does not respect ctx cancellation properly yet**,
// but checks ctx status when reaches timeout in the nats client, so ctx cancellation will be
// respected in the next iteration.
func (p PubSuber) Next(ctx context.Context, streamName, subject, durableName string) (core.Message, error) { //nolint:ireturn,lll
	var inactiveThreshold time.Duration
	if durableName != "" {
		inactiveThreshold = core.MaxTimeout
	}

	config := jetstream.ConsumerConfig{
		Durable:           durableName,
		FilterSubject:     subject,
		InactiveThreshold: inactiveThreshold,
	}

	cons, err := p.nats.jetStream.CreateOrUpdateConsumer(ctx, streamName, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	if durableName == "" {
		durableName = cons.CachedInfo().Name
	}

	logger := p.logger.WithFields(logrus.Fields{
		"stream":   streamName,
		"consumer": durableName,
		"subject":  subject,
	})

	logger.Debug("NATS Consumer created")

	var msg jetstream.Msg

	for {
		// TODO: respect ctx cancellation https://github.com/nats-io/nats.go/issues/1772
		msg, err = cons.Next(jetstream.FetchMaxWait(nextTimeout))
		if err == nil {
			break
		}

		if errors.Is(err, nats.ErrTimeout) {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
			}

			logger.Info("NATS Consumer timeout, resubscribing...")

			continue
		}

		return nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	return &messageWrapper{msg}, nil
}

func (p PubSuber) FunctionStreamName(functionName string) string {
	return fmt.Sprintf("%s:%s", core.InvocationStreamName, functionName)
}

func (p PubSuber) InvokeSubjectName(functionName string) string {
	return fmt.Sprintf("%s.%s", core.InvokeSubjectBase, functionName)
}

func (p PubSuber) ResponseSubjectName(functionName, requestID string) string {
	return fmt.Sprintf("%s.%s.%s.response", core.ResponseSubjectBase, functionName, requestID)
}

func (p PubSuber) ErrorSubjectName(functionName, requestID string) string {
	return fmt.Sprintf("%s.%s.%s.error", core.ResponseSubjectBase, functionName, requestID)
}
