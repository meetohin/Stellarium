// internal/middleware/recovery.go
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Log the panic
		logrus.WithFields(logrus.Fields{
			"panic":      recovered,
			"stack":      string(debug.Stack()),
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Error("Panic recovered")

		// Return error response
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"message": "An unexpected error occurred",
		})
	})
}
