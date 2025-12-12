# Commander--Soldier Messaging System

A distributed Commander–Soldier mission execution system built in Golang with RabbitMQ for message queuing.
The Commander service submits missions, and Soldier workers pick them up concurrently, process them, and push back live status updates.

## Architecture Overview
### Commander

Accepts mission submissions via API

Sends missions to orders_queue

Polls status_queue to track mission progress

Logs mission lifecycle (QUEUED → IN_PROGRESS → COMPLETED/FAILED)

### Soldier

Continuously listens on orders_queue

Processes missions (simulated work + randomized failures)

Sends back status updates to status_queue

Handles concurrency safely using channels and goroutines

## Message Queues

orders_queue → Commander → Soldier (mission orders)

status_queue → Soldier → Commander (mission progress/status)

## Commander Service 
The Commander service acts as the central controller of the system. It accepts incoming mission creation requests through HTTP and generates a unique mission_id for each mission. These missions are then published to the RabbitMQ orders_queue, where they are consumed by Soldier services. At the same time, the Commander listens for mission status updates coming from the status_queue, processes them, and updates each mission’s status in an in-memory store secured with a mutex to ensure thread-safe access. Additionally, the Commander exposes HTTP endpoints that allow clients to fetch the current status of any mission.

```Commander → Soldier ```
Publishes mission orders to orders_queue.

## Soldier Service
The Soldier service acts as the executor of missions received from the Commander. It continuously listens to the RabbitMQ orders_queue for new mission instructions. Upon receiving a mission, the Soldier authenticates itself with the Commander service, processes the mission, and simulates execution by introducing realistic delays. During execution, it sends status updates—such as IN_PROGRESS, COMPLETED, or FAILED—back to the Commander through the status_queue. The Soldier uses retry mechanisms to ensure reliable message delivery and maintains secure communication using JWT authentication.

Publishes mission status updates to status_queue.
Publishes mission progress (IN_PROGRESS, COMPLETED, FAILED) to status_queue.

## Concurrency & Safety Procedures

The Commander–Soldier system is built to safely handle multiple missions running in parallel while avoiding race conditions, deadlocks, and message-processing failures. The following safeguards are implemented throughout the codebase:

#### 1. Thread-Safe Mission Store (Commander)

Mission statuses are stored in an in-memory map protected by a sync.RWMutex.
Every read/write (GetMissionHandler, SaveMissionStatus) is fully synchronized.
Prevents concurrent updates from corrupting mission state.

#### 2. Safe Parallel Mission Execution (Soldier)

Each mission pulled from orders_queue is executed inside a separate goroutine.
A defer recover() is included to prevent a panic in one mission from crashing the Soldier service.

#### 3. Controlled Message Flow (Commander)

Status message consumption uses:

ch.Qos(1, 0, false)

This ensures the Commander processes only one unacknowledged status update at a time, preventing overload and ensuring stable state updates.

#### 4. Retry with Exponential Backoff

Both Commander and Soldier include:
RabbitMQ publish retry logic
Connection retry loops (SetupRabbitWithRetry)
Ensures services remain stable during queue outages or network issues.

#### 5. Thread-Safe JWT Token Lifecycle (Soldier)

AuthToken and RefreshToken are stored in a struct protected by sync.RWMutex.
Prevents race conditions when:
goroutines validate tokens
token refresh happens mid-execution
Expired tokens trigger an automatic refresh before mission execution.

#### 6. Structured Error Handling & Logging

All mission execution, authentication, and messaging logic includes explicit error paths and log outputs.
Failures never block other missions or consumers.

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

### Mission Status Flow

Status	Description

QUEUED	Mission received, waiting for processing

IN_PROGRESS	Soldier started execution

COMPLETED	Mission executed successfully

FAILED	Mission execution failed

### Design diagram

<img width="670" height="531" alt="design_diagram drawio" src="https://github.com/user-attachments/assets/8343548e-cfd3-4149-a101-14797d9b44c0" />

## JWT Authentication

JWT-based access tokens for Soldiers.

Short-lived access tokens + long-lived refresh tokens.

Tokens stored in a thread-safe struct with sync.RWMutex.

Soldiers auto-refresh expired tokens before mission execution.

