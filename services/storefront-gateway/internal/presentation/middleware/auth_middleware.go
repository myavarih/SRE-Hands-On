package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	authv1 "github.com/myavarih/sre-hands-on/services/gen/go/auth/v1"
	"github.com/myavarih/sre-hands-on/services/storefront-gateway/internal/presentation/controllers"
)

type AuthMiddleware struct {
	jwtSecret  []byte
	authClient authv1.AuthServiceClient
}

func NewAuthMiddleware(jwtSecret string, authClient authv1.AuthServiceClient) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:  []byte(jwtSecret),
		authClient: authClient,
	}
}

type TokenClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			controllers.RespondError(c, http.StatusUnauthorized, "Authorization header missing", "")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			controllers.RespondError(c, http.StatusUnauthorized, "Invalid authorization header format (must be Bearer <token>)", "")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Fast path: verify JWT signature locally
		claims := &TokenClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.jwtSecret, nil
		})

		if err == nil && token.Valid {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
			c.Next()
			return
		}

		// Fallback path: verify via auth-service gRPC
		reqCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		resp, err := m.authClient.ValidateToken(reqCtx, &authv1.ValidateTokenRequest{
			Token: tokenString,
		})
		if err != nil || !resp.GetValid() {
			controllers.RespondError(c, http.StatusUnauthorized, "Invalid or expired token", "")
			c.Abort()
			return
		}

		c.Set("user_id", resp.GetUserId())
		c.Set("username", resp.GetUsername())
		c.Set("role", resp.GetRole())
		c.Next()
	}
}
