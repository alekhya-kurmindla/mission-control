package handlers

import (
	"encoding/json"
	"mission_control/commander/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
)

func TestGenerateJWT(t *testing.T) {
	tokenStr, err := generateJWT()
	if err != nil {
		t.Fatalf("generateJWT returned error: %v", err)
	}

	// Parse the token
	token, err := jwt.Parse(tokenStr, func(tkn *jwt.Token) (interface{}, error) {
		return config.GetJWTSecret(), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse generated JWT: %v", err)
	}

	if !token.Valid {
		t.Fatalf("Generated token is invalid")
	}
}

func TestLoginHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	rr := httptest.NewRecorder()

	LoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]string
	err := json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&resp)
	if err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	tokenStr, ok := resp["token"]
	if !ok {
		t.Fatalf("response missing 'token' field")
	}

	token, err := jwt.Parse(tokenStr, func(tkn *jwt.Token) (interface{}, error) {
		return config.GetJWTSecret(), nil
	})

	if err != nil {
		t.Fatalf("token returned by LoginHandler is invalid: %v", err)
	}

	if !token.Valid {
		t.Fatalf("token returned by LoginHandler is not valid")
	}
}
