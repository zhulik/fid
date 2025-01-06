package config

type Config struct {
	HTTPPort        int `env:"HTTP_PORT"`
	WSServerPort_   int `env:"WS_PORT" envDefault:"8081"`
	InfoServerPort_ int `env:"INFO_PORT" envDefault:"8082"`
}

func (c Config) ProxyServerPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}
	return 8080
}

func (c Config) WSServerPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}
	return c.WSServerPort_
}

func (c Config) InfoServerPort() int {
	if c.HTTPPort != 0 {
		return c.HTTPPort
	}
	return c.InfoServerPort_
}
