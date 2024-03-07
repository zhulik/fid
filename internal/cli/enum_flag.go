package cli

import (
	"errors"
	"fmt"
)

var errUnknownValue = errors.New("unknown value")

type EnumFlag struct {
	selected     string
	possible     []string
	defaultValue string
}

func (e *EnumFlag) Set(value string) error {
	for _, enum := range e.possible {
		if enum == value {
			e.selected = value

			return nil
		}
	}

	return fmt.Errorf("%w, allowed values are %v", errUnknownValue, supportedBackends)
}

func (e *EnumFlag) Get() any {
	return e.selected
}

func (e *EnumFlag) String() string {
	if e.selected == "" {
		return e.defaultValue
	}

	return e.selected
}
