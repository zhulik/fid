package core

import (
	"context"
	"io"

	"github.com/samber/do"
)

type Config interface {
	WSServerPort() int
	ProxyServerPort() int
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

	Invoke(ctx context.Context, r io.Reader) ([]byte, error)
}

type Publisher interface {
	do.Healthcheckable
	do.Shutdownable

	Publish(ctx context.Context, subject string, msg any) error
	PublishWaitReply(ctx context.Context, subject string, payload any) ([]byte, error)
}

type Subscriber interface {
	do.Healthcheckable
	do.Shutdownable

	// If errors is returned from the handler, the message will be nacked.
	Subscribe(
		ctx context.Context, consumerName, subject string, handler func(payload []byte, unsubscribe func()) error,
	) error
}
