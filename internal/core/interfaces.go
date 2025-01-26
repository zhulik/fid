package core

import (
	"context"
	"time"

	"github.com/samber/do"
)

type Config interface {
	ForwarderPort() int
	GatewayPort() int
	InfoServerPort() int
	NatsURL() string
	LogLevel() string
}

type ContainerBackend interface {
	do.Healthcheckable
	do.Shutdownable

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
	do.Healthcheckable
	do.Shutdownable

	Publish(ctx context.Context, subject string, msg any) error
	PublishWaitReply(ctx context.Context, subject string, payload any,
		header map[string][]string, replyTimeout time.Duration) ([]byte, error)
}

type Subscriber interface {
	do.Healthcheckable
	do.Shutdownable

	Fetch(ctx context.Context, consumerName, subject string) (Message, error)
}

type Message interface {
	Data() []byte
	Headers() map[string][]string
}
