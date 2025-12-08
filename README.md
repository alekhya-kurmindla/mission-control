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

## Mission Control – Flow Diagram

                           ┌─────────────────────────┐
                           │     Mission Control     │
                           │      (HTTP API)         │
                           └──────────┬──────────────┘
                                      │
                                      │ 1. Receive Mission Request
                                      ▼
                           ┌─────────────────────────┐
                           │   Generate Mission ID   │
                           │   Store in MissionsMap  │
                           └──────────┬──────────────┘
                                      │
                                      │ 2. Publish Mission Order
                                      ▼
                         ┌────────────────────────────────┐
                         │         RabbitMQ Queue         │
                         │          orders_queue           │
                         └────────────────┬────────────────┘
                                          │
                                          │ 3. Mission picked by Soldier
                                          ▼
                         ┌────────────────────────────────┐
                         │         Soldier Service        │
                         │   (execute_mission.go logic)   │
                         └────────────────┬────────────────┘
                                          │
                                  ┌───────┴───────────────────────────────┐
                                  │ 3a. Publish "IN_PROGRESS" to RabbitMQ  │
                                  │     status_queue                        │
                                  └─────────────────────────────────────────┘
                                          │
                                          │ 4. Soldier executes mission
                                          │    • sleeps random 5–10 sec  
                                          │    • randomly completes/ fails  
                                          ▼
                                  ┌────────────────────────────────────────┐
                                  │ Publish final status (COMPLETED/FAILED)│
                                  │             to status_queue             │
                                  └────────────────────────────────────────┘
                                          │
                                          │ 5. Mission Control subscribes
                                          ▼
                       ┌──────────────────────────────────────────────────┐
                       │      Status Consumer (Mission Control Side)      │
                       └──────────────────┬───────────────────────────────┘
                                          │
                                      6. Update MissionsMap
                                          │
                                          ▼
                           ┌─────────────────────────┐
                           │  GetMissionHandler API  │
                           │  /missions/{id}         │
                           └──────────┬──────────────┘
                                      │
                             7. Client Requests Status
                                      │
                                      ▼
                           ┌─────────────────────────┐
                           │  Return Mission Status  │
                           │ (IN_PROGRESS/FAILED/OK) │
                           └─────────────────────────┘


## 1. Commander's Camp Service

```
┌─────────────────────────────────────────────┐
│           COMMANDER'S CAMP SERVICE          │
├─────────────────────────────────────────────┤
│  API Layer:                                 │
│  • REST API (Port: 8080)                    │
│    - POST /missions                         │
│    - GET /missions/{id}                     │
│    - POST /login (for workers)              │
│                                             │
│  Business Logic:                            │
│  • Mission Manager                          │
│  • Status Tracker                           │
│  • Auth Token Issuer                        │
│                                             │
│  Data Layer:                                │
│  • In-Memory Store /Map                     │
│    - Mission Status Cache                   │
│    - Token Registry                         │
└─────────────────────────────────────────────┘
```

### 2. Central Communications Hub (Message Queue)

```
┌─────────────────────────────────────────────┐
│         CENTRAL COMMUNICATIONS HUB          │
├─────────────────────────────────────────────┤
│  Message Queues:                            │
│  • orders_queue (FANOUT)                    │
│    - New mission orders                     │
│    - Persisted for reliability              │
│                                             │
│  • status_queue (PUB/SUB)                   │
│    - Status updates from soldiers           │
│    - Real-time updates                      │
│                                             │                                       │
│  Security:                                  │
│  • TLS/SSL enabled                          │
│  • Authentication required                  │
└─────────────────────────────────────────────┘

```
### 3. 3. Soldier Worker Service

```
┌─────────────────────────────────────────────┐
│              SOLDIER WORKER                 │
├─────────────────────────────────────────────┤
│  Message Consumer:                          │
│  • Polls orders_queue                       │
│  • Graceful failure handling                │
│  • Connection retry logic                   │
│                                             │
│  Mission Executor:                          │
│  • Thread Pool (configurable)               │
│  • Mission simulation                       │
│  • Random delay (5-15s)                     │
│  • Success rate (90%)                       │
│                                             │
│  Status Reporter:                           │
│  • Publishes to status_queue                |
│                                             │
│  Auth Manager:                              │
│  • Token management                         │
│  • Secure token storage                     │
└─────────────────────────────────────────────┘

```

### Data Flow Sequence
```

```

### Container Architecture 
```

```

## AI Usage Policy

AI used only for documenting, debugging, and readability improvements.
