package testhelpers

import (
	"github.com/samber/do/v2"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/config"
	"github.com/zhulik/fid/internal/core"
	natsPubSub "github.com/zhulik/fid/internal/pubsub/nats"
)

func NewInjector() do.Injector {
	injector := do.New()
	do.ProvideValue[logrus.FieldLogger](injector, logrus.New())

	do.ProvideValue[core.Config](injector, config.Config{})
	do.Provide(injector, natsPubSub.NewClient)

	return injector
}
