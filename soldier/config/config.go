package config

import "os"

const (
	SOLDIER_USER     = "SOLDIER"
	SOLDIER_ACCESS   = "SOLDIER_ACCESS"
)

// GetJWTSecret returns the JWT secret key from environment variables
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "supersecretkey123" //Test purpose
	}
	return []byte(secret)
}