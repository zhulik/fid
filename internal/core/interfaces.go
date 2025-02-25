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
	HTTPPort() int // For every component

	FunctionName() string // For runtimeapi

	NatsURL() string
	LogLevel() string

	ElectionsBucketTTL() time.Duration
}

type ContainerBackend interface {
	ServiceDependency

	Info(ctx context.Context) (map[string]any, error)
	Register(ctx context.Context, function Function) error
	Deregister(ctx context.Context, name string) error

	StartGateway(ctx context.Context) (string, error)
	StartInfoServer(ctx context.Context) (string, error)

	Function(ctx context.Context, name string) (Function, error)
	Functions(ctx context.Context) ([]Function, error)

	AddInstance(ctx context.Context, function Function) (string, error)
	KillInstance(ctx context.Context, instanceID string) error
}

type Function interface {
	Name() string

	Image() string

	Timeout() time.Duration
	ScalingConfig() ScalingConfig

	Env() map[string]string
}

type Subscription interface {
	C() <-chan Message
	Stop()
}

type PubSuber interface { //nolint:interfacebloat
	ServiceDependency

	Publish(ctx context.Context, msg Msg) error
	PublishWaitResponse(ctx context.Context, responseInput PublishWaitResponseInput) (Message, error)
	Next(ctx context.Context, streamName string, subjects []string, durableName string) (Message, error)

	Subscribe(ctx context.Context, streamName string, subjects []string, durableName string) (Subscription, error)

	CreateOrUpdateFunctionStream(ctx context.Context, function Function) error

	FunctionStreamName(function Function) string
	ScaleSubjectName(function Function) string
	InvokeSubjectName(function Function) string
	ConsumeSubjectName(function Function) string
	ResponseSubjectName(function Function, requestID string) string
	ErrorSubjectName(function Function, requestID string) string
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

type KVBucket interface {
	Name() string

	All(ctx context.Context) ([]KVEntry, error)
	AllFiltered(ctx context.Context, filters ...string) ([]KVEntry, error)

	Get(ctx context.Context, key string) ([]byte, error)
	Create(ctx context.Context, key string, value []byte) (uint64, error)
	Put(ctx context.Context, key string, value []byte) error
	Update(ctx context.Context, key string, value []byte, seq uint64) (uint64, error)
	Delete(ctx context.Context, key string) error

	Incr(ctx context.Context, key string, n int64) (int64, error)
	Decr(ctx context.Context, key string, n int64) (int64, error)
}

type KV interface {
	ServiceDependency

	CreateBucket(ctx context.Context, name string, ttl time.Duration) (KVBucket, error)
	Bucket(ctx context.Context, name string) (KVBucket, error)
	DeleteBucket(ctx context.Context, name string) error
}
