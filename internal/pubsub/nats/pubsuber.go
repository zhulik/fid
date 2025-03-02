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
	pubSuber := &PubSuber{
		nats:   do.MustInvoke[*Client](injector),
		logger: do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "pubsub.Nats.PubSuber"),
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

	_, err := p.nats.JetStream.PublishMsg(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// PublishWaitResponse Publishes a message to "subject", awaits for response on "subject.response".
// If payload is []byte, publishes as is, otherwise marshals to JSON.
func (p PubSuber) PublishWaitResponse(ctx context.Context, input core.PublishWaitResponseInput) (core.Message, error) { //nolint:lll
	replChan := lo.Async2(func() (core.Message, error) { return p.awaitResponse(ctx, input) })

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

func (p PubSuber) awaitResponse(ctx context.Context, input core.PublishWaitResponseInput) (core.Message, error) { //nolint:lll
	responseCtx, cancel := context.WithTimeout(ctx, input.Timeout)
	defer cancel()

	response, err := p.Next(responseCtx, input.Stream, input.Subjects, "")
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	response.Ack()

	return response, nil
}

// CreateOrUpdateFunctionStream creates or updates a stream for function invocation.
// TODO: something more universal, for any kind of streams?
func (p PubSuber) CreateOrUpdateFunctionStream(ctx context.Context, function core.FunctionDefinition) error {
	streamName := p.FunctionStreamName(function)
	logger := p.logger.WithField("streamName", streamName)

	cfg := jetstream.StreamConfig{
		Name: streamName,
		Subjects: []string{
			p.ScaleSubjectName(function),
			p.InvokeSubjectName(function),
			p.ResponseSubjectName(function, "*"),
			p.ErrorSubjectName(function, "*"),
		},
		Storage:   jetstream.FileStorage,
		Retention: jetstream.WorkQueuePolicy,
		MaxAge:    maxAge,
		MaxMsgs:   maxMsgs,
		MaxBytes:  maxBytes,
		Replicas:  1,
	}

	_, err := p.nats.JetStream.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create or update stream: %w", err)
	}

	logger.Info("Stream created or updated")

	return nil
}

// Next returns the next message from the stream, **does not respect ctx cancellation properly yet**,
// but checks ctx status when reaches timeout in the Nats client, so ctx cancellation will be
// respected in the next iteration.
func (p PubSuber) Next(ctx context.Context, streamName string, subjects []string, durableName string) (core.Message, error) { //nolint:lll
	var inactiveThreshold time.Duration
	if durableName != "" {
		inactiveThreshold = core.MaxTimeout
	}

	config := jetstream.ConsumerConfig{
		Durable:           durableName,
		FilterSubjects:    subjects,
		InactiveThreshold: inactiveThreshold,
	}

	cons, err := p.nats.JetStream.CreateOrUpdateConsumer(ctx, streamName, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumerCtx: %w", err)
	}

	if durableName == "" {
		durableName = cons.CachedInfo().Name
	}

	logger := p.logger.WithFields(logrus.Fields{
		"stream":      streamName,
		"consumerCtx": durableName,
		"subjects":    subjects,
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

func (p PubSuber) Subscribe(ctx context.Context, streamName string, subjects []string, durableName string) (core.Subscription, error) { //nolint:lll
	var inactiveThreshold time.Duration
	if durableName != "" {
		inactiveThreshold = core.MaxTimeout
	}

	config := jetstream.ConsumerConfig{
		Durable:           durableName,
		FilterSubjects:    subjects,
		InactiveThreshold: inactiveThreshold,
	}

	cons, err := p.nats.JetStream.CreateOrUpdateConsumer(ctx, streamName, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumerCtx: %w", err)
	}

	if durableName == "" {
		durableName = cons.CachedInfo().Name
	}

	logger := p.logger.WithFields(logrus.Fields{
		"stream":      streamName,
		"consumerCtx": durableName,
		"subjects":    subjects,
	})

	return newSubscriptionWrapper(cons, logger)
}

func (p PubSuber) FunctionStreamName(function core.FunctionDefinition) string {
	return fmt.Sprintf("%s:%s", core.StreamNameInvocation, function)
}

func (p PubSuber) ScaleSubjectName(function core.FunctionDefinition) string {
	return fmt.Sprintf("%s.%s", core.ScaleSubjectBase, function)
}

func (p PubSuber) InvokeSubjectName(function core.FunctionDefinition) string {
	return fmt.Sprintf("%s.%s", core.InvokeSubjectBase, function)
}

func (p PubSuber) ConsumeSubjectName(function core.FunctionDefinition) string {
	return fmt.Sprintf("%s.%s.consume", core.InvokeSubjectBase, function)
}

func (p PubSuber) ResponseSubjectName(function core.FunctionDefinition, requestID string) string {
	return fmt.Sprintf("%s.%s.%s.response", core.ResponseSubjectBase, function, requestID)
}

func (p PubSuber) ErrorSubjectName(function core.FunctionDefinition, requestID string) string {
	return fmt.Sprintf("%s.%s.%s.error", core.ResponseSubjectBase, function, requestID)
}
