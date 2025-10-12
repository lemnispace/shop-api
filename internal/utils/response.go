package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// JSONResponse sends a JSON response with the given status code and data using Gin
func JSONResponse(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// ErrorResponse sends an error response with the given status code and message using Gin
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	ErrorResponseWithDetails(c, statusCode, message, nil)
}

// ErrorResponseWithDetails sends an error response with details as specified in API_DESIGN.md using Gin
func ErrorResponseWithDetails(c *gin.Context, statusCode int, message string, details []map[string]string) {
	// Create error code based on status
	var errorCode string
	switch statusCode {
	case http.StatusBadRequest:
		errorCode = "BAD_REQUEST"
	case http.StatusUnauthorized:
		errorCode = "UNAUTHORIZED"
	case http.StatusForbidden:
		errorCode = "FORBIDDEN"
	case http.StatusNotFound:
		errorCode = "NOT_FOUND"
	case http.StatusMethodNotAllowed:
		errorCode = "METHOD_NOT_ALLOWED"
	case http.StatusConflict:
		errorCode = "CONFLICT"
	case http.StatusUnprocessableEntity:
		errorCode = "VALIDATION_ERROR"
	case http.StatusTooManyRequests:
		errorCode = "RATE_LIMIT_EXCEEDED"
	default:
		errorCode = "INTERNAL_SERVER_ERROR"
	}

	// Format the error response according to API specifications
	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorCode,
			"message": message,
		},
	}

	// Add details if provided
	if details != nil {
		errorResponse["error"].(map[string]interface{})["details"] = details
	}

	c.JSON(statusCode, errorResponse)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// AbortWithError aborts the request with an error response
func AbortWithError(c *gin.Context, statusCode int, message string) {
	errorCode := "INTERNAL_SERVER_ERROR"
	switch statusCode {
	case http.StatusBadRequest:
		errorCode = "BAD_REQUEST"
	case http.StatusUnauthorized:
		errorCode = "UNAUTHORIZED"
	case http.StatusForbidden:
		errorCode = "FORBIDDEN"
	case http.StatusNotFound:
		errorCode = "NOT_FOUND"
	}

	c.AbortWithStatusJSON(statusCode, gin.H{
		"error": gin.H{
			"code":    errorCode,
			"message": message,
		},
	})
}

// ValidationError sends a validation error response
func ValidationError(c *gin.Context, field string, message string) {
	details := []map[string]string{
		{
			"field":   field,
			"message": message,
		},
	}
	ErrorResponseWithDetails(c, http.StatusUnprocessableEntity, "Validation failed", details)
}

// MultiValidationError sends a validation error response with multiple errors
func MultiValidationError(c *gin.Context, validationErrors []map[string]string) {
	ErrorResponseWithDetails(c, http.StatusUnprocessableEntity, "Validation failed", validationErrors)
}
