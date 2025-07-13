package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"notsofluffy-backend/internal/auth"
	"notsofluffy-backend/internal/database"

	"github.com/gin-gonic/gin"
)

// MaintenanceMiddleware checks if the site is in maintenance mode
// and redirects non-admin users to the coming soon page
func MaintenanceMiddleware(db *sql.DB, jwtSecret string) gin.HandlerFunc {
	settingsQueries := database.NewSettingsQueries(db)

	return func(c *gin.Context) {
		// Skip maintenance check for certain paths
		path := c.Request.URL.Path
		
		// Always allow access to admin routes, auth routes, maintenance status, and static files
		if strings.HasPrefix(path, "/api/admin") ||
			strings.HasPrefix(path, "/api/auth") ||
			strings.HasPrefix(path, "/api/maintenance-status") ||
			strings.HasPrefix(path, "/uploads") ||
			path == "/api/maintenance-status" {
			c.Next()
			return
		}

		// Check maintenance mode
		isMaintenanceMode, err := settingsQueries.GetMaintenanceMode()
		if err != nil {
			// If we can't check maintenance mode, allow access to prevent site lockout
			c.Next()
			return
		}

		// If not in maintenance mode, proceed normally
		if !isMaintenanceMode {
			c.Next()
			return
		}

		// Site is in maintenance mode - check if user is admin
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := authHeader[7:] // Remove "Bearer " prefix
			
			claims, err := auth.ValidateToken(tokenString, jwtSecret)
			if err == nil && claims.Role == "admin" {
				// Admin user, allow access
				c.Next()
				return
			}
		}

		// Non-admin user accessing site in maintenance mode
		// For API requests, return maintenance mode response
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":            "Site is under maintenance",
				"maintenance_mode": true,
			})
			c.Abort()
			return
		}

		// For non-API requests, this would be handled by the frontend
		// Since this is a backend API, we return a maintenance response
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":            "Site is under maintenance",
			"maintenance_mode": true,
		})
		c.Abort()
	}
}