package auth

import (
	"context"
	"log"
	"time"
)

// Rotate token for every 30 seconds
func RotateToken(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("RotateToken job scheduler started")

	for {
		select {
		case <-ctx.Done():
			log.Println("RotateToken job stopped")
			return
		case t := <-ticker.C:
			log.Printf("RotateToken job executed at %v", t)

			_, refreshToken := GetTokens()
			RefreshToken(ctx, refreshToken)
		}
	}
}

