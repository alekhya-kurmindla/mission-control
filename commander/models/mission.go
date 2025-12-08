package models

// Mission represents a command sent to the soldier service
type Mission struct {
	ID     string `json:"mission_id"`
	Order  string `json:"order"`
	Status string `json:"status"`
	JWT    string `json:"jwt"` // Excluded from JSON output
}

// GetMission is used for responses where JWT should not be included
type GetMission struct {
	ID     string `json:"mission_id"`
	Order  string `json:"order"`
	Status string `json:"status"`
	JWT    string `json:"-"`
}
