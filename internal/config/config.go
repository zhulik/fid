package config

import (
	"log/slog"
)

type Config struct {
	HTTPPort           int
	FunctionName       string
	FunctionInstanceID string

	NATSURL     string
	LogLevel    slog.Level
	FidfilePath string
}
