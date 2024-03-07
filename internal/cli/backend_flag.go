package cli

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

var (
	supportedBackends = []string{"docker"}
	defaultBackend    = "docker"
)

func newBackendFlag() cli.Flag {
	return &cli.GenericFlag{
		Name:    flagNameBackend,
		Aliases: []string{"b"},
		Usage:   fmt.Sprintf("Set backend to `BACKEND`. Supported backends: %v", supportedBackends),
		Value: &EnumFlag{
			selected: defaultBackend,
		},
		Sources: cli.EnvVars("BACKEND"),
	}
}
