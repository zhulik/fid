package flags

import (
	"fmt"

	"github.com/urfave/cli/v3"
)

var (
	supportedBackends = []string{"docker"}
	defaultBackend    = "docker"
)

func NewBackendFlag() cli.Flag {
	return &cli.GenericFlag{
		Name:    FlagNameBackend,
		Aliases: []string{"b"},
		Usage:   fmt.Sprintf("Set backend to `BACKEND`. Supported backends: %v", supportedBackends),
		Value: &EnumFlag{
			selected: defaultBackend,
			possible: supportedBackends,
		},
		Sources: cli.EnvVars("BACKEND"),
	}
}
