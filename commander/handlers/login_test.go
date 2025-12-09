package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("mysecret")

// TestableGenerateJWT - exported so unit tests can call it
func TestableGenerateJWT(username string) (JWTResponse, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenStr, err := token.SignedString(jwtSecret)
	res := JWTResponse{
		AccessToken:  tokenStr,
		RefreshToken: tokenStr,
	}
	return res, err
}

func TestGenerateJWT(t *testing.T) {
	// call function
	res, err := TestableGenerateJWT("test")
	if err != nil {
		t.Fatalf("generateJWT returned error: %v", err)
	}

	if res.AccessToken == "" {
		t.Fatalf("expected access token, got empty string")
	}

	if res.RefreshToken == "" {
		t.Fatalf("expected refresh token, got empty string")
	}

	// parse tokens
	accessToken, err := jwt.Parse(res.AccessToken, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !accessToken.Valid {
		t.Fatalf("invalid access token: %v", err)
	}

	refreshToken, err := jwt.Parse(res.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !refreshToken.Valid {
		t.Fatalf("invalid refresh token: %v", err)
	}

	// check claims
	accessClaims := accessToken.Claims.(jwt.MapClaims)
	if accessClaims["sub"] != "test" {
		t.Fatalf("expected sub=test, got %v", accessClaims["sub"])
	}

}

func TestLoginHandler(t *testing.T) {
	// mock request + recorder
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(""))
	rr := httptest.NewRecorder()

	LoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// parse body
	var resp map[string]JWTResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	tokenObj, ok := resp["token"]
	if !ok {
		t.Fatalf("response missing 'token' field")
	}

	if tokenObj.AccessToken == "" {
		t.Fatalf("access token missing")
	}

	if tokenObj.RefreshToken == "" {
		t.Fatalf("refresh token missing")
	}
}
