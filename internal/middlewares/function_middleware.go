package middlewares

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhulik/fid/internal/core"
)

func FunctionMiddleware(backend core.ContainerBackend, getName func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		functionName := getName(c)

		if functionName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "function name is required"})
			c.Abort()

			return
		}

		function, err := backend.Function(ctx, functionName)
		if err != nil {
			if errors.Is(err, core.ErrFunctionNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "function not found"})
				c.Abort()

				return
			}

			c.Error(err)
			c.Abort()

			return
		}

		c.Set("function", function)

		c.Next()
	}
}
