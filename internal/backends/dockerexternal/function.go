package dockerexternal

import (
	"fmt"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/zhulik/fid/internal/core"
)

type Function struct {
	container types.Container

	timeout       time.Duration
	scalingConfig core.ScalingConfig
}

func NewFunction(container types.Container) (*Function, error) {
	timeoutStr := container.Labels[core.LabelNameTimeout]
	timeout := core.DefaultTimeout

	if timeoutStr != "" {
		parsedTimeout, err := strconv.ParseInt(timeoutStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout: %w", err)
		}

		timeout = time.Duration(parsedTimeout) * time.Second
	}

	var err error

	minScaleStr := container.Labels[core.LabelNameMinScale]
	minScale := int64(core.DefaultMinScale)

	if minScaleStr == "" {
		minScale, err = strconv.ParseInt(minScaleStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse min scale: %w", err)
		}
	}

	maxScaleStr := container.Labels[core.LabelNameMaxScale]
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
	}, nil
}

func (f Function) Name() string {
	return f.container.Names[0]
}

func (f Function) Timeout() time.Duration {
	return f.timeout
}

func (f Function) ScalingConfig() core.ScalingConfig {
	return f.scalingConfig
}
