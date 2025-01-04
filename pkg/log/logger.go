package log

import (
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}
