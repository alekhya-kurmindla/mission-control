package main

import (
	"encoding/json"
	"log"
	"mission_control/soldier/auth"
	"mission_control/soldier/execute_mission"
	"mission_control/soldier/models"
	"mission_control/soldier/rabbitmq"
)

func main() {
	log.Println("Soldier starting up...")

	//Connect to RabbitMQ with retry
	conn, ch := rabbitmq.SetupRabbitWithRetry()
	defer conn.Close()
	defer ch.Close()

	//Auth soldier with retry
	if !auth.GetAuthWithRetry() {
		log.Fatal("Soldier cannot start without authentication")
	}

	//Start consuming messages
	msgs, err := ch.Consume(rabbitmq.OrdersQueue, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}
	log.Println("Soldier waiting for missions...")

	//Process incoming missions
	for d := range msgs {
		var mission models.Mission

		// Validate incoming JSON
		if err := json.Unmarshal(d.Body, &mission); err != nil {
			log.Printf("Invalid mission JSON: %s", err.Error())
			continue // Skip invalid mission
		}

		if mission.ID == "" {
			log.Println("Received mission with empty ID — skipping")
			continue
		}

		log.Printf("Mission received: %s", mission.ID)

		// Execute mission safely
		go func(m models.Mission) {
			// Recover from panics inside mission execution
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in mission %s: %v", m.ID, r)
				}
			}()

			execute_mission.ExecuteMission(auth.Ctx, m, ch)
		}(mission)
	}
	log.Println("RabbitMQ channel closed — soldier stopping")
}
