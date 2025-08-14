package invocation

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/zhulik/fid/internal/core"
)

// TODO: move to pubusub?

func NewInvoker(injector do.Injector) (*Invoker, error) {
	return &Invoker{
		pubSuber: do.MustInvoke[core.PubSuber](injector),
		logger:   do.MustInvoke[*slog.Logger](injector).With("component", "invocation.Invoker"),
		kv:       do.MustInvoke[core.KV](injector),
	}, nil
}

type Invoker struct {
	pubSuber core.PubSuber
	kv       core.KV
	logger   *slog.Logger
}

func (i Invoker) HealthCheck() error {
	return nil
}

func (i Invoker) Shutdown() error {
	return nil
}

func (i Invoker) Invoke(ctx context.Context, function core.FunctionDefinition, payload []byte) ([]byte, error) {
	requestID := uuid.NewString()
	subject := i.pubSuber.InvokeSubjectName(function)
	deadline := time.Now().Add(function.Timeout()).UnixMilli()

	msg := core.Msg{
		Subject: subject,
		Data:    payload,
		Header: map[string][]string{
			core.HeaderNameRequestID:       {requestID},
			core.HeaderNameRequestDeadline: {strconv.FormatInt(deadline, 10)},
		},
	}

	errorSubject := i.pubSuber.ErrorSubjectName(function, requestID)

	responseInput := core.PublishWaitResponseInput{
		Subjects: []string{
			i.pubSuber.ResponseSubjectName(function, requestID),
			errorSubject,
		},
		Stream:  i.pubSuber.FunctionStreamName(function),
		Msg:     msg,
		Timeout: function.Timeout(),
	}

	i.logger.With("requestID", requestID, "function", function).Info("Invoking...")

	response, err := i.pubSuber.PublishWaitResponse(ctx, responseInput)
	if err != nil {
		return nil, fmt.Errorf("failed to publish and wait for response: %w", err)
	}

	data := response.Data()
	if response.Subject() == errorSubject {
		return nil, fmt.Errorf("%w: %s", core.ErrFunctionErrored, string(data))
	}

	return data, nil
}
