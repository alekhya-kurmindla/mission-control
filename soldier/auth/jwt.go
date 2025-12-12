package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mission_control/soldier/config"
	"mission_control/soldier/models"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type TokenStore struct {
	AuthToken    string // Stores access token
	RefreshToken string // Stores refresh token
	mu           sync.RWMutex // Mutex for thread-safe access
}

var tokens = &TokenStore{} // Global token store

func SetTokens(auth, refresh string) {
	tokens.mu.Lock()            // Lock for writing
	defer tokens.mu.Unlock()    // Unlock after update
	tokens.AuthToken = auth     // Set auth token
	tokens.RefreshToken = refresh // Set refresh token
}

func GetTokens() (string, string) {
	tokens.mu.RLock()          // Lock for reading
	defer tokens.mu.RUnlock()  // Unlock after read
	return tokens.AuthToken, tokens.RefreshToken // Return tokens
}

// ValidateToken validates JWT token syntactically and checks expiration
func ValidateToken(authHeader string) error {
	if authHeader == "" { // Empty header check
		return errors.New("missing token")
	}

	token, err := extractBearerToken(authHeader) // Extract token from Bearer header
	if err != nil {
		return fmt.Errorf("invalid token: %v", err)
	}

	jwtSecret := config.GetJWTSecret() // Get signing secret

	_, err = jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil // Validate token signature
	})

	if err != nil { // Invalid or expired
		return fmt.Errorf("invalid or expired token: %v", err)
	}

	return nil
}

// extractBearerToken extracts token from "Bearer <token>"
func extractBearerToken(authHeader string) (string, error) {
	authHeader = strings.TrimSpace(authHeader) // Trim spaces
	if authHeader == "" {
		return "", errors.New("empty authorization header")
	}

	parts := strings.Fields(authHeader) // Split header
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header must be in the format Bearer <token>")
	}

	return parts[1], nil // Return token only
}

// isTokenExpired checks if JWT expiration time has passed
func isTokenExpired(accessToken string) (bool, error) {
	if accessToken == "" { // Token empty check
		log.Println("token is empty")
		return true, errors.New("token is empty")
	}

	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{}) // Parse without verifying
	if err != nil {
		log.Println("Error parsing token:", err)
		return true, err
	}
	claims, ok := token.Claims.(jwt.MapClaims) // Extract claims
	if !ok {
		log.Println("invalid token claims")
		return true, errors.New("invalid token claims")
	}
	expFloat, ok := claims["exp"].(float64) // Read exp field
	if !ok {
		log.Println("no exp claim found")
		return true, errors.New("no exp claim found")
	}
	expTime := time.Unix(int64(expFloat), 0) // Convert to time
	now := time.Now()                       // Current time

	if now.After(expTime) { // Compare expiration
		log.Println("Token is expired")
		return true, nil
	} else {
		log.Println("Token is valid, expires at:", expTime)
		return false, nil
	}
}

// RefreshToken makes a refresh request using the refresh token
func RefreshToken(ctx context.Context, refreshToken string) error {
	commanderURL := os.Getenv("COMMANDER_URL") // Base commander URL
	creds := map[string]string{
		"refresh_token": refreshToken, // Prepare request payload
	}
	body, _ := json.Marshal(creds) // Convert to JSON

	req, _ := http.NewRequest("POST", commanderURL+"/refresh", bytes.NewBuffer(body)) // Build request
	req.Header.Set("Content-Type", "application/json") // Set JSON header

	err := makeAuthCall(req) // Execute refresh call
	if err != nil {
		log.Println("Got an error: ", err.Error())
		return err
	}
	return nil
}

// makeAuthCall sends HTTP request and stores returned tokens
func makeAuthCall(req *http.Request) error {
	client := &http.Client{} // HTTP client
	resp, err := client.Do(req) // Send request
	if err != nil {
		log.Println("Got an error: ", err.Error())
		return err
	}
	defer resp.Body.Close() // Close response body

	respBytes, _ := io.ReadAll(resp.Body) // Read response

	var loginResp models.LoginResponse // Parse login response
	json.Unmarshal(respBytes, &loginResp)

	SetTokens(loginResp.Token.AccessToken, loginResp.Token.RefreshToken) // Store new tokens
	return nil
}

// ValidateSoldier ensures the access token is valid and user has soldier permissions
func ValidateSoldier(ctx context.Context) error {
	authToken, refreshToken := GetTokens() // Get current tokens

	isExpired, err := isTokenExpired(authToken) // Check expiration
	if err != nil {
		log.Println("Got an error in ExecuteMission isTokenExpired: ", err.Error())
		return err
	}
	if isExpired { // If expired then refresh
		log.Println("Soldier token has been expired. Generating new token using refresh token")

		err = RefreshToken(ctx, refreshToken) // Call refresh endpoint
		if err != nil {
			log.Println("Got an error in ExecuteMission RefreshToken: ", err.Error())
			return err
		}

		authToken, _ = GetTokens() // Load new tokens
	}
	if authToken != "" { // Ensure token available

		token, err := jwt.Parse(authToken, func(t *jwt.Token) (interface{}, error) {
			return config.GetJWTSecret(), nil // Validate signature
		})

		if err != nil || !token.Valid { // Invalid token
			return errors.New("invalid or expired token")
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			user := claims["user"].(string) // Extract user
			role := claims["role"].(string) // Extract role

			if user != config.SOLDIER_USER { // Check correct user
				return errors.New("only soldier can perform this action")
			}

			if role != config.SOLDIER_ACCESS { // Check soldier privileges
				return errors.New("you do not have enough privileges to perform this action")
			}

		}

	} else {
		return errors.New("token is empty") // Missing token
	}
	return nil // Successful validation
}
