package config

import (
	"time"
)

type Config struct {
	HttpPort int `env:"HTTP_PORT" envDefault:"8180"` //nolint:stylecheck

	Functionname string `env:"FUNCTION_NAME"`

	NATSURL  string `env:"NATS_URL"`
	Loglevel string `env:"LOG_LEVEL" envDefault:"info"`

	ElectionsBucketTtl time.Duration `env:"ELECTIONS_BUCKET_TTL" envDefault:"2s"` //nolint:stylecheck
}

func (c Config) ElectionsBucketTTL() time.Duration {
	return c.ElectionsBucketTtl
}

func (c Config) NatsURL() string {
	return c.NATSURL
}

func (c Config) HTTPPort() int {
	return c.HttpPort
}

func (c Config) LogLevel() string {
	return c.Loglevel
}

func (c Config) FunctionName() string {
	return c.Functionname
}
