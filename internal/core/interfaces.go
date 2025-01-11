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
}

type Publisher interface {
	do.Healthcheckable
	do.Shutdownable

	Publish(ctx context.Context, subject string, msg any) error
	PublishWaitReply(ctx context.Context, subject string, payload any, replyTimeout time.Duration) ([]byte, error)
}

type Subscriber interface {
	do.Healthcheckable
	do.Shutdownable

	// If errors is returned from the handler, the message will be nacked.
	Subscribe(
		ctx context.Context, consumerName, subject string, handler func(payload []byte, unsubscribe func()) error,
	) error
}
