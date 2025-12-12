package rabbitmq

import (
	"fmt"
	"log"
	"os"
	"time"


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

	ch.QueueDeclare(OrdersQueue, true, false, false, false, nil)
	ch.QueueDeclare(StatusQueue, true, false, false, false, nil)

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

func SetupRabbitWithRetry() (*amqp.Connection, *amqp.Channel) {
	for {
		conn, ch := SetupRabbitMQ()
		if conn != nil && ch != nil {
			log.Println("Connected to RabbitMQ")
			return conn, ch
		}

		log.Println("RabbitMQ connection failed — retrying in 5s...")
		time.Sleep(5 * time.Second)
	}
}

