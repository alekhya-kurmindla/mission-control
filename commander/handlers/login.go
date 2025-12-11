package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"mission_control/commander/config"
	"net/http"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// Generates a new JWT token
type JWTResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginPayload struct {
	APIKey string `json:"api_key"`
	User   string `json:"user"`
}

func generateJWT(req LoginPayload) (*JWTResponse, error) {

	accessToken, refreshToken, err := getTokenAndRefreshToken(req.User)
	if err != nil {
		return nil, err
	}

	// Return both tokens
	return &JWTResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func getTokenAndRefreshToken(user string) (string, string, error) {

	//for soldier
	duration := 30 * time.Second

	if user == config.COMMANDER_USER {
		//access for 30 minutes
		duration = 30 * time.Minute
	}
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

	// --- Refresh Token (1 day) ---
	refreshClaims := jwt.MapClaims{
		"exp":  time.Now().Add(1 * 24 * time.Hour).Unix(),
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

func generateNewTokenByRefreshToken(refreshToken string) (*JWTResponse, error) {

	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		return config.GetJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

		// Example: read "sub"
		user := claims["user"].(string)

		expFloat, ok := claims["exp"].(float64)
		if !ok {
			log.Println("no exp claim found")
			return nil, errors.New("no exp claim found")
		}

		expTime := time.Unix(int64(expFloat), 0)
		now := time.Now()

		if now.After(expTime) {
			log.Println("Token is expired")
			return nil, errors.New("Refresh Token is expired")
		}

		accessToken, refreshToken, err := getTokenAndRefreshToken(user)
		if err != nil {
			return nil, err
		}

		// Return both tokens
		return &JWTResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, nil

	}

	return nil, errors.New("Invalid token")
}

// RefreshTokenRequest input
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func RefreshAccessToken(refreshToken string) (*JWTResponse, error) {
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

	if claims["type"] != "refresh" {
		return nil, errors.New("token is not a refresh token")
	}

	expUnix, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("invalid exp claim")
	}

	if int64(expUnix) < time.Now().Unix() {
		return nil, errors.New("refresh token expired")
	}

	newTokens, err := generateNewTokenByRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return newTokens, nil
}

// LoginHandler returns a JWT token
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginPayload
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid JSON",
		})
		return
	}

	if req.APIKey == "" || req.User == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad request. Required api_key and user",
		})
		return
	}

	if req.User != config.COMMANDER_USER && req.User != config.SOLDIER_USER {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad request. Invalid user",
		})
		return
	}

	if req.User == config.COMMANDER_USER && req.APIKey != os.Getenv("COMMANDER_API_KEY") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Unauthorized request. Invalid API_KEY",
		})
		return
	}

	if req.User == config.SOLDIER_USER && req.APIKey != os.Getenv("SOLDIER_API_KEY") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)

		json.NewEncoder(w).Encode(map[string]string{
			"message": "Unauthorized request. Invalid API_KEY",
		})

		return
	}

	token, err := generateJWT(req)
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

	token, err := RefreshAccessToken(req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]*JWTResponse{
		"token": token,
	})
}
