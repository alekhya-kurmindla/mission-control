package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	missions      = make(map[string]*Mission)
	missionsMutex = sync.RWMutex{}
	statusQueue   = "status_queue"
	ordersQueue   = "orders_queue"
)

// Returns the JWT secret from environment or fallback value
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "MYSECRET" //Should return error.
	}
	return secret
}

// Generates a new JWT token with expiry and subject claims
func generateJWT() (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"sub": "commander-user",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// Middleware to validate JWT token for protected API endpoints
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing or invalid Authorization header"))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return []byte(getJWTSecret()), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid or expired token"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Mission represents a command sent to the soldier service
// JWT is excluded from JSON output for safety
type Mission struct {
	ID     string `json:"mission_id"`
	Order  string `json:"order"`
	Status string `json:"status"`
	JWT    string `json:"jwt"`
}

type GetMission struct {
	ID     string `json:"mission_id"`
	Order  string `json:"order"`
	Status string `json:"status"`
	JWT    string `json:"-"`
}

// Logs a fatal error and message
func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/myvhost")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// Open AMQP channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// Declare required queues
	ch.QueueDeclare(ordersQueue, false, false, false, false, nil)
	ch.QueueDeclare(statusQueue, false, false, false, false, nil)

	// Start a goroutine that listens for status updates
	go consumeStatusUpdates(ch)

	// Public login endpoint
	http.HandleFunc("/login", loginHandler)

	// Protected endpoints
	http.Handle("/missions", jwtMiddleware(http.HandlerFunc(createMissionHandler(ch))))
	http.Handle("/missions/", jwtMiddleware(http.HandlerFunc(getMissionHandler)))

	log.Println("Commander API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handles login request and returns a JWT token
func loginHandler(w http.ResponseWriter, r *http.Request) {
	token, err := generateJWT()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

// Handles mission creation and publishes it to RabbitMQ
func createMissionHandler(ch *amqp.Channel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Order string `json:"order"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		authHeader := r.Header.Get("Authorization")

		mission := &Mission{
			ID:     uuid.New().String(),
			Order:  req.Order,
			Status: "QUEUED",
			JWT:    authHeader,
		}

		// Store mission in in memory map
		missionsMutex.Lock()
		missions[mission.ID] = mission
		missionsMutex.Unlock()

		// Publish mission with retry logic
		if err := publishMission(ch, mission); err != nil {
			http.Error(w, "Failed to publish mission", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"mission_id": mission.ID})
	}
}

// Returns mission by ID
func getMissionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/missions/"):]
	missionsMutex.RLock()
	defer missionsMutex.RUnlock()

	if mission, ok := missions[id]; ok {
		mission.JWT = ""
		json.NewEncoder(w).Encode(mission)
		return
	}
	http.Error(w, "Mission not found", http.StatusNotFound)
}

// Listens to status updates and updates mission state
func consumeStatusUpdates(ch *amqp.Channel) {
	msgs, err := ch.Consume(
		statusQueue,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register consumer")

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

// Publishes mission to RabbitMQ with retry logic and backoff
func publishMission(ch *amqp.Channel, mission *Mission) error {
	body, _ := json.Marshal(mission)

	maxRetries := 5
	backoff := time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {

		err := ch.Publish(
			"",
			ordersQueue,
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

		log.Printf("Publish failed. Attempt %d. Retrying...", attempt)
		time.Sleep(backoff)
		backoff *= 2
	}

	return fmt.Errorf("failed to publish mission after retries")
}
