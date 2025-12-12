package config

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	COMMANDER_USER   = "COMMANDER"        // Commander user type
	COMMANDER_ACCESS = "COMMANDER_ACCESS" // Commander access role
	SOLDIER_USER     = "SOLDIER"          // Soldier user type
	SOLDIER_ACCESS   = "SOLDIER_ACCESS"   // Soldier access role
)

// Returns the JWT secret key
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "supersecretkey123" // Default secret for testing
	}
	return []byte(secret)
}

// Checks if the JWT access token is expired
func IsTokenExpired(accessToken string) bool {
	// Parse token without signature verification
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		log.Println("Error parsing token:", err)
		return true
	}
	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("Invalid token claims")
		return true
	}
	// Read expiration time
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		log.Println("No exp claim found")
		return true
	}
	// Convert exp to time
	expTime := time.Unix(int64(expFloat), 0)
	now := time.Now()
	// Check if expired
	if now.After(expTime) {
		log.Println("Token is expired")
		return true
	} else {
		log.Println("Token is valid, expires at:", expTime)
		return false
	}
}
