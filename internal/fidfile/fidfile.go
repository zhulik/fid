package fidfile

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

var (
	ErrValidationFailed = errors.New("functions file validation failed")
	validate            = validator.New() //nolint:gochecknoglobals
)

type ServiceConfig struct {
	Port      int `validate:"required" yaml:"port"`
	Instances int `yaml:"instances"`
}

type Fidfile struct {
	Version   int                  `validate:"required,eq=1"               yaml:"version"`
	Backend   string               `validate:"required,oneof=docker swarm" yaml:"backend"`
	Functions map[string]*Function `validate:"required,dive"               yaml:"functions"`

	Gateway    *ServiceConfig `yaml:"gateway"`
	InfoServer *ServiceConfig `yaml:"infoserver"`
}

// TODO: add proper support for versioning

func ParseFile(path string) (*Fidfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read functions file: %w", err)
	}

	config, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fidfile %s: %w", path, err)
	}

	return config, nil
}

func Parse(data []byte) (*Fidfile, error) {
	functionsConfig := Fidfile{}

	err := yaml.Unmarshal(data, &functionsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal fidfile: %w", err)
	}

	for name, function := range functionsConfig.Functions {
		function.Name_ = name
	}

	err = validate.Struct(functionsConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	return &functionsConfig, nil
}
