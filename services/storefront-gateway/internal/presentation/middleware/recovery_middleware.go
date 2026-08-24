package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
)

func RecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logger.Error("Panic recovered in HTTP handler", "error", r, "stack", stack)
				controllers.RespondError(c, http.StatusInternalServerError, "Internal server error occurred", "")
				c.Abort()
			}
		}()
		c.Next()
	}
}
