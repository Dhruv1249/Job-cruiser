package utils

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

/*
SessionTokenDuration defines the extended session lifespan of four months (120 days).
*/
const SessionTokenDuration = 120 * 24 * time.Hour

/*
GenerateToken creates a signed JSON Web Token for the specified user identifier with a four-month lifespan.
*/
func GenerateToken(userID string) (string, error) {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		log.Fatalf("CRITICAL ERROR: JWT_SECRET environment variable is missing.")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(SessionTokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
