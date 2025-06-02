// internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tradingbothub/platform/api/proto/auth"
)

func JWTAuth(authClient authpb.AuthServiceClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token with auth service
		req := &authpb.ValidateTokenRequest{
			AccessToken: token,
		}

		resp, err := authClient.ValidateToken(context.Background(), req)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if !resp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": resp.Error})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", resp.User.Id)
		c.Set("user", resp.User)

		c.Next()
	}
}
