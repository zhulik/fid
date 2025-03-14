package core

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
)

type ServiceDependency interface {
	do.Healthchecker
	do.ShutdownerWithError
}

type Config interface {
	HTTPPort() int // For every component

	FunctionName() string       // For runtimeapi
	FunctionInstanceID() string // For runtimeapi

	NATSURL() string
	LogLevel() string

	ElectionsBucketTTL() time.Duration
}

type ContainerBackend interface {
	ServiceDependency

	Info(ctx context.Context) (map[string]any, error)

	Register(ctx context.Context, function FunctionDefinition) error
	Deregister(ctx context.Context, function FunctionDefinition) error

	StartGateway(ctx context.Context) (string, error)
	StartInfoServer(ctx context.Context) (string, error)

	AddInstance(ctx context.Context, function FunctionDefinition) (string, error)
	StopInstance(ctx context.Context, instanceID string) error
}

type FunctionsRepo interface {
	ServiceDependency

	Upsert(ctx context.Context, function FunctionDefinition) error
	Get(ctx context.Context, name string) (FunctionDefinition, error)
	List(ctx context.Context) ([]FunctionDefinition, error)
	Delete(ctx context.Context, name string) error
}

type InstancesRepo interface {
	ServiceDependency

	Add(ctx context.Context, function FunctionDefinition, id string) error
	SetLastExecuted(ctx context.Context, function FunctionDefinition, id string, timestamp time.Time) error
	SetBusy(ctx context.Context, function FunctionDefinition, id string, busy bool) error
	CountIdle(ctx context.Context, function FunctionDefinition) (int, error)

	Get(ctx context.Context, function FunctionDefinition, id string) (FunctionInstance, error)
	List(ctx context.Context, function FunctionDefinition) ([]FunctionInstance, error)
	Delete(ctx context.Context, function FunctionDefinition, id string) error
	Count(ctx context.Context, function FunctionDefinition) (int, error)
}

type FunctionDefinition interface {
	fmt.Stringer

	Name() string

	Image() string

	Timeout() time.Duration
	ScalingConfig() ScalingConfig

	Env() map[string]string
}

type FunctionInstance interface {
	ID() string
	LastExecuted() time.Time
	Busy() bool
	Function() FunctionDefinition
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

	CreateOrUpdateFunctionStream(ctx context.Context, function FunctionDefinition) error

	FunctionStreamName(function FunctionDefinition) string
	ScaleSubjectName(function FunctionDefinition) string
	InvokeSubjectName(function FunctionDefinition) string
	ResponseSubjectName(function FunctionDefinition, requestID string) string
	ErrorSubjectName(function FunctionDefinition, requestID string) string
}

type Invoker interface {
	ServiceDependency

	Invoke(ctx context.Context, function FunctionDefinition, payload []byte) ([]byte, error)
}

// Message is a message received from a pubsub system.
type Message interface {
	Subject() string
	Data() []byte
	Headers() map[string][]string
	Ack() error
	Nak() error
}

type KVBucket interface { //nolint:interfacebloat
	Name() string

	// Keys returns a list of keys in the bucket. Expensive if the bucket is big.
	// When an empty filters list is passed - returns all keys.
	Keys(ctx context.Context, filters ...string) ([]string, error)

	// Count returns a count of keys in the bucket. Expensive if the bucket is big.
	// When an empty filters list is passed - counts all keys.
	Count(ctx context.Context, filters ...string) (int, error)

	// All returns a list of values in the bucket. Expensive if the bucket is big.
	// When an empty filters list is passed - returns all entries.
	All(ctx context.Context, filters ...string) ([]KVEntry, error)

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
