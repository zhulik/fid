package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func NewRouter(logger logrus.FieldLogger) *gin.Engine {
	router := gin.New()

	router.Use(JSONRecovery())
	router.Use(LoggingMiddleware(logger))
	router.Use(JSONErrorHandler())

	return router
}
