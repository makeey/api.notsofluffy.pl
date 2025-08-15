package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders middleware adds security headers for production
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request is from HTTPS (behind proxy)
		isHTTPS := isSecureRequest(c)

		// Security headers
		if isHTTPS {
			// HSTS (HTTP Strict Transport Security) - only for HTTPS
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// XSS Protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content Security Policy
		csp := buildCSP(isHTTPS)
		c.Header("Content-Security-Policy", csp)

		// Remove server information
		c.Header("Server", "")

		c.Next()
	}
}

// TrustedProxyHeaders middleware handles headers from trusted reverse proxy (Nginx)
func TrustedProxyHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set real IP from X-Real-IP or X-Forwarded-For headers
		if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
			c.Set("real_ip", realIP)
		} else if forwardedFor := c.GetHeader("X-Forwarded-For"); forwardedFor != "" {
			// Take the first IP from the X-Forwarded-For header
			ips := strings.Split(forwardedFor, ",")
			if len(ips) > 0 {
				c.Set("real_ip", strings.TrimSpace(ips[0]))
			}
		}

		// Set original protocol
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			c.Set("original_proto", proto)
		}

		// Set original host
		if host := c.GetHeader("X-Forwarded-Host"); host != "" {
			c.Set("original_host", host)
		} else if host := c.GetHeader("Host"); host != "" {
			c.Set("original_host", host)
		}

		c.Next()
	}
}

// CORSWithProxy middleware handles CORS with proxy support
func CORSWithProxy(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin || allowedOrigin == "*" {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400") // 24 hours
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestLogger middleware logs requests with real IP addresses
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Get real IP if available
		realIP := param.ClientIP
		if ip, exists := param.Keys["real_ip"]; exists {
			if ipStr, ok := ip.(string); ok {
				realIP = ipStr
			}
		}

		// Format log entry
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s %s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			realIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// isSecureRequest checks if the request is HTTPS (considering proxy headers)
func isSecureRequest(c *gin.Context) bool {
	// Check X-Forwarded-Proto header (set by Nginx)
	if proto := c.GetHeader("X-Forwarded-Proto"); proto == "https" {
		return true
	}

	// Check if TLS is enabled directly
	if c.Request.TLS != nil {
		return true
	}

	// Check X-Forwarded-SSL header
	if ssl := c.GetHeader("X-Forwarded-SSL"); ssl == "on" {
		return true
	}

	return false
}

// buildCSP builds Content Security Policy based on environment
func buildCSP(isHTTPS bool) string {
	protocol := "http:"
	if isHTTPS {
		protocol = "https:"
	}

	csp := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'", // Adjust as needed for your app
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: " + protocol,
		"font-src 'self' data:",
		"connect-src 'self' " + protocol,
		"media-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}, "; ")

	return csp
}

// RateLimitByIP creates a simple in-memory rate limiter by IP
func RateLimitByIP() gin.HandlerFunc {
	// This is a simple implementation
	// For production, consider using Redis or a proper rate limiting library
	
	return func(c *gin.Context) {
		// Simple rate limiting logic would go here
		// For now, just continue - implement proper rate limiting as needed
		c.Next()
	}
}

// APIKeyAuth middleware for API authentication (optional)
func APIKeyAuth(validAPIKeys []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Check query parameter as fallback
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Validate API key
		valid := false
		for _, validKey := range validAPIKeys {
			if apiKey == validKey {
				valid = true
				break
			}
		}

		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// HealthCheck middleware provides a simple health check endpoint
func HealthCheck(endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == endpoint {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().Unix(),
				"service":   "notsofluffy-api",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}