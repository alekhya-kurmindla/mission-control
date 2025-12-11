package models

// Mission represents a command sent to the soldier service
type Mission struct {
	ID     string `json:"mission_id"`
	Order  string `json:"order"`
	Status string `json:"status"`
}

type Token struct {
	AccessToken    string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginResponse struct {
	Token Token `json:"token"`
}

