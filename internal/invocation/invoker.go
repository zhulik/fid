package invocation

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

func NewInvoker(injector *do.Injector) (*Invoker, error) {
	logger, err := do.Invoke[logrus.FieldLogger](injector)
	if err != nil {
		return nil, err
	}

	logger = logger.WithField("component", "invocation.Invoker")

	publisher, err := do.Invoke[core.Publisher](injector)
	if err != nil {
		return nil, err
	}

	return &Invoker{
		publisher: publisher,
		logger:    logger,
	}, nil
}

type Invoker struct {
	publisher core.Publisher
	logger    logrus.FieldLogger
}

func (i Invoker) HealthCheck() error {
	return nil
}

func (i Invoker) Shutdown() error {
	return nil
}

func (i Invoker) Invoke(ctx context.Context, function core.Function, payload []byte) ([]byte, error) {
	requestID := uuid.NewString()
	subject := fmt.Sprintf("%s.%s", core.InvokeSubjectBase, function.Name())
	deadline := time.Now().Add(function.Timeout()).UnixMilli()

	msg := core.Msg{
		Subject: subject,
		Data:    payload,
		Header: map[string][]string{
			core.RequestIDHeaderName:       {requestID},
			core.RequestDeadlineHeaderName: {strconv.FormatInt(deadline, 10)},
		},
	}

	replyInput := core.PublishWaitReplyInput{
		Subject: fmt.Sprintf("%s.%s", core.ReplySubjectBase, requestID),
		Stream:  core.ReplyStreamName,
		Timeout: function.Timeout(),
	}

	i.logger.WithFields(logrus.Fields{
		"requestID":    requestID,
		"functionName": function.Name(),
	}).Info("Invoking...")

	reply, err := i.publisher.PublishWaitReply(ctx, msg, replyInput)
	if err != nil {
		return nil, fmt.Errorf("failed to publish and wait for reply: %w", err)
	}

	return reply, nil
}
