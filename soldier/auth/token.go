package auth

import (
	"context"
	"log"
	"sync"
	"time"
)

var (
	// Ctx        context.Context
	cancel   context.CancelFunc
	tokenMu  sync.RWMutex
	tokenExp time.Time
)

// Call this once during startup
func StartTokenManager() {
	Ctx, cancel = context.WithCancel(context.Background())
	go func() {
		for {
			time.Sleep(25 * time.Second) // refresh BEFORE expiry

			if time.Until(tokenExp) <= 5*time.Second {
				log.Println("Auth token expiring — rotating token")

				if !GetAuthWithRetry() {
					log.Println("--->Token refresh failed — retrying")
					continue
				}

				log.Println("Auth token rotated successfully")
			}
		}
	}()
}

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

