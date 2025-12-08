package store

import "sync"
import "mission_control/commander/models"

// Missions map stores all missions in memory
var MissionsMap = make(map[string]*models.Mission)

// MissionsMutex protects access to the Missions map
var MissionsMutex = sync.RWMutex{}
