package core

import (
	"io"

	"context"
	"github.com/samber/do"
)

type Backend interface {
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
