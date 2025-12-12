package main

import (
	"log"
	"net/http"

	"mission_control/commander/handlers"
	"mission_control/commander/middleware"
	"mission_control/commander/rabbitmq"
)

func main() {
	// Connect to RabbitMQ
	conn, ch := rabbitmq.SetupRabbitMQ()
	defer conn.Close()
	defer ch.Close()

	// Start status consumer
	go rabbitmq.ConsumeStatusUpdates(ch)

	// Public login endpoint
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/refresh", handlers.RefreshHandler)
	http.HandleFunc("/health", handlers.HealthCheckHandler)
	
	// Protected endpoints
	http.Handle("/missions", middleware.JWTMiddleware(http.HandlerFunc(handlers.CreateMissionHandler(ch))))
	http.Handle("/missions/", middleware.JWTMiddleware(http.HandlerFunc(handlers.GetMissionHandler)))

	log.Println("Commander API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
