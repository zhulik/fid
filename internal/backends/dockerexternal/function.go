package dockerexternal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/zhulik/fid/internal/core"
)

type Function struct {
	container types.ContainerJSON

	timeout       time.Duration
	scalingConfig core.ScalingConfig
	env           map[string]string
}

func (f Function) Env() map[string]string {
	return f.env
}

func NewFunction(container types.ContainerJSON) (*Function, error) {
	timeoutStr := container.Config.Labels[core.LabelNameTimeout]
	timeout := core.DefaultTimeout

	if timeoutStr != "" {
		parsedTimeout, err := strconv.ParseInt(timeoutStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout: %w", err)
		}

		timeout = time.Duration(parsedTimeout) * time.Second
	}

	var err error

	minScaleStr := container.Config.Labels[core.LabelNameMinScale]
	minScale := int64(core.DefaultMinScale)

	if minScaleStr == "" {
		minScale, err = strconv.ParseInt(minScaleStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse min scale: %w", err)
		}
	}

	maxScaleStr := container.Config.Labels[core.LabelNameMaxScale]
	maxScale := int64(core.DefaultMaxScale)

	if maxScaleStr != "" {
		maxScale, err = strconv.ParseInt(maxScaleStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse max scale: %w", err)
		}
	}

	return &Function{
		container: container,
		timeout:   timeout,
		scalingConfig: core.ScalingConfig{
			Min: minScale,
			Max: maxScale,
		},
		env: parseEnv(container.Config.Env),
	}, nil
}

func parseEnv(env []string) map[string]string {
	envMap := make(map[string]string)

	for _, str := range env {
		parts := strings.SplitN(str, "=", 2) //nolint:mnd

		envMap[parts[0]] = parts[1]
	}

	return envMap
}

func (f Function) Name() string {
	return f.container.Name
}

func (f Function) Timeout() time.Duration {
	return f.timeout
}

func (f Function) ScalingConfig() core.ScalingConfig {
	return f.scalingConfig
}
