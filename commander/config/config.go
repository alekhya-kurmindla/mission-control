package config

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	COMMANDER_USER   = "COMMANDER"
	COMMANDER_ACCESS = "COMMANDER_ACCESS"
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

func IsTokenExpired(accessToken string) bool {
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		log.Println("Error parsing token:", err)
		return true
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("Invalid token claims")
		return true
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		log.Println("No exp claim found")
		return true
	}

	expTime := time.Unix(int64(expFloat), 0)
	now := time.Now()

	if now.After(expTime) {
		log.Println("Token is expired")
		return true
	} else {
		log.Println("Token is valid, expires at:", expTime)
		return false
	}
}
