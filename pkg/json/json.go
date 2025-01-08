package json

import (
	libJSON "encoding/json"
	"fmt"

	"github.com/goccy/go-json"
)

func Unmarshal[T any](data []byte) (T, error) { //nolint:ireturn
	var result T

	err := libJSON.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}

func Marshal(data any) ([]byte, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return bytes, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return bytes, nil
}

func MarshalIndent(data any, prefix, indent string) ([]byte, error) {
	bytes, err := json.MarshalIndent(data, prefix, indent)
	if err != nil {
		return bytes, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return bytes, nil
}
