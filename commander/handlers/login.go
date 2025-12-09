package handlers

import (
	"encoding/json"
	"errors"
	"mission_control/commander/config"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// Generates a new JWT token
type JWTResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func generateJWT() (*JWTResponse, error) {

	// --- Access Token (1 hour) ---
	accessClaims := jwt.MapClaims{
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"sub":  "commander-user",
		"type": "access",
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString(config.GetJWTSecret())
	if err != nil {
		return nil, err
	}

	// --- Refresh Token (7 days) ---
	refreshClaims := jwt.MapClaims{
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"sub":  "commander-user",
		"type": "refresh",
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString(config.GetJWTSecret())
	if err != nil {
		return nil, err
	}

	// Return both tokens
	return &JWTResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshTokenRequest input
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse output
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func RefreshAccessToken(refreshToken string) (*RefreshTokenResponse, error) {

	// Parse & validate refresh token
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		return config.GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// --- Check token type ---
	if claims["type"] != "refresh" {
		return nil, errors.New("token is not a refresh token")
	}

	// --- Check expiration ---
	expUnix, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid exp claim")
	}
	if int64(expUnix) < time.Now().Unix() {
		return nil, errors.New("refresh token expired")
	}

	// --- Issue new tokens ---
	newTokens, err := generateJWT()
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		AccessToken:  newTokens.AccessToken,
		RefreshToken: newTokens.RefreshToken,
	}, nil
}

// LoginHandler returns a JWT token
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	token, err := generateJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]*JWTResponse{
		"token": token,
	})
}

func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	res, err := RefreshAccessToken(req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
