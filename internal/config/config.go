package config

import (
	"time"
)

type Config struct {
	HTTPPort_ int `env:"HTTP_PORT" envDefault:"8180"`

	FunctionName_ string `env:"FUNCTION_NAME"`

	NATSURL_  string `env:"NATS_URL"` //nolint:stylecheck
	LogLevel_ string `env:"LOG_LEVEL" envDefault:"info"`

	ElectionsBucketTTL_ time.Duration `env:"ELECTIONS_BUCKET_TTL" envDefault:"2s"`
}

func (c Config) ElectionsBucketTTL() time.Duration {
	return c.ElectionsBucketTTL_
}

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
