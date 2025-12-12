package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"mission_control/commander/models"
	"mission_control/commander/rabbitmq"
	"mission_control/commander/store"
	"mission_control/commander/utils"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// CreateMissionHandler creates a new mission and publishes it to RabbitMQ
func CreateMissionHandler(ch *amqp.Channel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Order string `json:"order"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			data := map[string]string{
				"message": "Invalid request",
			}
			utils.RenderJsonMessage(data, w, http.StatusBadRequest)
			return
		}
		if req.Order == "" {
			data := map[string]string{
				"message": "Require order",
			}
			utils.RenderJsonMessage(data, w, http.StatusBadRequest)
			return
		}
		mission := &models.Mission{
			MissionID: uuid.New().String(),
			Order:     req.Order,
			Status:    "QUEUED",
		}
		if err := rabbitmq.PublishMission(ch, mission); err != nil {
			data := map[string]string{
				"message": "Failed to publish mission",
			}
			utils.RenderJsonMessage(data, w, http.StatusInternalServerError)
			return
		} else {
			rabbitmq.SaveMissionStatus(mission.MissionID, mission.Status)
			log.Printf("Success: Mission has been published. mission_id: %v ", mission.MissionID)
		}
		data := map[string]string{"mission_id": mission.MissionID, "status": "QUEUED"}
		utils.RenderJsonMessage(data, w, http.StatusAccepted)
	}
}

// GetMissionHandler returns mission status by ID
func GetMissionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/missions/"):]
	store.MissionsMutex.RLock()
	defer store.MissionsMutex.RUnlock()

	if mission, ok := store.MissionsMap[id]; ok {
		json.NewEncoder(w).Encode(mission)
		return
	}
	data := map[string]string{
		"message": "Mission not found",
	}
	utils.RenderJsonMessage(data, w, http.StatusNotFound)
}

// App health check
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":  "ok",
		"service": "commander",
		"message": "Service is healthy",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
