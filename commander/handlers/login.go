package handlers

import (
	"encoding/json"
	"errors"
	jwt "github.com/golang-jwt/jwt/v5"
	"log"
	"mission_control/commander/config"
	"net/http"
	"os"
	"time"
)

// JWTResponse represents the JSON response containing access + refresh tokens.
type JWTResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// LoginPayload represents the expected login JSON body.
type LoginPayload struct {
	APIKey string `json:"api_key"`
	User   string `json:"user"`
}

// generateAccessAndRefreshTokens generates a new access token and refresh token for the given user.
func generateAccessAndRefreshTokens(req LoginPayload) (*JWTResponse, error) {
	// Generate both access & refresh tokens.
	accessToken, refreshToken, err := getAccessTokenAndRefreshToken(req.User)
	if err != nil {
		return nil, err
	}
	// Return token response object.
	return &JWTResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// getAccessTokenAndRefreshToken creates an access token and a refresh token.
// Token expiry differs for COMMANDER vs SOLDIER.
func getAccessTokenAndRefreshToken(user string) (string, string, error) {
	// Default access token expiry (for soldier = 30 seconds)
	duration := 30 * time.Second
	// Commander gets 30-minute token
	if user == config.COMMANDER_USER {
		duration = 30 * time.Minute
	}
	// Create ACCESS TOKEN
	accessClaims := jwt.MapClaims{
		"exp":  time.Now().Add(duration).Unix(),
		"iat":  time.Now().Unix(),
		"user": user,
		"type": "access",
		"role": user + "_ACCESS",
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString(config.GetJWTSecret())
	if err != nil {
		return "", "", err
	}
	// Create REFRESH TOKEN (valid 24 hours)
	refreshClaims := jwt.MapClaims{
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"user": user,
		"type": "refresh",
		"role": user + "_ACCESS",
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString(config.GetJWTSecret())
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// generateNewTokenByRefreshToken validates the refresh token and issues a new pair.
func generateNewTokenByRefreshToken(refreshToken string) (*JWTResponse, error) {
	// Parse refresh token
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		return config.GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

		// Extract username
		user := claims["user"].(string)
		// Extract expiry timestamp
		expFloat, ok := claims["exp"].(float64)
		if !ok {
			log.Println("no exp claim found")
			return nil, errors.New("no exp claim found")
		}
		// Convert expiry
		expTime := time.Unix(int64(expFloat), 0)
		now := time.Now()

		// Reject expired refresh tokens
		if now.After(expTime) {
			log.Println("Token is expired")
			return nil, errors.New("refresh token is expired")
		}
		// Generate new token pair
		accessToken, refreshToken, err := getAccessTokenAndRefreshToken(user)
		if err != nil {
			return nil, err
		}
		return &JWTResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil
	}
	return nil, errors.New("invalid token")
}

// RefreshTokenRequest represents JSON input for /refresh.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshAccessToken validates a refresh token and issues a new token pair.
func RefreshAccessToken(refreshToken string) (*JWTResponse, error) {
	// Parse refresh token
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		return config.GetJWTSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}
	// Validate claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	// Ensure token type is "refresh"
	if claims["type"] != "refresh" {
		return nil, errors.New("token is not a refresh token")
	}
	// Validate expiry
	expUnix, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid exp claim")
	}
	if int64(expUnix) < time.Now().Unix() {
		return nil, errors.New("refresh token expired")
	}
	// Issue a new token pair
	newTokens, err := generateNewTokenByRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	return newTokens, nil
}

// LoginHandler handles /login API and returns JWT access + refresh tokens.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginPayload
	// Parse JSON body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid JSON"})
		return
	}
	// Validate required fields
	if req.APIKey == "" || req.User == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Bad request. Required api_key and user"})
		return
	}
	// Only allow known user types
	if req.User != config.COMMANDER_USER && req.User != config.SOLDIER_USER {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Bad request. Invalid user"})
		return
	}
	// Commander key check
	if req.User == config.COMMANDER_USER && req.APIKey != os.Getenv("COMMANDER_API_KEY") {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized request. Invalid API_KEY"})
		return
	}
	// Soldier key check
	if req.User == config.SOLDIER_USER && req.APIKey != os.Getenv("SOLDIER_API_KEY") {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized request. Invalid API_KEY"})
		return
	}
	// Generate JWT tokens
	token, err := generateAccessAndRefreshTokens(req)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	// Return token response
	json.NewEncoder(w).Encode(map[string]*JWTResponse{
		"token": token,
	})
}

// RefreshHandler handles /refresh API and returns a new token pair.
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req RefreshTokenRequest
	// Parse refresh token request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Bad request. Required refresh_token"})
		return
	}
	// Issue a new token
	token, err := RefreshAccessToken(req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// Return response JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]*JWTResponse{
		"token": token,
	})
}
