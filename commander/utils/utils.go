package utils

import (
	"encoding/json"
	"net/http"
)

// RenderJSONMessage writes JSON response with status code
func RenderJsonMessage(data any, w http.ResponseWriter, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	json.NewEncoder(w).Encode(data)
}