package middleware

import (
	"encoding/json"
	"mission_control/commander/config"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware validates JWT for protected endpoints
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// validating the auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing or invalid Authorization header"))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return config.GetJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid or expired token"))
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

			// Example: read "sub"
			user := claims["user"].(string)
			role := claims["role"].(string)

			if user != config.COMMANDER_USER {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
				"message": "Only commander can perform this action",
				})
				return
			}

			if role != config.COMMANDER_ACCESS { 
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
				"message": "You do not have enough privileges to perform this action",
				})
				return
			}

		}

		next.ServeHTTP(w, r)
	})
}
