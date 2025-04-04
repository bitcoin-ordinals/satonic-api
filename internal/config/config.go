package config

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"encoding/base64"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Email    EmailConfig    `json:"email"`
	Auth     AuthConfig     `json:"auth"`
}

// ServerConfig contains server related configurations
type ServerConfig struct {
	Port int `json:"port"`
}

// DatabaseConfig contains database related configurations
type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// EmailConfig contains email service configurations
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
}

// AuthConfig contains authentication related configurations
type AuthConfig struct {
	JWTSecret     string `json:"jwt_secret"`
	JWTExpiration int    `json:"jwt_expiration"` // in hours
	CodeLength    int    `json:"code_length"`
	CodeExpiration int   `json:"code_expiration"` // in minutes
}

// Load loads the configuration from file and environment
func Load() (*Config, error) {
	// Default config
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			Host:   "localhost",
			Port:   5432,
			Name:   "satonic",
		},
		Email: EmailConfig{
			SMTPPort:  587,
			FromEmail: "noreply@satonic.com",
		},
		Auth: AuthConfig{
			JWTExpiration: 24,
			CodeLength:    6,
			CodeExpiration: 15,
		},
	}

	// Look for config file
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		// Use default config file path
		configFile = filepath.Join("configs", "config.json")
	}

	// Try to load config from file
	if _, err := os.Stat(configFile); err == nil {
		file, err := os.Open(configFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables if present
	if port := os.Getenv("SERVER_PORT"); port != "" {
		var serverPort int
		if _, err := fmt.Sscanf(port, "%d", &serverPort); err == nil {
			cfg.Server.Port = serverPort
		}
	}

	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		var databasePort int
		if _, err := fmt.Sscanf(dbPort, "%d", &databasePort); err == nil {
			cfg.Database.Port = databasePort
		}
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		cfg.Database.Password = dbPass
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.Database.Name = dbName
	}

	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		cfg.Email.SMTPHost = smtpHost
	}
	if smtpPort := os.Getenv("SMTP_PORT"); smtpPort != "" {
		var emailPort int
		if _, err := fmt.Sscanf(smtpPort, "%d", &emailPort); err == nil {
			cfg.Email.SMTPPort = emailPort
		}
	}
	if smtpUser := os.Getenv("SMTP_USER"); smtpUser != "" {
		cfg.Email.SMTPUser = smtpUser
	}
	if smtpPass := os.Getenv("SMTP_PASSWORD"); smtpPass != "" {
		cfg.Email.SMTPPassword = smtpPass
	}
	if fromEmail := os.Getenv("FROM_EMAIL"); fromEmail != "" {
		cfg.Email.FromEmail = fromEmail
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Auth.JWTSecret = jwtSecret
	} else if cfg.Auth.JWTSecret == "" {
		// Generate a random JWT secret if not provided
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, err
		}
		cfg.Auth.JWTSecret = base64.StdEncoding.EncodeToString(randomBytes)
	}

	return cfg, nil
} 