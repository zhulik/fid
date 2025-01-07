package httpserver

import (
	"encoding/json"
	"fmt"
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
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

func JSONErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			// TODO: log errors
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
		}
	}
}

func WriteJSON(doc any, w http.ResponseWriter, status int) error {
	jsonErr, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		return fmt.Errorf("failed marshal json: %w", err)
	}

	w.WriteHeader(status)

	_, err = w.Write(jsonErr)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
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
