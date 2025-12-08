package execute_mission

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"mission_control/soldier/models"
	"mission_control/soldier/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ExecuteMission runs the mission logic and sends updates to RabbitMQ
func ExecuteMission(m models.Mission, ch *amqp.Channel) {
	status := struct {
		MissionID string `json:"mission_id"`
		Status    string `json:"status"`
	}{MissionID: m.ID, Status: "IN_PROGRESS"}

	body, _ := json.Marshal(status)
	rabbitmq.PublishWithRetry(ch, rabbitmq.StatusQueue, body)

	delay := time.Duration(5+rand.Intn(10)) * time.Second
	time.Sleep(delay)

	outcome := "COMPLETED"
	if rand.Float32() > 0.9 {
		outcome = "FAILED"
	}

	status.Status = outcome
	body, _ = json.Marshal(status)
	rabbitmq.PublishWithRetry(ch, rabbitmq.StatusQueue, body)

	fmt.Printf("Mission %s finished: %s\n", m.ID, outcome)
}
