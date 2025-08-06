package config

import (
	"os"
	"time"
)

var JWTSecret []byte
var JWTExpiration time.Duration

func init() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-this-in-production"
	}
	JWTSecret = []byte(secret)
	JWTExpiration = 24 * time.Hour
}
