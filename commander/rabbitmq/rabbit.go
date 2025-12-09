package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"mission_control/commander/models"
	"mission_control/commander/store"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrdersQueue = "orders_queue"
	StatusQueue = "status_queue"
)

func SetupRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	//conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/myvhost")
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/myvhost" // fallback
	}
	conn, err := amqp.Dial(rabbitmqURL)
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	ch.QueueDeclare(OrdersQueue, false, false, false, false, nil)
	ch.QueueDeclare(StatusQueue, false, false, false, false, nil)

	return conn, ch
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// Consumes status updates from the queue and updates mission map
func ConsumeStatusUpdates(ch *amqp.Channel) {
	msgs, err := ch.Consume(StatusQueue, "", true, false, false, false, nil)
	failOnError(err, "Failed to register consumer")

	for d := range msgs {
		var statusUpdate struct {
			MissionID string `json:"mission_id"`
			Status    string `json:"status"`
		}
		json.Unmarshal(d.Body, &statusUpdate)
		log.Printf("DEBUG: consumed data: %v ", statusUpdate)
		store.MissionsMutex.Lock()
		if mission, ok := store.MissionsMap[statusUpdate.MissionID]; ok {
			mission.Status = statusUpdate.Status
		}
		store.MissionsMutex.Unlock()
	}
}

// PublishMission publishes mission to RabbitMQ with retries
func PublishMission(ch *amqp.Channel, mission *models.Mission) error {
	body, _ := json.Marshal(mission)

	maxRetries := 5
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := ch.Publish("", OrdersQueue, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

		if err == nil {
			return nil
		}

		log.Printf("Publish failed. Attempt %d. Retrying...", attempt)
		time.Sleep(backoff)
		backoff *= 2
	}

	return fmt.Errorf("failed to publish mission after retries")
}
