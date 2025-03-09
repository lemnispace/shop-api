package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSONResponse sends a JSON response with the given status code and data
func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// ErrorResponse sends an error response with the given status code and message
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	ErrorResponseWithDetails(w, statusCode, message, nil)
}

// ErrorResponseWithDetails sends an error response with details as specified in API_DESIGN.md
func ErrorResponseWithDetails(w http.ResponseWriter, statusCode int, message string, details []map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

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

	err := json.NewEncoder(w).Encode(errorResponse)
	if err != nil {
		log.Printf("Error encoding error response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
