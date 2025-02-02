package core

import (
	"time"
)

type PublishWaitResponseInput struct {
	Msg Msg

	Stream string // To listen for response on

	Subjects []string      // To listen for response on
	Timeout  time.Duration // Give up waiting after this duration
}
