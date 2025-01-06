package core

import (
	"io"

	"context"
	"github.com/samber/do"
)

type ContainerBackend interface {
	do.Healthcheckable
	do.Shutdownable

	Info(ctx context.Context) (map[string]any, error)

	Function(ctx context.Context, string string) (Function, error)
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
}

type Subscriber interface {
	do.Healthcheckable
	do.Shutdownable

	// Returning an error from the receiver means unsubscribing.
	// Returned errors except for ErrUnsubscribe are logged.
	Subscribe(ctx context.Context, subject string, receiver func([]byte) error) error
}
