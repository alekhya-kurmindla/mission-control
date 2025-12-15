package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"mission_control/soldier/config"
	"net/http"
	"os"
	"time"
)

var (
	Ctx = context.Background() // Global context
)

// GetAuth performs a single authentication request to the Commander API
func GetAuth(ctx context.Context) error {
	commanderURL := os.Getenv("COMMANDER_URL")

	// Prepare login credentials
	creds := map[string]string{
		"user":    config.SOLDIER_USER,          // Soldier username
		"api_key": os.Getenv("SOLDIER_API_KEY"), // Soldier API key from environment
	}

	body, _ := json.Marshal(creds) // Convert credentials to JSON

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

// GetAuthWithRetry attempts soldier authentication with retries and sets token expiry on success
func GetAuthWithRetry() bool {
	for retries := 1; retries <= 5; retries++ {

		log.Printf("Soldier login attempt %d/5...", retries)
		if err := GetAuth(Ctx); err == nil {
			log.Println("Soldier authenticated successfully") // Success
			return true
		}

		log.Println("Login failed — retrying in 3 seconds") // Retry message
		time.Sleep(3 * time.Second) //backoff
	}

	log.Println("Soldier login failed after 5 attempts") // Final failure after retries
	return false
}
