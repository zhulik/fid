package httpserver

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	ReadHeaderTimeout = 5 * time.Second
)

func JSONRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// TODO: log errors
			if err := recover(); err != nil {
				c.IndentedJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

func JSONErrorHandler(logger logrus.FieldLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.WithError(err.Err).Errorf("Error during handling %s: %s", c.Request.URL.Path, err)
			}

			c.IndentedJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
		}
	}
}

// LoggingMiddleware logs each request's URI and method.
func LoggingMiddleware(logger logrus.FieldLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		defer func() {
			total := time.Since(start)
			logger.WithFields(logrus.Fields{
				"method":   c.Request.Method,
				"path":     c.Request.URL.Path,
				"duration": total,
				"status":   c.Writer.Status(),
			}).Infof("%s %s", c.Request.Method, c.Request.URL.Path)
		}()

		c.Next()
	}
}
