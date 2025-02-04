package elect

import "time"

type Option func(*Config)

type Config struct {
	ID             string
	Key            string
	UpdateInterval time.Duration
	PollInterval   time.Duration
}
