package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/carlosindriago/agendadorplus/internal/domain"
	"github.com/gin-gonic/gin"
)

// HTTPErrorResponse is the standard error format returned to clients.
type HTTPErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MapDomainErrorToHTTP translates a domain.DomainError into an HTTP status code
// and sends the standardized JSON response.
func MapDomainErrorToHTTP(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// 1. Check if it's our typed domain error
	de, ok := domain.IsDomainError(err)
	if ok {
		var statusCode int
		switch de.Code {
		case "validation_error":
			statusCode = http.StatusBadRequest
		case "slot_locked", "slot_unavailable", "conflict":
			statusCode = http.StatusConflict
		case "not_found":
			statusCode = http.StatusNotFound
		case "invalid_credentials":
			statusCode = http.StatusUnauthorized
		default:
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, HTTPErrorResponse{
			Code:    de.Code,
			Message: de.Message,
		})
		return
	}

	// 2. Check context deadline/cancellation (Timeout)
	if errors.Is(err, contextExceededError) || errors.Is(err, contextCancelledError) {
		c.JSON(http.StatusRequestTimeout, HTTPErrorResponse{
			Code:    "timeout",
			Message: "the request took too long to process, please try again",
		})
		return
	}

	// 3. Fallback for unexpected errors
	// Important: We do not expose internal error details to the client
	// The actual error is automatically logged by Gin's default logger
	c.Error(err) // Attach the error to the context for Gin logger
	c.JSON(http.StatusInternalServerError, HTTPErrorResponse{
		Code:    "internal_error",
		Message: "an unexpected error occurred",
	})
}

// Dummy errors to match context timeout/cancelation
var contextExceededError = contextExceeded()
var contextCancelledError = contextCancelled()

func contextExceeded() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Nanosecond)
	return ctx.Err()
}

func contextCancelled() error {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx.Err()
}
