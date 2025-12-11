package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mission_control/soldier/auth"
	"mission_control/soldier/execute_mission"
	"mission_control/soldier/models"
	"mission_control/soldier/rabbitmq"
)

var (
	ctx           = context.Background()
)

func main() {
	log.Println("Soldier starting up...")
	conn, ch := rabbitmq.SetupRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	msgs, err := ch.Consume(rabbitmq.OrdersQueue, "", true, false, false, false, nil)
	rabbitmq.FailOnError(err, "Failed to register consumer")

	log.Println("Soldier waiting for missions...")
	err = auth.Login(ctx)

	if err != nil {
		msg := fmt.Sprintf("ERROR Login failed for Soldier %s...\n", err.Error())
		log.Println(msg)
		panic(msg) //do not start soldier
	}
	for d := range msgs {
		var mission models.Mission
		json.Unmarshal(d.Body, &mission)
		log.Printf("Valid JWT. Executing mission %s...\n", mission.ID)
		go execute_mission.ExecuteMission(ctx, mission, ch)
	}
}
