package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/pkg/core"
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
	Retention: jetstream.LimitsPolicy,
	MaxAge:    maxAge,
	MaxMsgs:   maxMsgs,
	MaxBytes:  maxBytes,
	Replicas:  1,
}

type Publisher struct {
	nats      *nats.Conn
	jetStream jetstream.JetStream

	logger logrus.FieldLogger
}

func NewPublisher(injector *do.Injector) (*Publisher, error) {
	config, err := do.Invoke[core.Config](injector)
	if err != nil {
		return nil, err
	}

	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "pubsub.nats.Publisher")

	defer logrus.Info("Nats publisher created.")

	natsClient, err := nats.Connect(config.NatsURL()) // TODO: from config
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS client: %w", err)
	}

	jetStream, err := jetstream.New(natsClient)
	if err != nil {
		return nil, fmt.Errorf("failed to build JetStream client: %w", err)
	}

	publisher := &Publisher{
		nats:      natsClient,
		jetStream: jetStream,
		logger:    logger,
	}

	err = publisher.createOrUpdateStreams(context.Background(), invocationStreamConfig)
	if err != nil {
		return nil, err
	}

	return publisher, nil
}

func (p Publisher) HealthCheck() error {
	p.logger.Debug("Publisher health check...")

	_, err := p.nats.GetClientID()
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	_, err = p.jetStream.AccountInfo(context.Background())
	if err != nil {
		return fmt.Errorf("healthcheck failed: %w", err)
	}

	return nil
}

func (p Publisher) Shutdown() error {
	p.logger.Debug("Shitting down nats publisher...")
	p.jetStream.CleanupPublisher()
	p.nats.Close()

	return nil
}

func (p Publisher) Publish(ctx context.Context, subject string, msg any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = p.jetStream.Publish(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// PublishWaitReply Publishes a message to "subject", awaits for response on "subject.reply".
func (p Publisher) PublishWaitReply(ctx context.Context, subject string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := nats.NewMsg(subject)
	msg.Data = data
	msg.Reply = subject + ".reply"

	_, err = p.jetStream.PublishMsg(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	cons, err := p.jetStream.CreateConsumer(ctx, InvocationStreamName, jetstream.ConsumerConfig{
		Name:          msg.Reply,
		FilterSubject: msg.Reply,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	defer func() {
		err := p.jetStream.DeleteConsumer(ctx, InvocationStreamName, msg.Reply)
		if err != nil {
			p.logger.WithError(err).Error("failed to delete consumer")
		}
	}()

	var result []byte

	var sub jetstream.ConsumeContext

	// TODO: timeout
	sub, err = cons.Consume(func(msg jetstream.Msg) {
		result = msg.Data()

		sub.Stop()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to consume: %w", err)
	}
	defer sub.Stop()

	<-sub.Closed()

	return result, nil
}

func (p Publisher) createOrUpdateStreams(ctx context.Context, streams ...jetstream.StreamConfig) error {
	for _, stream := range streams {
		_, err := p.jetStream.CreateStream(ctx, stream)
		if err != nil {
			if !errors.Is(err, jetstream.ErrStreamNameAlreadyInUse) {
				return fmt.Errorf("failed to create stream: %w", err)
			}
		} else {
			_, err = p.jetStream.UpdateStream(ctx, stream)
			if err != nil {
				return fmt.Errorf("failed to update stream: %w", err)
			}
		}
	}

	return nil
}
