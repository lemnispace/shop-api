package middleware

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsAdmin(t *testing.T) {
	// Set up test cases
	tests := []struct {
		name          string
		adminEmails   string
		userEmail     string
		emailExists   bool
		expectedAdmin bool
	}{
		{
			name:          "Admin email in allowlist",
			adminEmails:   "admin@example.com,admin2@example.com",
			userEmail:     "admin@example.com",
			emailExists:   true,
			expectedAdmin: true,
		},
		{
			name:          "Admin email in allowlist with spaces",
			adminEmails:   "admin@example.com, admin2@example.com",
			userEmail:     "admin2@example.com",
			emailExists:   true,
			expectedAdmin: true,
		},
		{
			name:          "Non-admin email",
			adminEmails:   "admin@example.com,admin2@example.com",
			userEmail:     "user@example.com",
			emailExists:   true,
			expectedAdmin: false,
		},
		{
			name:          "Email not in context",
			adminEmails:   "admin@example.com",
			userEmail:     "",
			emailExists:   false,
			expectedAdmin: false,
		},
		{
			name:          "Empty admin emails env var",
			adminEmails:   "",
			userEmail:     "admin@example.com",
			emailExists:   true,
			expectedAdmin: false,
		},
		{
			name:          "No admin emails configured",
			adminEmails:   "",
			userEmail:     "",
			emailExists:   false,
			expectedAdmin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Setenv("ADMIN_EMAILS", tt.adminEmails)
			defer os.Unsetenv("ADMIN_EMAILS")

			// Create gin context
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(nil)

			// Set customer email in context if it exists
			if tt.emailExists {
				c.Set("customerEmail", tt.userEmail)
			}

			// Test IsAdmin function
			result := IsAdmin(c)

			if result != tt.expectedAdmin {
				t.Errorf("IsAdmin() = %v, want %v", result, tt.expectedAdmin)
			}
		})
	}
}
