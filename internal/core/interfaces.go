package core

import (
	"context"
	"time"

	"github.com/samber/do"
)

type ServiceDependency interface {
	do.Healthcheckable
	do.Shutdownable
}

type Config interface {
	ForwarderPort() int
	GatewayPort() int
	InfoServerPort() int
	NatsURL() string
	LogLevel() string
}

type ContainerBackend interface {
	ServiceDependency

	Info(ctx context.Context) (map[string]any, error)

	Function(ctx context.Context, name string) (Function, error)
	Functions(ctx context.Context) ([]Function, error)
}

type Function interface {
	Name() string

	Timeout() time.Duration
	ScalingConfig() ScalingConfig

	Env() map[string]string
}

type Publisher interface {
	ServiceDependency

	Publish(ctx context.Context, subject string, msg any) error
	PublishWaitReply(ctx context.Context, subject string, payload any,
		header map[string][]string, replyTimeout time.Duration) ([]byte, error)
}

type Subscriber interface {
	ServiceDependency

	Next(ctx context.Context, streamName, consumerName, subject string) (Message, error)
}

// Message is a message received from a pubsub system.
type Message interface {
	Data() []byte
	Headers() map[string][]string
	Ack() error
	Nak() error
}
