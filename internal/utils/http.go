package utils

import (
	"encoding/json"
	"net/http"
)

// SendJSONResponse sends a JSON response with the given status code and data
func SendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	// Set content type header
	w.Header().Set("Content-Type", "application/json")

	// Set status code
	w.WriteHeader(statusCode)

	// Marshal and write the JSON response
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			ErrorLog("Failed to encode JSON response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
