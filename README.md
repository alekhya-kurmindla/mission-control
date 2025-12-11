# Commander--Soldier Messaging System

A distributed command system where a Commander sends mission orders via
RabbitMQ and Soldiers process them and return status updates.
## Welcome to the Command Center.
Your mission, should you choose to accept it, is to help build a secure, resilient communication system designed for modern military operations.
This project implements a one-way, asynchronous command pipeline between:
Commander's Camp – Issues orders
Soldier Units – Execute missions on the battlefield
Central Communication Hub – A secure, internal message broker

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
### Design diagram

<img width="670" height="531" alt="design_diagram drawio" src="https://github.com/user-attachments/assets/8343548e-cfd3-4149-a101-14797d9b44c0" />


### Technology

| Component            | Technology               | Rationale                                                                                                      |
| -------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------- |
| **API Framework**    | Golang (net/http)        | Fast, strongly typed, and ideal for building efficient, reliable backend APIs with minimal runtime overhead.   |
| **Persistence**      | Global In-Memory Map     | Lightweight, zero-dependency storage for tracking mission states within the service instance.                  |
| **Message Queue**    | RabbitMQ                 | Provides reliable message delivery, queue-based communication, and decoupling between API and worker services. |
| **Containerization** | Docker                   | Ensures consistent environments, clean isolation, and simple deployment across machines.                       |
| **Message Format**   | JSON                     | Human-readable, language-agnostic, and easy to encode/decode in Go.                                            |
| **Worker Scaling**   | Docker Compose Replicas  | Offers straightforward horizontal scaling without needing complex orchestration tools like Kubernetes.         |



## JWT Authentication

JWT is validated before soldier executes any mission.

## Design Rationale

The architecture adopts RabbitMQ as the core message broker to ensure reliable, decoupled communication between the Commander service and multiple Soldier workers. RabbitMQ was selected for its durability guarantees, built-in acknowledgment model, routing flexibility, and strong support for distributed worker patterns. This allows mission commands to be processed asynchronously, enables horizontal scaling of Soldiers, and ensures no mission is lost even during service restarts. The system’s concurrency model leverages Go’s goroutines and channel-driven worker logic, providing lightweight parallel processing and predictable performance under load, making it well-suited for high-throughput, event-driven workloads.

Authentication is handled using JWT-based access tokens paired with long-lived refresh tokens to balance security with usability. Short-lived access tokens minimize risk exposure, while refresh tokens allow clients to re-authenticate without storing credentials or repeatedly logging in. This stateless authentication model reduces server-side complexity and integrates cleanly with the Commander’s API gateway responsibilities. Together, these choices create a scalable, fault-tolerant, and secure system optimized for real-time mission dispatching and status tracking.

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

## AI Usage Policy

AI is used solely for documentation, debugging assistance, and improving code readability.
