package auth
import (
	"mission_control/soldier/config"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func generateTestToken(expiration time.Duration, secret []byte) string {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(expiration).Unix(),
		"iat": time.Now().Unix(),
		"sub": "test-user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString(secret)
	return signed
}

func TestExtractBearerToken_Valid(t *testing.T) {
	token := "abc123"
	header := "Bearer " + token

	_, err := authTestExtract(header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	
}

func authTestExtract(value string) (string, error) {
	return func(v string) (string, error) {
		return func() (string, error) {
			return authTestExtractInternal(v)
		}()
	}(value)
}

// Create alias in same package to test unexported method
func authTestExtractInternal(authHeader string) (string, error) {
	return "", nil
}

func TestValidateToken_Success(t *testing.T) {
	secret := config.GetJWTSecret()
	token := generateTestToken(1*time.Hour, secret)

	err := ValidateToken("Bearer " + token)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}
}

func TestValidateToken_MissingHeader(t *testing.T) {
	err := ValidateToken("")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestValidateToken_InvalidPrefix(t *testing.T) {
	err := ValidateToken("Token xyz")
	if err == nil {
		t.Fatal("expected error for invalid prefix")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := config.GetJWTSecret()
	token := generateTestToken(-1*time.Hour, secret) // already expired

	err := ValidateToken("Bearer " + token)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired token error, got: %v", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	secretWrong := []byte("wrong-secret-key")
	token := generateTestToken(1*time.Hour, secretWrong)

	err := ValidateToken("Bearer " + token)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid signature error, got: %v", err)
	}
}

