package core

import (
	"time"
)

type PublishWaitResponseInput struct {
	Subject string
	Stream  string

	Msg Msg

	Timeout time.Duration
}
