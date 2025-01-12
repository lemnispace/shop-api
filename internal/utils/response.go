package utils

import (
	"encoding/json"
	"net/http"
)

func JSONResponse(w http.ResponseWriter, statusCode int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(payload)
}

func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	err := JSONResponse(w, statusCode, map[string]string{"error": message})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
