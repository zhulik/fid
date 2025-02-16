package nats

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/core"
)

type subscriptionWrapper struct {
	consumerCtx jetstream.ConsumeContext
	ch          chan core.Message
	logger      logrus.FieldLogger
}

func newSubscriptionWrapper(cons jetstream.Consumer, logger logrus.FieldLogger) (subscriptionWrapper, error) {
	msgChan := make(chan core.Message)

	consumerCtx, err := cons.Consume(func(msg jetstream.Msg) {
		msgChan <- messageWrapper{msg}
	})
	if err != nil {
		return subscriptionWrapper{}, fmt.Errorf("failed to consume: %w", err)
	}

	return subscriptionWrapper{
		consumerCtx: consumerCtx,
		ch:          msgChan,
		logger:      logger,
	}, nil
}

func (s subscriptionWrapper) C() <-chan core.Message {
	return s.ch
}

func (s subscriptionWrapper) Stop() {
	s.consumerCtx.Drain()
	s.consumerCtx.Stop()
	close(s.ch)
	s.logger.Info("Subscription stopped")
}
