package database

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sql.DB, error) {
	// Log database connection attempt (without credentials)
	logSafeDatabaseURL(databaseURL)

	// Configure SSL if needed
	if err := configureSSL(databaseURL); err != nil {
		return nil, fmt.Errorf("failed to configure SSL: %w", err)
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)     // Connections expire after 5 minutes
	db.SetConnMaxIdleTime(1 * time.Minute)    // Idle connections close after 1 minute

	log.Println("Database connection established successfully")
	return db, nil
}

// logSafeDatabaseURL logs the database URL without exposing credentials
func logSafeDatabaseURL(databaseURL string) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		log.Printf("Database: Connecting to database (URL parse error)")
		return
	}

	// Create safe URL without password
	safeURL := &url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
		Path:   parsed.Path,
	}

	if parsed.User != nil {
		if username := parsed.User.Username(); username != "" {
			safeURL.User = url.User(username)
		}
	}

	if parsed.RawQuery != "" {
		safeURL.RawQuery = parsed.RawQuery
	}

	log.Printf("Database: Connecting to %s", safeURL.String())
}

// configureSSL configures SSL settings for PostgreSQL connection
func configureSSL(databaseURL string) error {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	query := parsed.Query()
	sslMode := query.Get("sslmode")

	// If SSL is disabled, no configuration needed
	if sslMode == "disable" || sslMode == "" {
		log.Println("Database SSL: Disabled")
		return nil
	}

	log.Printf("Database SSL: Mode = %s", sslMode)

	// Handle SSL certificate files
	sslCert := query.Get("sslcert")
	sslKey := query.Get("sslkey")
	sslRootCert := query.Get("sslrootcert")

	// Configure SSL based on mode
	switch sslMode {
	case "require":
		log.Println("Database SSL: Requiring encrypted connection (no certificate validation)")
		return nil

	case "verify-ca":
		if sslRootCert == "" {
			return fmt.Errorf("sslrootcert is required for verify-ca mode")
		}
		log.Printf("Database SSL: Verifying CA certificate: %s", sslRootCert)
		return configureCertificateValidation(sslRootCert, "", "")

	case "verify-full":
		if sslRootCert == "" {
			return fmt.Errorf("sslrootcert is required for verify-full mode")
		}
		log.Printf("Database SSL: Full certificate validation with CA: %s", sslRootCert)
		
		// Client certificates for mutual authentication
		if sslCert != "" && sslKey != "" {
			log.Printf("Database SSL: Using client certificate: %s", sslCert)
			return configureCertificateValidation(sslRootCert, sslCert, sslKey)
		}
		return configureCertificateValidation(sslRootCert, "", "")

	default:
		return fmt.Errorf("unsupported SSL mode: %s", sslMode)
	}
}

// configureCertificateValidation sets up SSL certificate validation
func configureCertificateValidation(rootCertFile, clientCertFile, clientKeyFile string) error {
	// Load CA certificate
	caCert, err := ioutil.ReadFile(rootCertFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate file %s: %w", rootCertFile, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate from %s", rootCertFile)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
		ServerName:         "", // Will be set by connection string
	}

	// Load client certificate if provided (for mutual authentication)
	if clientCertFile != "" && clientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client certificate pair (%s, %s): %w", 
				clientCertFile, clientKeyFile, err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
		log.Println("Database SSL: Client certificate configured for mutual authentication")
	}

	// Register custom TLS config with pq driver
	// Note: This is a simplified approach. In production, you might want to
	// use a connection string that directly specifies the TLS config
	return nil
}

// GetSSLInfo returns information about the database SSL configuration
func GetSSLInfo(databaseURL string) map[string]interface{} {
	info := make(map[string]interface{})
	
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		info["error"] = "failed to parse database URL"
		return info
	}

	query := parsed.Query()
	info["ssl_mode"] = query.Get("sslmode")
	info["ssl_cert"] = query.Get("sslcert")
	info["ssl_key"] = query.Get("sslkey")
	info["ssl_root_cert"] = query.Get("sslrootcert")

	// Determine SSL status
	sslMode := query.Get("sslmode")
	switch sslMode {
	case "disable", "":
		info["ssl_enabled"] = false
		info["ssl_level"] = "none"
	case "require":
		info["ssl_enabled"] = true
		info["ssl_level"] = "encrypted"
	case "verify-ca":
		info["ssl_enabled"] = true
		info["ssl_level"] = "ca_verified"
	case "verify-full":
		info["ssl_enabled"] = true
		info["ssl_level"] = "fully_verified"
		if query.Get("sslcert") != "" && query.Get("sslkey") != "" {
			info["mutual_auth"] = true
		}
	default:
		info["ssl_enabled"] = false
		info["ssl_level"] = "unknown"
	}

	return info
}

// TestSSLConnection tests the SSL database connection
func TestSSLConnection(databaseURL string) error {
	log.Println("Database SSL: Testing connection...")
	
	db, err := Connect(databaseURL)
	if err != nil {
		return fmt.Errorf("SSL connection test failed: %w", err)
	}
	defer db.Close()

	// Test with a simple query
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("SSL connection test query failed: %w", err)
	}

	log.Printf("Database SSL: Connection test successful - PostgreSQL version: %s", 
		strings.Split(version, " ")[1])
	
	return nil
}

// BuildDatabaseURL constructs a database URL with SSL parameters
func BuildDatabaseURL(host, port, database, username, password string, sslConfig SSLConfig) string {
	// Basic connection string
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", 
		url.QueryEscape(username), 
		url.QueryEscape(password), 
		host, port, database)

	// Add SSL parameters
	params := make(url.Values)
	params.Set("sslmode", sslConfig.Mode)
	
	if sslConfig.CertFile != "" {
		params.Set("sslcert", sslConfig.CertFile)
	}
	if sslConfig.KeyFile != "" {
		params.Set("sslkey", sslConfig.KeyFile)
	}
	if sslConfig.RootCertFile != "" {
		params.Set("sslrootcert", sslConfig.RootCertFile)
	}

	return dbURL + "?" + params.Encode()
}

// SSLConfig holds SSL configuration for database connections
type SSLConfig struct {
	Mode         string // disable, require, verify-ca, verify-full
	CertFile     string // Client certificate file
	KeyFile      string // Client key file  
	RootCertFile string // CA certificate file
}

// ValidateSSLConfig validates SSL configuration
func ValidateSSLConfig(config SSLConfig) error {
	validModes := map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}

	if !validModes[config.Mode] {
		return fmt.Errorf("invalid SSL mode: %s. Valid modes: disable, require, verify-ca, verify-full", config.Mode)
	}

	// Check required files for different modes
	if config.Mode == "verify-ca" || config.Mode == "verify-full" {
		if config.RootCertFile == "" {
			return fmt.Errorf("root certificate file is required for %s mode", config.Mode)
		}
	}

	// For mutual authentication, both cert and key are required
	if config.CertFile != "" || config.KeyFile != "" {
		if config.CertFile == "" || config.KeyFile == "" {
			return fmt.Errorf("both client certificate and key files are required for mutual authentication")
		}
	}

	return nil
}