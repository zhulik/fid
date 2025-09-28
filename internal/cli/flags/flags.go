package flags

import (
	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/core"
)

const defaultHTTPPort = 8080

const (
	FlagNameNATSURL            = "nats-url"
	FlagNameFunctionName       = "function-name"
	FlagNameFunctionInstanceID = "function-instance-id"
	FlagNameServerPort         = "port"
	FlagNameLogLevel           = "log-level"
	FlagNameBackend            = "backend"
	FlagNameDockerURL          = "docker-url"
	FlagNameFIDFile            = "fidfile"
	FlagNameInitOnly           = "init-only"
)

var (
	NatsURL = &cli.StringFlag{
		Name:    FlagNameNATSURL,
		Aliases: []string{"n"},
		Usage:   "Nats `URL`, eg nats://127.0.0.1:4222",
		Value:   "nats://127.0.0.1:4222",
		Sources: cli.EnvVars(core.EnvNameNatsURL),
	}

	FunctionName = &cli.StringFlag{
		Name:     FlagNameFunctionName,
		Aliases:  []string{"f"},
		Usage:    "Set function to `NAME`.",
		Sources:  cli.EnvVars(core.EnvNameFunctionName),
		Required: true,
	}

	FunctionInstanceID = &cli.StringFlag{
		Name:     FlagNameFunctionInstanceID,
		Aliases:  []string{"fid"},
		Usage:    "Set function instance to `ID`.",
		Sources:  cli.EnvVars(core.EnvNameInstanceID),
		Required: true,
	}

	ServerPort = &cli.IntFlag{
		Name:    FlagNameServerPort,
		Aliases: []string{"p"},
		Usage:   "Set server port to `PORT`.",
		Value:   defaultHTTPPort,
		Sources: cli.EnvVars("HTTP_PORT"),
	}

	LogLevel = &cli.StringFlag{
		Name:    FlagNameLogLevel,
		Aliases: []string{"l"},
		Usage:   "Set log level to `LEVEL`.",
		Value:   "info",
		Sources: cli.EnvVars("LOG_LEVEL"),
	}

	Backend = NewBackendFlag()

	DockerURL = &cli.StringFlag{
		Name:    FlagNameDockerURL,
		Aliases: []string{"du"},
		Usage:   "Set docker url `URL`. For docker backend only. Can be a TCP socket or a Unix socket.",
		Value:   "/var/run/docker.sock",
		Sources: cli.EnvVars("DOCKER_URL"),
	}

	Common = []cli.Flag{
		NatsURL,
		LogLevel,
	}

	ForServer = append(
		Common,
		ServerPort,
	)

	ForBackend = []cli.Flag{Backend, DockerURL}
)
