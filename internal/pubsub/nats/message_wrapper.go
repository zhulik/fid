package nats

import (
	"github.com/nats-io/nats.go/jetstream"
)

// messageWrapper is a wrapper around jetstream.Msg to implement core.Message interface.
type messageWrapper struct {
	msg jetstream.Msg
}

func (m messageWrapper) Ack() error {
	return m.msg.Ack() //nolint:wrapcheck
}

func (m messageWrapper) Nak() error {
	return m.msg.Nak() //nolint:wrapcheck
}

func (m messageWrapper) Headers() map[string][]string {
	return m.msg.Headers()
}

func (m messageWrapper) Data() []byte {
	return m.msg.Data()
}
