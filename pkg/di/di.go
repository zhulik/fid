package di

import (
	"github.com/samber/do"
)

func New() *do.Injector {
	injector := do.New()

	// TODO: inject config

	return injector
}
