package log

import (
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func init() {
	//Logger.SetReportCaller(true)
}
