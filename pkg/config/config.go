package config

const (
	DefaultHTTPPort = 8080
)

type Config struct {
	HTTPPort int `env:"HTTP_PORT"`
	WSPort   int `env:"WS_PORT"   envDefault:"8081"`
	InfoPort int `env:"INFO_PORT" envDefault:"8082"`
}

func (c Config) ProxyServerPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}

	return DefaultHTTPPort
}

func (c Config) WSServerPort() int {
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
