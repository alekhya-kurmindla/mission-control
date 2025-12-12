package execute_mission

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"mission_control/soldier/auth"
	"mission_control/soldier/models"
	"mission_control/soldier/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ExecuteMission runs the mission logic and sends status updates
func ExecuteMission(ctx context.Context, m models.Mission, ch *amqp.Channel) {

	// Validate soldier token before executing mission
	err := auth.ValidateSoldier(ctx)

	if err != nil {
		// Log authentication failure and mark mission unfinished
		log.Println("Got an error while ValidateSoldier: ", err.Error())
		log.Printf("Mission %s is unfinished due to Authentication error: %s\n", m.ID, err.Error())
	} else {
		// Prepare initial mission IN_PROGRESS status
		status := struct {
			MissionID string `json:"mission_id"`
			Status    string `json:"status"`
		}{MissionID: m.ID, Status: "IN_PROGRESS"}

		// Convert status to JSON
		body, err := json.Marshal(status)

		if err != nil {
			// Log serialization failure
			log.Println("Got an error while Marshal mission status: ", err.Error())
		}

		// Publish IN_PROGRESS status with retry logic
		rabbitmq.PublishWithRetry(ch, rabbitmq.StatusQueue, body)

		// Simulate mission execution time
		delay := time.Duration(1+rand.Intn(5)) * time.Second
		time.Sleep(delay)

		// Randomly determine mission outcome
		outcome := "COMPLETED"
		if rand.Float32() > 0.9 {
			outcome = "FAILED"
		}

		// Update final mission status
		status.Status = outcome
		body, _ = json.Marshal(status)

		// Publish final mission status update
		rabbitmq.PublishWithRetry(ch, rabbitmq.StatusQueue, body)

		// Log the mission result
		log.Printf("Mission %s finished: %s\n", m.ID, outcome)
	}

}
