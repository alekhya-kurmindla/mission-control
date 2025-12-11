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
	AuthToken    string
	RefreshToken string
	mu           sync.RWMutex
}

var tokens = &TokenStore{}

func SetTokens(auth, refresh string) {
	tokens.mu.Lock()
	defer tokens.mu.Unlock()

	tokens.AuthToken = auth
	tokens.RefreshToken = refresh
}

func GetTokens() (string, string) {
	tokens.mu.RLock()
	defer tokens.mu.RUnlock()

	return tokens.AuthToken, tokens.RefreshToken
}

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

func isTokenExpired(accessToken string) (bool, error) {
	if accessToken == "" {
		log.Println("token is empty")
		return true, errors.New("token is empty")
	}
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		log.Println("Error parsing token:", err)
		return true, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("invalid token claims")
		return true, errors.New("invalid token claims")
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		log.Println("no exp claim found")
		return true, errors.New("no exp claim found")
	}

	expTime := time.Unix(int64(expFloat), 0)
	now := time.Now()

	if now.After(expTime) {
		log.Println("Token is expired")
		return true, nil
	} else {
		log.Println("Token is valid, expires at:", expTime)
		return false, nil
	}
}

func Login(ctx context.Context) error {

	commanderURL := os.Getenv("COMMANDER_URL")

	// Prepare login request
	creds := map[string]string{
		"user":    config.SOLDIER_USER,
		"api_key": os.Getenv("SOLDIER_API_KEY"),
	}

	body, _ := json.Marshal(creds)

	req, err := http.NewRequest("POST", commanderURL+"/login", bytes.NewBuffer(body))

	if err != nil {
		log.Println("Got an error while making new Login Request: ", err.Error())
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	err = makeAuthCall(req)
	if err != nil {
		log.Println("Got an error in makeAuthCall: ", err.Error())
		return err
	}
	return nil
}

func RefreshToken(ctx context.Context, refreshToken string) error {

	commanderURL := os.Getenv("COMMANDER_URL")

	// Prepare login request
	creds := map[string]string{
		"refresh_token": refreshToken,
	}

	body, _ := json.Marshal(creds)

	req, _ := http.NewRequest("POST", commanderURL+"/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	err := makeAuthCall(req)
	if err != nil {
		log.Println("Got an error: ", err.Error())
		return err
	}
	return nil
}

func makeAuthCall(req *http.Request) error {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Got an error: ", err.Error())
		return err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	// Parse token response
	var loginResp models.LoginResponse
	json.Unmarshal(respBytes, &loginResp)
	
	//set to global variable as ctx value is missing while refreshing the token
	SetTokens(loginResp.Token.AccessToken, loginResp.Token.RefreshToken)
	return nil
}

func ValidateSoldier(ctx context.Context) error {
	authToken, refreshToken := GetTokens()
	isExpired, err := isTokenExpired(authToken)
	if err != nil {
		log.Println("Got an error in ExecuteMission isTokenExpired: ", err.Error())
		return err
	}

	if isExpired {

		log.Println("Soldier token has been expired. Generating new token using refresh token")

		err = RefreshToken(ctx, refreshToken)
		if err != nil {
			log.Println("Got an error in ExecuteMission RefreshToken: ", err.Error())
			return err
		}

		authToken, _ = GetTokens()
	}

	if authToken != "" {

		token, err := jwt.Parse(authToken, func(t *jwt.Token) (interface{}, error) {
			return config.GetJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			return errors.New("invalid or expired token")
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			// Example: read "sub"
			user := claims["user"].(string)
			role := claims["role"].(string)

			if user != config.SOLDIER_USER {
				return errors.New("only soldier can perform this action")
			}

			if role != config.SOLDIER_ACCESS {
				return errors.New("you do not have enough privileges to perform this action")
			}

		}

	} else {
		return errors.New("token is empty")
	}

	return nil
}
