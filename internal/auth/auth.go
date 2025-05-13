package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/partadox/wags_queue/internal/config"
	"github.com/partadox/wags_queue/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrNoToken            = errors.New("no token provided")
)

// Authenticator handles authentication operations
type Authenticator struct {
	db         *sql.DB
	jwtSecret  []byte
	jwtExpires time.Duration
}

// NewAuthenticator creates a new Authenticator
func NewAuthenticator(db *sql.DB, cfg config.AuthConfig) *Authenticator {
	return &Authenticator{
		db:         db,
		jwtSecret:  []byte(cfg.JWTSecret),
		jwtExpires: cfg.JWTExpires,
	}
}

// Login authenticates a user and returns a JWT token
// Note: This method is now deprecated as we're using direct API key authentication
// Kept for backwards compatibility
func (a *Authenticator) Login(username, key string) (string, error) {
	// Query the user from the database
	var user models.User
	var apiKey string
	
	err := a.db.QueryRow("SELECT username, `key` FROM user WHERE username = ?", username).
		Scan(&user.Username, &apiKey)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("database error: %w", err)
	}
	
	// Compare API keys directly (no bcrypt for API keys)
	if apiKey != key {
		return "", ErrInvalidCredentials
	}
	
	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(a.jwtExpires).Unix(),
	})
	
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}
	
	return tokenString, nil
}

// GetUsernameFromAPIKey gets the username associated with an API key
func (a *Authenticator) GetUsernameFromAPIKey(apiKey string) (string, error) {
	// Query the database to find the username associated with the API key
	var username string
	err := a.db.QueryRow("SELECT username FROM user WHERE `key` = ?", apiKey).Scan(&username)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrInvalidToken
		}
		return "", fmt.Errorf("database error: %w", err)
	}
	
	return username, nil
}

// ExtractAPIKeyFromHeader extracts the API key from the X-Api-Key header
func ExtractAPIKeyFromHeader(r *http.Request) (string, error) {
	// Check X-Api-Key header
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey == "" {
		return "", ErrNoToken
	}
	
	return apiKey, nil
}

// Middleware creates a middleware that checks for a valid API key
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := ExtractAPIKeyFromHeader(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		username, err := a.GetUsernameFromAPIKey(apiKey)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		// Store the username in the request context
		ctx := r.Context()
		ctx = WithUsername(ctx, username)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HashPassword hashes a password using bcrypt
// Note: Not used for API keys as they are stored directly
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}
