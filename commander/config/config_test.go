package config

import (
	"os"
	"testing"
)

func TestGetJWTSecret_WhenEnvSet(t *testing.T) {
	// Arrange
	expected := "mycustomsecret"
	os.Setenv("JWT_SECRET", expected)

	// Act
	secretBytes := GetJWTSecret()
	secret := string(secretBytes)

	// Assert
	if secret != expected {
		t.Errorf("expected %s, got %s", expected, secret)
	}

	// Cleanup
	os.Unsetenv("JWT_SECRET")
}

func TestGetJWTSecret_WhenEnvNotSet(t *testing.T) {
	// Arrange
	os.Unsetenv("JWT_SECRET")
	expected := "supersecretkey123" // fallback value

	// Act
	secretBytes := GetJWTSecret()
	secret := string(secretBytes)

	// Assert
	if secret != expected {
		t.Errorf("expected fallback %s, got %s", expected, secret)
	}
}
