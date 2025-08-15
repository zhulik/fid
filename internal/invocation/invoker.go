package invocation

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zhulik/fid/internal/core"
)

// TODO: move to pubusub?

type Invoker struct {
	PubSuber core.PubSuber
	KV       core.KV
	Logger   *slog.Logger
}

func (i Invoker) Invoke(ctx context.Context, function core.FunctionDefinition, payload []byte) ([]byte, error) {
	requestID := uuid.NewString()
	subject := i.PubSuber.InvokeSubjectName(function)
	deadline := time.Now().Add(function.Timeout()).UnixMilli()

	msg := core.Msg{
		Subject: subject,
		Data:    payload,
		Header: map[string][]string{
			core.HeaderNameRequestID:       {requestID},
			core.HeaderNameRequestDeadline: {strconv.FormatInt(deadline, 10)},
		},
	}

	errorSubject := i.PubSuber.ErrorSubjectName(function, requestID)

	responseInput := core.PublishWaitResponseInput{
		Subjects: []string{
			i.PubSuber.ResponseSubjectName(function, requestID),
			errorSubject,
		},
		Stream:  i.PubSuber.FunctionStreamName(function),
		Msg:     msg,
		Timeout: function.Timeout(),
	}

	i.Logger.With("requestID", requestID, "function", function).Info("Invoking...")

	response, err := i.PubSuber.PublishWaitResponse(ctx, responseInput)
	if err != nil {
		return nil, fmt.Errorf("failed to publish and wait for response: %w", err)
	}

	data := response.Data()
	if response.Subject() == errorSubject {
		return nil, fmt.Errorf("%w: %s", core.ErrFunctionErrored, string(data))
	}

	return data, nil
}
