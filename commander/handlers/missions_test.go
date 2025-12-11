package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mission_control/commander/models"
	"mission_control/commander/store"
)

func TestGetMissionHandler_Success(t *testing.T) {
	// Insert mission into global map
	mission := &models.Mission{
		MissionID:     "abc123",
		Order:  "Recon",
		Status: "QUEUED",
	}

	store.MissionsMutex.Lock()
	store.MissionsMap["abc123"] = mission
	store.MissionsMutex.Unlock()

	// Create request
	req := httptest.NewRequest("GET", "/missions/abc123", nil)
	rr := httptest.NewRecorder()

	// Call handler
	GetMissionHandler(rr, req)

	// Expectations
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rr.Code)
	}

	var resp models.Mission
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.MissionID != "abc123" {
		t.Fatalf("expected ID abc123, got %s", resp.MissionID)
	}

	if resp.Order != "Recon" {
		t.Fatalf("expected order Recon, got %s", resp.Order)
	}
}

func TestGetMissionHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/missions/unknown", nil)
	rr := httptest.NewRecorder()

	GetMissionHandler(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	expected := "Mission not found\n"
	if rr.Body.String() != expected {
		t.Fatalf("expected body %q, got %q", expected, rr.Body.String())
	}
}
