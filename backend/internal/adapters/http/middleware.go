package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TimezoneMiddleware extracts the client's timezone from the "X-Timezone" header.
// If absent, defaults to "UTC".
func TimezoneMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tz := c.GetHeader("X-Timezone")
		if tz == "" {
			tz = "UTC"
		}
		
		// Validate that the timezone is a valid IANA timezone
		if _, err := time.LoadLocation(tz); err != nil {
			c.JSON(http.StatusBadRequest, HTTPErrorResponse{
				Code:    "invalid_timezone",
				Message: "the X-Timezone header must contain a valid IANA timezone (e.g. 'America/Lima')",
			})
			c.Abort()
			return
		}

		c.Set("client_timezone", tz)
		c.Next()
	}
}

// TimeoutMiddleware sets a strict timeout for the request.
func TimeoutMiddleware(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the request context with a timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		// Replace the request with the specific context
		c.Request = c.Request.WithContext(ctx)

		// Create a channel to wait for the handler execution
		finished := make(chan struct{})

		go func() {
			c.Next()
			close(finished)
		}()

		select {
		case <-finished:
			return
		case <-ctx.Done():
			// The timeout was exceeded or context canceled
			c.AbortWithStatusJSON(http.StatusRequestTimeout, HTTPErrorResponse{
				Code:    "timeout",
				Message: "request took too long to process (timeout)",
			})
			return
		}
	}
}

// AuthMiddleware validates the JWT token.
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, HTTPErrorResponse{
				Code:    "unauthorized",
				Message: "missing or invalid authorization header",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, domain.NewValidationError("unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, HTTPErrorResponse{
				Code:    "unauthorized",
				Message: "invalid or expired token",
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, HTTPErrorResponse{
				Code:    "unauthorized",
				Message: "invalid token claims",
			})
			c.Abort()
			return
		}

		// Set claims in context
		c.Set("user_id", claims["sub"])
		c.Set("tenant_id", claims["tenant_id"])
		
		c.Next()
	}
}
