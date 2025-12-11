package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mission_control/commander/config"
	"mission_control/commander/middleware"

	jwt "github.com/golang-jwt/jwt/v5"
)


func generateValidToken() string {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"user": config.COMMANDER_USER,
		"role": config.COMMANDER_ACCESS,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, _ := token.SignedString(config.GetJWTSecret())
	return str
}

func generateInvalidToken() string {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"sub": "test-user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, _ := token.SignedString([]byte("WRONG_SECRET_KEY"))
	return str
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	validToken := generateValidToken()

	// Handler that should run when token is valid
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	rr := httptest.NewRecorder()
	handler := middleware.JWTMiddleware(nextHandler)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	if rr.Body.String() == "" {
		t.Fatalf("expected response body, got empty")
	}
}

func TestJWTMiddleware_MissingToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/protected", nil)
	rr := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := middleware.JWTMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}

	expected := "Missing or invalid Authorization header"
	if rr.Body.String() != expected {
		t.Fatalf("expected %q, got %q", expected, rr.Body.String())
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	invalidToken := generateInvalidToken()

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)

	rr := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := middleware.JWTMiddleware(next)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}

	expected := "Invalid or expired token"
	if rr.Body.String() != expected {
		t.Fatalf("expected %q, got %q", expected, rr.Body.String())
	}
}
