package auth

import (
	"errors"
	"fmt"
	"mission_control/soldier/config"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
)

// ValidateToken validates JWT and ensures it is not expired
func ValidateToken(authHeader string) error {
	if authHeader == "" {
		return errors.New("missing token")
	}

	token, err := extractBearerToken(authHeader)
	if err != nil {
		return fmt.Errorf("invalid token: %v", err)
	}
	jwtSecret := config.GetJWTSecret()
	_, err = jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return fmt.Errorf("invalid or expired token: %v", err)
	}

	return nil
}

// extractBearerToken extracts the token from "Bearer <token>" format
func extractBearerToken(authHeader string) (string, error) {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return "", errors.New("empty authorization header")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header must be in the format Bearer <token>")
	}

	return parts[1], nil
}