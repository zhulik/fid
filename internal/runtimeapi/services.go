package runtimeapi

import (
	"github.com/samber/do"
)

func Register() {
	do.Provide(nil, NewServer)
}
