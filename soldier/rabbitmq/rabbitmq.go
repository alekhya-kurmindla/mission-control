package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"mission_control/soldier/models"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	OrdersQueue = "orders_queue"
	StatusQueue = "status_queue"
)

// FailOnError logs a fatal error if one occurs
func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// SetupRabbitMQ connects to RabbitMQ and declares required queues
func SetupRabbitMQ() (*amqp.Connection, *amqp.Channel) {
	//conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/myvhost")

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/myvhost" // fallback
	}
	conn, err := amqp.Dial(rabbitmqURL)

	
	FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")

	ch.QueueDeclare(OrdersQueue, false, false, false, false, nil)
	ch.QueueDeclare(StatusQueue, false, false, false, false, nil)

	return conn, ch
}

// PublishWithRetry publishes a message to a queue with retries
func PublishWithRetry(ch *amqp.Channel, queue string, body []byte) error {
	maxAttempts := 5
	wait := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := ch.Publish("", queue, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

		if err == nil {
			return nil
		}

		log.Printf("Publish failed for queue %s. Attempt %d/%d", queue, attempt, maxAttempts)
		time.Sleep(wait)
		wait *= 2
	}

	return fmt.Errorf("failed to publish message to queue %s after retries", queue)
}

// ConsumeStatusUpdates consumes status updates and applies them to the mission map
func ConsumeStatusUpdates(ch *amqp.Channel, missions map[string]*models.Mission, missionsMutex *sync.RWMutex) {
	msgs, err := ch.Consume(StatusQueue, "", true, false, false, false, nil)
	FailOnError(err, "Failed to register consumer")

	for d := range msgs {
		var statusUpdate struct {
			MissionID string `json:"mission_id"`
			Status    string `json:"status"`
		}
		json.Unmarshal(d.Body, &statusUpdate)

		missionsMutex.Lock()
		if mission, ok := missions[statusUpdate.MissionID]; ok {
			mission.Status = statusUpdate.Status
		}
		missionsMutex.Unlock()
	}
}
