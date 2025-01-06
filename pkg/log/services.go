package log

import (
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
)

func Register(injector *do.Injector) {
	do.Provide[logrus.FieldLogger](injector, func(_ *do.Injector) (logrus.FieldLogger, error) {
		return logrus.New(), nil
	})
}
