package elect

import "time"

type Option func(*Config)

type Config struct {
	ID             []byte
	Key            string
	UpdateInterval time.Duration
	PollInterval   time.Duration
}
