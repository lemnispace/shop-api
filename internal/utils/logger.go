package utils

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

var (
	// IsDebug indicates if debug logging is enabled
	IsDebug = os.Getenv("DEBUG") == "true"
)

// DebugLog logs a message if debug mode is enabled
func DebugLog(format string, args ...interface{}) {
	if IsDebug {
		// Get caller information for better context
		_, file, line, _ := runtime.Caller(1)
		// Extract just the filename without the full path
		parts := strings.Split(file, "/")
		shortFile := parts[len(parts)-1]

		// Format the log with file, line and message
		logMsg := fmt.Sprintf(format, args...)
		log.Printf("[DEBUG] %s:%d - %s", shortFile, line, logMsg)
	}
}

// InfoLog logs an informational message
func InfoLog(format string, args ...interface{}) {
	// Get caller information for better context
	_, file, line, _ := runtime.Caller(1)
	// Extract just the filename without the full path
	parts := strings.Split(file, "/")
	shortFile := parts[len(parts)-1]

	// Format the log with file, line and message
	logMsg := fmt.Sprintf(format, args...)
	log.Printf("[INFO] %s:%d - %s", shortFile, line, logMsg)
}

// ErrorLog logs an error message
func ErrorLog(format string, args ...interface{}) {
	// Get caller information for better context
	_, file, line, _ := runtime.Caller(1)
	// Extract just the filename without the full path
	parts := strings.Split(file, "/")
	shortFile := parts[len(parts)-1]

	// Format the log with file, line and message
	logMsg := fmt.Sprintf(format, args...)
	log.Printf("[ERROR] %s:%d - %s", shortFile, line, logMsg)
}
