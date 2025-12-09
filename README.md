# Commander--Soldier Messaging System

A distributed command system where a Commander sends mission orders via
RabbitMQ and Soldiers process them and return status updates.
## Welcome to the Command Center.
Your mission, should you choose to accept it, is to help build a secure, resilient communication system designed for modern military operations.
This project implements a one-way, asynchronous command pipeline between:
Commander's Camp – Issues orders
Soldier Units – Execute missions on the battlefield
Central Communication Hub – A secure, internal message broker

### Technology

| Component            | Technology               | Rationale                                                                                                      |
| -------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------- |
| **API Framework**    | Golang (net/http)        | Fast, strongly typed, and ideal for building efficient, reliable backend APIs with minimal runtime overhead.   |
| **Persistence**      | Global In-Memory Map     | Lightweight, zero-dependency storage for tracking mission states within the service instance.                  |
| **Message Queue**    | RabbitMQ                 | Provides reliable message delivery, queue-based communication, and decoupling between API and worker services. |
| **Containerization** | Docker                   | Ensures consistent environments, clean isolation, and simple deployment across machines.                       |
| **Message Format**   | JSON                     | Human-readable, language-agnostic, and easy to encode/decode in Go.                                            |
| **Worker Scaling**   | Docker Compose Replicas  | Offers straightforward horizontal scaling without needing complex orchestration tools like Kubernetes.         |



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
mission_control/
│
├── docker-compose.yaml
│
├── commander/
│   ├── config/config.go
│   ├── middleware/jwt.go     // JWT generation + validation
│   ├── store/store.go        // manages the in-memory mission storage 
│   ├── handlers/login.go     // mission handlers (POST + GET)
|   |        └── missions.go
│   ├── rabbitmq/rabbit.go    // publishing + consuming logic
│   ├── models/mission.go     // Mission struct
|   ├── main.go         
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── soldier/
│   ├── config/config.go
│   ├── auth/jwt.go                // verify JWT before consuming
|   ├── execute_mission/soldier.go
│   ├── rabbitmq/rabbitmq.go       // queues & publisher
│   ├── executor.go                // mission execution logic
│   ├── models/model.go            // Mission struct
|   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
└── README.md
```

#### Run Commander

```
go run commander/main.go
```

#### Run Soldier

```
go run soldier/main.go
```

#### Docker compose
RUN docker
```
docoker-compose up
```

<img width="1573" height="775" alt="image" src="https://github.com/user-attachments/assets/31bf2b08-f7c8-4b0c-9f30-1dbc2ad36b17" />

## API Documentation
<table>
    <tr>
        <td><img width="1421" height="803" alt="image" src="https://github.com/user-attachments/assets/adf1a313-72e7-43c6-bc86-70af23413afb" /></td>
    </tr>
     <tr>
        <td><img width="1278" height="811" alt="image" src="https://github.com/user-attachments/assets/f00ac572-e327-4328-ac26-295f6066ba89" /></td>
    </tr>
     <tr>
        <td><img width="1299" height="706" alt="image" src="https://github.com/user-attachments/assets/70d575db-01d7-40c3-921c-ddf7e7b489ae" /></td>
    </tr>
     <tr>
        <td><img width="1252" height="267" alt="image" src="https://github.com/user-attachments/assets/e046848d-ae9e-452d-8a94-e92f63dea93e" /></td>
    </tr>
</table>





## JWT Authentication

JWT is validated before soldier executes any mission.

## Design Rationale

RabbitMQ chosen for simple command-response behavior. Go concurrency
ensures missions run in parallel. JWT prevents unauthorized orders.


### Mission Status Flow
<table>
    <tr>
        <td><b>QUEUED</b></td>
        <td>Mission received and waiting for processing</td>
    </tr>
    <tr>
        <td><b>IN_PROGRESS</b></td>
        <td>Worker has started executing the mission</td>
    </tr>
    <tr>
        <td><b>COMPLETED</b></td>
        <td>Mission executed successfully</td>
    </tr>
    <tr>
        <td><b>FAILED</b></td>
        <td>Mission execution was unsuccessful</td>
    </tr>
</table>

<img width="1369" height="635" alt="image" src="https://github.com/user-attachments/assets/a4ed9562-e74a-421e-809a-1fa1c0bffa04" />


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
                         │          orders_queue          │
                         └────────────────┬───────────────┘
                                          │
                                          │ 3. Mission picked by Soldier
                                          ▼
                         ┌────────────────────────────────┐
                         │         Soldier Service        │
                         │   (execute_mission.go logic)   │
                         └────────────────┬───────────────┘
                                          │
                                  ┌───────┴─────────────────────────────-──┐
                                  │ 3a. Publish "IN_PROGRESS" to RabbitMQ  │
                                  │     status_queue                       │
                                  └────────────────────────────────────────┘
                                          │
                                          │ 4. Soldier executes mission
                                          │    • sleeps random 5–10 sec  
                                          │    • randomly completes/ fails  
                                          ▼
                                  ┌───────────────────────────────────────-─┐
                                  │ Publish final status (COMPLETED/FAILED) │
                                  │             to status_queue             |
                                  └──────────────────────────────────────-──┘
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

<img width="4009" height="4411" alt="dataflow" src="https://github.com/user-attachments/assets/ee20c293-c43a-401d-a8f8-0ddc12f21482" />

### Container Architecture 

<img width="4850" height="2580" alt="container_architecture" src="https://github.com/user-attachments/assets/399d5f5b-8243-4cca-98b0-b9419f1137d6" />

### Execution logs

#### Container logs
<img width="1613" height="380" alt="image" src="https://github.com/user-attachments/assets/f29ceabd-37b0-4a81-b638-8a44c69e22ff" />

#### Login
<img width="1098" height="730" alt="image" src="https://github.com/user-attachments/assets/a6046a0b-9c7b-4dfb-ac0f-c7821b5f432a" />

#### Post an order
<img width="1082" height="741" alt="image" src="https://github.com/user-attachments/assets/62db84d0-4982-4618-a6dd-b2a94ed85197" />

#### Verify the order status
<img width="1072" height="751" alt="image" src="https://github.com/user-attachments/assets/063bd3d4-2464-402e-8653-46b76f014f1c" />







## AI Usage Policy

AI used only for documenting, debugging, and readability improvements.
