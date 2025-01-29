package core

import (
	"time"
)

type PublishWaitReplyInput struct {
	Subject string
	Stream  string
	Timeout time.Duration
}
