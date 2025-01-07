package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/do"
	"github.com/sirupsen/logrus"
)

func NewRouter(injector *do.Injector, logger logrus.FieldLogger) *gin.Engine {
	router := gin.New()

	router.Use(JSONRecovery())
	router.Use(LoggingMiddleware(logger))
	router.Use(JSONErrorHandler())

	router.GET("/health", func(c *gin.Context) {
		errs := injector.HealthCheck()

		for _, err := range errs {
			if err != nil {
				c.Error(err)
			}
		}
	})

	return router
}
