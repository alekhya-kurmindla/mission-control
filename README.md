# Commander--Soldier Messaging System

A distributed command system where a Commander sends mission orders via
RabbitMQ and Soldiers process them and return status updates.

## Setup Instructions

### Install Go

Download Go from https://go.dev/dl/

### Install RabbitMQ on Windows

1.  Install Erlang OTP
2.  Install RabbitMQ Server
3.  Enable management plugin: rabbitmq-plugins enable
    rabbitmq_management
4.  Open dashboard at http://localhost:15672

## Project Structure
```
├── commander/
│   └── main.go
|   └── Dockerfile
├── soldier/
│   └── main.go
|   └── Dockerfile
├── docker-compose.yaml
└── README.md
└── test_missions.bat
```

### Run Commander

go run commander/main.go

### Run Soldier

go run soldier/main.go

## API Documentation

Commander publishes missions to orders_queue. Soldier returns updates to
status_queue.

## JWT Authentication

JWT is validated before soldier executes any mission.

## Design Rationale

RabbitMQ chosen for simple command-response behavior. Go concurrency
ensures missions run in parallel. JWT prevents unauthorized orders.

## AI Usage Policy

AI used only for documenting, debugging, and readability improvements.
