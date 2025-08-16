package config

import (
	"log/slog"
	"time"
)

type Config struct {
	HTTPPort           int
	FunctionName       string
	FunctionInstanceID string

	NATSURL            string
	LogLevel           slog.Level
	ElectionsBucketTTL time.Duration
	FidfilePath        string
}
