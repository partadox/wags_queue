package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig
	DB          DBConfig
	Auth        AuthConfig
	ExternalAPI ExternalAPIConfig
}

// ServerConfig holds HTTP server related configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DBConfig holds database related configuration
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

// AuthConfig holds authentication related configuration
type AuthConfig struct {
	JWTSecret  string
	JWTExpires time.Duration
}

// ExternalAPIConfig holds configuration for the external message sending API
type ExternalAPIConfig struct {
	URL string
	Key string
}

// Load loads configuration from environment variables (.env file)
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Server config
	port := getEnv("SERVER_PORT", "8080")
	readTimeout, _ := strconv.Atoi(getEnv("SERVER_READ_TIMEOUT", "15"))
	writeTimeout, _ := strconv.Atoi(getEnv("SERVER_WRITE_TIMEOUT", "15"))
	idleTimeout, _ := strconv.Atoi(getEnv("SERVER_IDLE_TIMEOUT", "60"))

	// DB config
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "db_wags")

	// Auth config
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")
	jwtExpires, _ := strconv.Atoi(getEnv("JWT_EXPIRES", "24")) // hours

	// External API config
	externalAPIURL := getEnv("EXTERNAL_API_URL", "https://wag.artakusuma.com/api/clients")
	externalAPIKey := getEnv("EXTERNAL_API_KEY", "changeme")

	if jwtSecret == "your-secret-key" {
		fmt.Println("WARNING: Using default JWT secret key. This is insecure. Set JWT_SECRET environment variable.")
	}

	return &Config{
		Server: ServerConfig{
			Port:         port,
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
			IdleTimeout:  time.Duration(idleTimeout) * time.Second,
		},
		DB: DBConfig{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			Name:     dbName,
		},
		Auth: AuthConfig{
			JWTSecret:  jwtSecret,
			JWTExpires: time.Duration(jwtExpires) * time.Hour,
		},
		ExternalAPI: ExternalAPIConfig{
			URL: externalAPIURL,
			Key: externalAPIKey,
		},
	}, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
