package config

import (
	"github.com/nats-io/nats.go"
)

const (
	DefaultHTTPPort = 8080
)

type Config struct {
	HTTPPort int    `env:"HTTP_PORT"`
	WSPort   int    `env:"WS_PORT"   envDefault:"8081"`
	InfoPort int    `env:"INFO_PORT" envDefault:"8082"`
	NATSURL  string `env:"NATS_URL"`
	Loglevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func (c Config) NatsURL() string {
	if c.NATSURL == "" {
		return nats.DefaultURL
	}

	return c.NATSURL
}

func (c Config) GatewayPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}

	return DefaultHTTPPort
}

func (c Config) ForwarderPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}

	return c.WSPort
}

func (c Config) InfoServerPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}

	return c.InfoPort
}

func (c Config) LogLevel() string {
	return c.Loglevel
}
