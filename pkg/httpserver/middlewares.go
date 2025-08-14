package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ReadHeaderTimeout = 5 * time.Second
)

func JSONRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
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

func JSONErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.Error("Error during handling", "url", c.Request.URL.Path, "error", err)
			}

			c.IndentedJSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
		}
	}
}

// LoggingMiddleware logs each request's URI and method.
func LoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		defer func() {
			total := time.Since(start)
			logger.With(
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"duration", total,
				"status", c.Writer.Status(),
			).Info("Request", "method", c.Request.Method, "path", c.Request.URL.Path)
		}()

		c.Next()
	}
}
