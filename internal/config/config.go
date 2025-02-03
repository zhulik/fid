package config

import (
	"github.com/nats-io/nats.go"
)

type Config struct {
	HttpPort int `env:"HTTP_PORT" envDefault:"8180"` //nolint:stylecheck

	Functionname string `env:"FUNCTION_NAME"`

	NATSURL  string `env:"NATS_URL"`
	Loglevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func (c Config) NatsURL() string {
	if c.NATSURL == "" {
		return nats.DefaultURL
	}

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
