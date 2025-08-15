package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Database configuration
	DatabaseURL string
	JWTSecret   string

	// HTTPS/SSL configuration
	EnableHTTPS     bool
	Domain          string
	CertCacheDir    string
	HTTPSPort       string
	HTTPPort        string
	ACMEEmail       string
	AllowedOrigins  []string

	// Database SSL configuration
	DBSSLMode     string
	DBSSLCert     string
	DBSSLKey      string
	DBSSLRootCert string

	// Development mode
	Development bool
}

func Load() *Config {
	cfg := &Config{
		// Basic configuration
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/notsofluffy?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),

		// HTTPS configuration
		EnableHTTPS:    getBoolEnv("ENABLE_HTTPS", false),
		Domain:         getEnv("DOMAIN", "localhost"),
		CertCacheDir:   getEnv("CERT_CACHE_DIR", "./certs"),
		HTTPSPort:      getEnv("HTTPS_PORT", "443"),
		HTTPPort:       getEnv("HTTP_PORT", "80"),
		ACMEEmail:      getEnv("ACME_EMAIL", ""),
		AllowedOrigins: getSliceEnv("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:3001"}),

		// Database SSL configuration
		DBSSLMode:     getEnv("DB_SSL_MODE", "disable"),
		DBSSLCert:     getEnv("DB_SSL_CERT", ""),
		DBSSLKey:      getEnv("DB_SSL_KEY", ""),
		DBSSLRootCert: getEnv("DB_SSL_ROOT_CERT", ""),

		// Development mode
		Development: getBoolEnv("DEVELOPMENT", true),
	}

	// Update database URL with SSL configuration if provided
	if cfg.DBSSLMode != "disable" {
		cfg.DatabaseURL = updateDatabaseURLWithSSL(cfg.DatabaseURL, cfg)
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// updateDatabaseURLWithSSL updates the database URL with SSL parameters
func updateDatabaseURLWithSSL(databaseURL string, cfg *Config) string {
	// Remove existing sslmode parameter
	if strings.Contains(databaseURL, "sslmode=") {
		parts := strings.Split(databaseURL, "?")
		if len(parts) == 2 {
			params := strings.Split(parts[1], "&")
			var newParams []string
			for _, param := range params {
				if !strings.HasPrefix(param, "sslmode=") &&
					!strings.HasPrefix(param, "sslcert=") &&
					!strings.HasPrefix(param, "sslkey=") &&
					!strings.HasPrefix(param, "sslrootcert=") {
					newParams = append(newParams, param)
				}
			}
			if len(newParams) > 0 {
				databaseURL = parts[0] + "?" + strings.Join(newParams, "&")
			} else {
				databaseURL = parts[0]
			}
		}
	}

	// Add SSL parameters
	var sslParams []string
	sslParams = append(sslParams, "sslmode="+cfg.DBSSLMode)

	if cfg.DBSSLCert != "" {
		sslParams = append(sslParams, "sslcert="+cfg.DBSSLCert)
	}
	if cfg.DBSSLKey != "" {
		sslParams = append(sslParams, "sslkey="+cfg.DBSSLKey)
	}
	if cfg.DBSSLRootCert != "" {
		sslParams = append(sslParams, "sslrootcert="+cfg.DBSSLRootCert)
	}

	if strings.Contains(databaseURL, "?") {
		databaseURL += "&" + strings.Join(sslParams, "&")
	} else {
		databaseURL += "?" + strings.Join(sslParams, "&")
	}

	return databaseURL
}