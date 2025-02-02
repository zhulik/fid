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

type PubSuber interface {
	ServiceDependency

	CreateOrUpdateFunctionStream(ctx context.Context, functionName string) error

	Publish(ctx context.Context, msg Msg) error
	PublishWaitResponse(ctx context.Context, responseInput PublishWaitResponseInput) (Message, error)
	Next(ctx context.Context, streamName string, subjects []string, durableName string) (Message, error)

	FunctionStreamName(functionName string) string
	InvokeSubjectName(functionName string) string
	ResponseSubjectName(functionName, requestID string) string
	ErrorSubjectName(functionName, requestID string) string
}

type Invoker interface {
	ServiceDependency

	CreateOrUpdateFunctionStream(ctx context.Context, function Function) error

	Invoke(ctx context.Context, function Function, payload []byte) ([]byte, error)
}

// Message is a message received from a pubsub system.
type Message interface {
	Subject() string
	Data() []byte
	Headers() map[string][]string
	Ack() error
	Nak() error
}
