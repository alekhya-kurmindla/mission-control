package config

import "os"


const (
	SOLDIER_USER     = "SOLDIER"
	SOLDIER_ACCESS   = "SOLDIER_ACCESS"
)

func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "supersecretkey123" //Test purpose
	}
	return []byte(secret)
}