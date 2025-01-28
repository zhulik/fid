package forwarder

import (
	"github.com/gin-gonic/gin"
)

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: write proper JWT middleware, get function name from the token.
		// A temporary solution to pass the function name to the handler.
		c.Set("functionName", c.GetHeader("function-name"))
		c.Next()
	}
}
