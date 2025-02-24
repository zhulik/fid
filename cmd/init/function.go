package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/zhulik/fid/internal/core"
)

var (
	ErrValidationFailed = errors.New("functions file validation failed")
	validate            = validator.New() //nolint:gochecknoglobals
)

func ParseFile(path string) (map[string]*Function, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read functions file: %w", err)
	}

	functionsConfig := Config{}

	err = yaml.Unmarshal(data, &functionsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", path, err)
	}

	for name, function := range functionsConfig.Functions {
		function.Name_ = name

		err := validate.Struct(function)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
		}
	}

	return functionsConfig.Functions, nil
}

type Function struct {
	Name_    string            `validate:"required"           yaml:"-"`
	Image_   string            `validate:"required"           yaml:"image"`
	Env_     map[string]string `yaml:"env"`
	Min      int64             `validate:"gte=0,ltefield=Max" yaml:"min"`
	Max      int64             `validate:"gte=0,gtefield=Min" yaml:"max"`
	Timeout_ time.Duration     `validate:"required,gte=1s"    yaml:"timeout"`
}

func (f Function) Name() string {
	return f.Name_
}

func (f Function) Image() string {
	return f.Image_
}

func (f Function) Timeout() time.Duration {
	return f.Timeout_
}

func (f Function) ScalingConfig() core.ScalingConfig {
	return core.ScalingConfig{
		Min: f.Min,
		Max: f.Max,
	}
}

func (f Function) Env() map[string]string {
	return f.Env_
}
