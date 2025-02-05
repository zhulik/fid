package elect

import "time"

type Option func(*Config)

type Config struct {
	ID  []byte
	Key string

	Timeout time.Duration // Timeout for interactions with KV

	UpdateInterval time.Duration
	PollInterval   time.Duration
}
