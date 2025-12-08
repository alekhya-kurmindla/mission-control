package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type Mission struct {
	ID     string `json:"mission_id"`
	Order  string `json:"order"`
	Status string `json:"status"`
	JWT    string `json:"jwt"`
}

const (
	ordersQueue = "orders_queue"
	statusQueue = "status_queue"
)

// Validates incoming JWT token and ensures it has not expired
func validateToken(tokenString string) error {
	if tokenString == "" {
		return errors.New("missing token")
	}

	token, err := extractBearerToken(tokenString)
	fmt.Println(" token: ", token)
	if err != nil {
		return errors.New(fmt.Sprintf("invalid token. error: %v ", err.Error()))
	}

	// Parse JWT using the configured secret
	_, err = jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return errors.New(fmt.Sprintf("invalid or expired token. error: %v ", err.Error()))
	}
	return nil
}

// Extracts token from the Authorization header in the format: Bearer token
func extractBearerToken(authHeader string) (string, error) {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return "", errors.New("empty authorization header")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header must be in the format Bearer token")
	}
	return parts[1], nil
}

// Exits the program when a critical error occurs
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// Publishes messages to RabbitMQ with retry logic
func publishWithRetry(ch *amqp.Channel, queue string, body []byte) error {
	maxAttempts := 5
	wait := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {

		err := ch.Publish(
			"",
			queue,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)

		if err == nil {
			return nil
		}

		log.Printf("Publish failed for queue %s. Attempt %d of %d", queue, attempt, maxAttempts)
		time.Sleep(wait)
		wait = wait * 2
	}

	return errors.New("publish failed after retries")
}

// Executes a mission and sends progress and completion updates back to RabbitMQ
func executeMission(m Mission, ch *amqp.Channel) {
	// Build initial status update
	status := struct {
		MissionID string `json:"mission_id"`
		Status    string `json:"status"`
	}{MissionID: m.ID, Status: "IN_PROGRESS"}

	body, _ := json.Marshal(status)

	// Publish status message with retry logic
	publishWithRetry(ch, statusQueue, body)

	// Simulate work by sleeping for a random number of seconds
	delay := time.Duration(5+rand.Intn(10)) * time.Second
	time.Sleep(delay)

	// Random outcome to simulate real mission success or failure
	outcome := "COMPLETED"
	if rand.Float32() > 0.9 {
		outcome = "FAILED"
	}

	status.Status = outcome
	body, _ = json.Marshal(status)

	// Publish final status update
	publishWithRetry(ch, statusQueue, body)

	fmt.Printf("Mission %s finished: %s\n", m.ID, outcome)
}

func main() {
	fmt.Println("Soldier starting up...")
	rand.Seed(time.Now().UnixNano())

	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/myvhost")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// Open channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()

	// Declare the queues the service depends on
	ch.QueueDeclare(ordersQueue, false, false, false, false, nil)
	ch.QueueDeclare(statusQueue, false, false, false, false, nil)

	// Register consumer to listen for new missions
	msgs, err := ch.Consume(ordersQueue, "", true, false, false, false, nil)
	failOnError(err, "Failed to register consumer")

	fmt.Println("Soldier waiting for missions...")

	// Loop and process missions as they arrive
	for d := range msgs {
		var mission Mission
		json.Unmarshal(d.Body, &mission)

		// Validate JWT before allowing mission execution
		if err := validateToken(mission.JWT); err != nil {
			fmt.Printf("Rejecting mission %s: %s\n", mission.ID, err.Error())
			continue
		}

		fmt.Printf("Valid JWT. Executing mission %s...\n", mission.ID)

		// Execute mission concurrently
		go executeMission(mission, ch)
	}
}
