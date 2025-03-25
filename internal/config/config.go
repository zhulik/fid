package config

import (
	"time"

	"github.com/samber/lo"
)

type Config struct {
	HTTPPort_ int

	FunctionName_       string
	FunctionInstanceID_ string

	NATSURL_  string
	LogLevel_ string
}

func (c Config) FunctionInstanceID() string {
	return c.FunctionInstanceID_
}

func (c Config) ElectionsBucketTTL() time.Duration {
	return lo.Must(time.ParseDuration("2s"))
} // TODO: use const everywhere

func (c Config) NATSURL() string {
	return c.NATSURL_
}

func (c Config) HTTPPort() int {
	return c.HTTPPort_
}

func (c Config) LogLevel() string {
	return c.LogLevel_
}

func (c Config) FunctionName() string {
	return c.FunctionName_
}