## Mission Control – Flow Diagram



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

## Overview of the Unit Testing Strategy

The Mission Control project includes a comprehensive suite of unit tests that validate the core functionality of both the Commander and Soldier services. These tests cover mission creation, mission retrieval, in-memory state management, and JWT-based authentication. By mocking external dependencies such as RabbitMQ channels, the test suite verifies message publishing, status propagation, and error handling without requiring the actual broker to be running. This ensures that each component behaves correctly in isolation and adheres to expected API contracts.

<img width="1183" height="482" alt="image" src="https://github.com/user-attachments/assets/0f1a13d4-3d1f-479d-bbd1-b58b605ccd50" />


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
<table>
    <tr>
        <td><img width="1573" height="775" alt="image" src="https://github.com/user-attachments/assets/31bf2b08-f7c8-4b0c-9f30-1dbc2ad36b17" /></td>
    </tr>
     <tr>
        <td><img width="1369" height="635" alt="image" src="https://github.com/user-attachments/assets/a4ed9562-e74a-421e-809a-1fa1c0bffa04" />  </td>
    </tr>
     <tr>
        <td><img width="1530" height="861" alt="image" src="https://github.com/user-attachments/assets/19ff486a-62f8-4173-8c53-91166831c4f0" /></td>
    </tr>
</table>

### Execution logs
<img width="1034" height="464" alt="image" src="https://github.com/user-attachments/assets/99e796ea-f075-42f3-b3ce-eeafdaaa8b0c" />

#### Container logs
<img width="714" height="377" alt="image" src="https://github.com/user-attachments/assets/93ffc817-84bd-4255-8663-7315d6502a38" />


#### Login
<img width="1054" height="830" alt="image" src="https://github.com/user-attachments/assets/0b0fc10b-fff6-4c14-8472-cd3f397f85dc" />


#### Post an order
<img width="1070" height="662" alt="image" src="https://github.com/user-attachments/assets/da75dbf1-eca8-45a1-b521-859b689a5634" />


#### Verify the order status
<img width="1072" height="751" alt="image" src="https://github.com/user-attachments/assets/063bd3d4-2464-402e-8653-46b76f014f1c" />

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

### Technology

| Component            | Technology               | Rationale                                                                                                      |
| -------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------- |
| **API Framework**    | Golang (net/http)        | Fast, strongly typed, and ideal for building efficient, reliable backend APIs with minimal runtime overhead.   |
| **Persistence**      | Global In-Memory Map     | Lightweight, zero-dependency storage for tracking mission states within the service instance.                  |
| **Message Queue**    | RabbitMQ                 | Provides reliable message delivery, queue-based communication, and decoupling between API and worker services. |
| **Containerization** | Docker                   | Ensures consistent environments, clean isolation, and simple deployment across machines.                       |
| **Message Format**   | JSON                     | Human-readable, language-agnostic, and easy to encode/decode in Go.                                            |
| **Worker Scaling**   | Docker Compose Replicas  | Offers straightforward horizontal scaling without needing complex orchestration tools like Kubernetes.         |


## Design Rationale

The architecture adopts RabbitMQ as the core message broker to ensure reliable, decoupled communication between the Commander service and multiple Soldier workers. RabbitMQ was selected for its durability guarantees, built-in acknowledgment model, routing flexibility, and strong support for distributed worker patterns. This allows mission commands to be processed asynchronously, enables horizontal scaling of Soldiers, and ensures no mission is lost even during service restarts. The system’s concurrency model leverages Go’s goroutines and channel-driven worker logic, providing lightweight parallel processing and predictable performance under load, making it well-suited for high-throughput, event-driven workloads.

Authentication is handled using JWT-based access tokens paired with long-lived refresh tokens to balance security with usability. Short-lived access tokens minimize risk exposure, while refresh tokens allow clients to re-authenticate without storing credentials or repeatedly logging in. This stateless authentication model reduces server-side complexity and integrates cleanly with the Commander’s API gateway responsibilities. Together, these choices create a scalable, fault-tolerant, and secure system optimized for real-time mission dispatching and status tracking.

## AI Usage Policy

AI is used solely for documentation, debugging assistance, and improving code readability.
