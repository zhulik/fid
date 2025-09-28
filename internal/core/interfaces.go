package core

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type KVEntry struct {
	Key   string
	Value []byte
}

type PublishWaitResponseInput struct {
	Msg *nats.Msg

	Stream string // To listen for response on

	Subjects []string      // To listen for response on
	Timeout  time.Duration // Give up waiting after this duration
}

type ContainerBackend interface {
	Info(ctx context.Context) (map[string]any, error)

	Register(ctx context.Context, function FunctionDefinition) error
	Deregister(ctx context.Context, function FunctionDefinition) error

	StartGateway(ctx context.Context) (string, error)
	StartInfoServer(ctx context.Context) (string, error)

	AddInstance(ctx context.Context, function FunctionDefinition) (string, error)
	StopInstance(ctx context.Context, instanceID string) error
}

type FunctionsRepo interface {
	Upsert(ctx context.Context, function FunctionDefinition) error
	Get(ctx context.Context, name string) (FunctionDefinition, error)
	List(ctx context.Context) ([]FunctionDefinition, error)
	Delete(ctx context.Context, name string) error
}

type InstancesRepo interface {
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
	C() <-chan jetstream.Msg
	Stop()
}

type PubSuber interface {
	Publish(ctx context.Context, msg *nats.Msg) error
	PublishWaitResponse(ctx context.Context, responseInput PublishWaitResponseInput) (jetstream.Msg, error)
	Next(ctx context.Context, streamName string, subjects []string, durableName string) (jetstream.Msg, error)

	Subscribe(ctx context.Context, streamName string, subjects []string, durableName string) (Subscription, error)

	CreateOrUpdateFunctionStream(ctx context.Context, function FunctionDefinition) error

	FunctionStreamName(function FunctionDefinition) string
	InvokeSubjectName(function FunctionDefinition) string
	ResponseSubjectName(function FunctionDefinition, requestID string) string
	ErrorSubjectName(function FunctionDefinition, requestID string) string
}

type Invoker interface {
	Invoke(ctx context.Context, function FunctionDefinition, payload []byte) ([]byte, error)
}

type KVBucket interface {
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
}

type KV interface {
	CreateBucket(ctx context.Context, name string) (KVBucket, error)
	Bucket(ctx context.Context, name string) (KVBucket, error)
	DeleteBucket(ctx context.Context, name string) error
}
