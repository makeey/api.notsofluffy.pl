package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

var (
	// Store will hold our session data
	Store *sessions.CookieStore
)

const (
	SessionName = "notsofluffy-session"
	SessionKey  = "session_id"
)

// InitSessionStore initializes the session store with a secret key
func InitSessionStore(secretKey string) {
	Store = sessions.NewCookieStore([]byte(secretKey))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 30, // 30 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
}

// SessionMiddleware handles session management
func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := Store.Get(c.Request, SessionName)
		if err != nil {
			// If session is corrupted, create a new one
			session = sessions.NewSession(Store, SessionName)
		}

		// Get or create session ID
		sessionID, ok := session.Values[SessionKey].(string)
		if !ok || sessionID == "" {
			// Generate new session ID
			sessionID = generateSessionID()
			session.Values[SessionKey] = sessionID
			session.IsNew = true
		}

		// Save session
		if err := session.Save(c.Request, c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			c.Abort()
			return
		}

		// Store session ID in context
		c.Set("session_id", sessionID)
		c.Set("session", session)

		c.Next()
	}
}

// generateSessionID generates a secure random session ID
func generateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return hex.EncodeToString([]byte("fallback-session-id"))
	}
	return hex.EncodeToString(bytes)
}

// GetSessionID gets the session ID from gin context
func GetSessionID(c *gin.Context) string {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return ""
	}
	return sessionID.(string)
}

// GetSession gets the session from gin context
func GetSession(c *gin.Context) *sessions.Session {
	session, exists := c.Get("session")
	if !exists {
		return nil
	}
	return session.(*sessions.Session)
}