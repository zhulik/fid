package cli

import (
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/core"
)

const defaultHTTPPort = 8080

const (
	flagNameNATSURL            = "nats-url"
	flagNameFunctionName       = "function-name"
	flagNameFunctionInstanceID = "function-instance-id"
	flagNameServerPort         = "port"
	flagNameLogLevel           = "log-level"
)

var (
	natsURLFlag = &cli.StringFlag{
		Name:    flagNameNATSURL,
		Aliases: []string{"n"},
		Usage:   "Nats `URL`, eg nats://127.0.0.1:4222",
		Value:   "nats://127.0.0.1:4222",
		Sources: cli.EnvVars(core.EnvNameNatsURL),
	}

	functionNameFlag = &cli.StringFlag{
		Name:     flagNameFunctionName,
		Aliases:  []string{"f"},
		Usage:    "Set function to `NAME`.",
		Sources:  cli.EnvVars(core.EnvNameFunctionName),
		Required: true,
	}

	functionInstanceIDFlag = &cli.StringFlag{
		Name:     flagNameFunctionInstanceID,
		Aliases:  []string{"fid"},
		Usage:    "Set function instance to `ID`.",
		Sources:  cli.EnvVars(core.EnvNameInstanceID),
		Required: true,
	}

	serverPortFlag = &cli.IntFlag{
		Name:    flagNameServerPort,
		Aliases: []string{"p"},
		Usage:   "Set server port tp `ID`.",
		Value:   defaultHTTPPort,
		Sources: cli.EnvVars("HTTP_PORT"),
	}

	logLevelFlag = &cli.StringFlag{
		Name:    flagNameLogLevel,
		Aliases: []string{"l"},
		Usage:   "Set log level to `LEVEL`.",
		Value:   "info",
		Sources: cli.EnvVars("LOG_LEVEL"),
	}
)
