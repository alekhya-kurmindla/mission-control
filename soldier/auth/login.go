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

func GetAuth(ctx context.Context) error {
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

func GetAuthWithRetry() bool {
	for retries := 1; retries <= 5; retries++ {

		log.Printf("Soldier login attempt %d/5...", retries)
		if err := GetAuth(Ctx); err == nil {
			log.Println("Soldier authenticated successfully")
			return true
		}

		log.Println("Login failed — retrying in 3 seconds")
		time.Sleep(3 * time.Second)
	}

	log.Println("Soldier login failed after 5 attempts")
	return false
}
