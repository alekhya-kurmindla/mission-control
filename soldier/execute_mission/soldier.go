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

// ExecuteMission runs the mission logic and sends updates to RabbitMQ
func ExecuteMission(ctx context.Context, m models.Mission, ch *amqp.Channel) {

	//before updating status, authorize soldier
	err := auth.ValidateSoldier(ctx)

	if err != nil {
		log.Println("Got an error while ValidateSoldier: ", err.Error())
		log.Printf("Mission %s is unfinished due to Authentication error: %s\n", m.ID, err.Error())
	} else {
		status := struct {
			MissionID string `json:"mission_id"`
			Status    string `json:"status"`
		}{MissionID: m.ID, Status: "IN_PROGRESS"}

		body, err := json.Marshal(status)

		if err != nil {
			log.Println("Got an error while Marshal mission status: ", err.Error())
		}

		rabbitmq.PublishWithRetry(ch, rabbitmq.StatusQueue, body)

		delay := time.Duration(1+rand.Intn(5)) * time.Second
		time.Sleep(delay)

		outcome := "COMPLETED"
		if rand.Float32() > 0.9 {
			outcome = "FAILED"
		}

		status.Status = outcome
		body, _ = json.Marshal(status)
		rabbitmq.PublishWithRetry(ch, rabbitmq.StatusQueue, body)

		log.Printf("Mission %s finished: %s\n", m.ID, outcome)
	}

}
