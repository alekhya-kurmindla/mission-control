package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"mission_control/soldier/auth"
	"mission_control/soldier/execute_mission"
	"mission_control/soldier/models"
	"mission_control/soldier/rabbitmq"
)

var (
	missions      = make(map[string]*models.Mission)
	missionsMutex = sync.RWMutex{}
)

func main() {
	fmt.Println("Soldier starting up...")
	rand.Seed(time.Now().UnixNano())

	conn, ch := rabbitmq.SetupRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	go rabbitmq.ConsumeStatusUpdates(ch, missions, &missionsMutex)

	msgs, err := ch.Consume(rabbitmq.OrdersQueue, "", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "Failed to register consumer")

	fmt.Println("Soldier waiting for missions...")

	for d := range msgs {
		var mission models.Mission
		json.Unmarshal(d.Body, &mission)

		if err := auth.ValidateToken(mission.JWT); err != nil {
			fmt.Printf("Rejecting mission %s: %s\n", mission.ID, err.Error())
			continue
		}

		fmt.Printf("Valid JWT. Executing mission %s...\n", mission.ID)
		go execute_mission.ExecuteMission(mission, ch)
	}
}

