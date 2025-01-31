package forwarder

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhulik/fid/internal/core"
)

func FunctionMiddleware(backend core.ContainerBackend) gin.HandlerFunc {
	return func(c *gin.Context) {
		functionName := c.MustGet("functionName").(string) //nolint:forcetypeassert

		function, err := backend.Function(c.Request.Context(), functionName)
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
