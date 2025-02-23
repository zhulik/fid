package core

import (
	"fmt"
)

func MapToEnvList(maps ...map[string]string) []string {
	var envList []string

	for _, kv := range maps {
		for key, value := range kv {
			envList = append(envList, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return envList
}
