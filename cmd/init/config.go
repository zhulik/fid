package main

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

type Config struct {
	Version   int                  `validate:"required"                    yaml:"version"`
	Backend   string               `validate:"required,oneof=docker swarm" yaml:"backend"`
	Functions map[string]*Function `validate:"required,dive"               yaml:"functions"`

	Gateway    ServiceConfig `yaml:"gateway"`
	InfoServer ServiceConfig `yaml:"infoserver"`
}

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
	}

	err = validate.Struct(functionsConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	return functionsConfig.Functions, nil
}
