package cli

import (
	"context"

	"github.com/urfave/cli/v3"
	"github.com/zhulik/fid/internal/cli/flags"
	"github.com/zhulik/fid/internal/core"
	"github.com/zhulik/fid/internal/runtimeapi"
)

var runtimeapiCMD = &cli.Command{
	Name:     core.ComponentNameRuntimeAPI,
	Aliases:  []string{"ra"},
	Usage:    "Runtime api server is a component that runs as a side car with each function instance and mimics the AWS Lambda runtime API.", //nolint:lll
	Category: "Function",
	Flags: append(
		flags.ForServer,
		flags.FunctionName,
		flags.FunctionInstanceID,
	),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return runApp(ctx, cmd, runtimeapi.Provide())
	},
}
