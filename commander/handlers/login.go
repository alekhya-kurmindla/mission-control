package handlers

import (
	"encoding/json"
	"mission_control/commander/config"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// Generates a new JWT token
func generateJWT() (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"sub": "commander-user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.GetJWTSecret())
}

// LoginHandler returns a JWT token
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	token, err := generateJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
