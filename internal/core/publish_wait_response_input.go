package core

import (
	"time"
)

type PublishWaitResponseInput struct {
	Subject string
	Stream  string
	Timeout time.Duration
}
