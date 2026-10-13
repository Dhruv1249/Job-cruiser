package utils_test

import (
	"os"
	"testing"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/golang-jwt/jwt/v5"
)

/*
TestSessionTokenDurationLifespan verifies that the session token duration corresponds to a four-month interval.
*/
func TestSessionTokenDurationLifespan(testingContext *testing.T) {
	expectedDuration := 120 * 24 * time.Hour
	if utils.SessionTokenDuration != expectedDuration {
		testingContext.Fatalf("expected session token duration %v, got %v", expectedDuration, utils.SessionTokenDuration)
	}
}

/*
TestGenerateTokenSuccess verifies that GenerateToken produces a valid signed JWT with expected claims and expiration.
*/
func TestGenerateTokenSuccess(testingContext *testing.T) {
	testSecretKey := "unit-test-jwt-secret-key-12345"
	os.Setenv("JWT_SECRET", testSecretKey)
	defer os.Unsetenv("JWT_SECRET")

	targetUserID := "user-uuid-9988-7766"
	generatedTokenString, generationError := utils.GenerateToken(targetUserID)
	if generationError != nil {
		testingContext.Fatalf("unexpected error generating token: %v", generationError)
	}

	if generatedTokenString == "" {
		testingContext.Fatal("expected non-empty token string")
	}

	parsedToken, parseError := jwt.Parse(generatedTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, isHMAC := token.Method.(*jwt.SigningMethodHMAC); !isHMAC {
			testingContext.Fatalf("unexpected signing algorithm: %v", token.Header["alg"])
		}
		return []byte(testSecretKey), nil
	})

	if parseError != nil || !parsedToken.Valid {
		testingContext.Fatalf("expected valid token, got parse error: %v", parseError)
	}

	tokenClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		testingContext.Fatal("expected MapClaims structure from parsed token")
	}

	extractedUserID, hasUserID := tokenClaims["user_id"].(string)
	if !hasUserID || extractedUserID != targetUserID {
		testingContext.Fatalf("expected user_id %s, got %s", targetUserID, extractedUserID)
	}

	rawExpirationTimestamp, hasExpiration := tokenClaims["exp"].(float64)
	if !hasExpiration {
		testingContext.Fatal("token claims missing exp field")
	}

	expirationTime := time.Unix(int64(rawExpirationTimestamp), 0)
	minimumExpectedExpiration := time.Now().Add(119 * 24 * time.Hour)
	maximumExpectedExpiration := time.Now().Add(121 * 24 * time.Hour)

	if expirationTime.Before(minimumExpectedExpiration) || expirationTime.After(maximumExpectedExpiration) {
		testingContext.Fatalf("expected expiration around 120 days from now, got %v", expirationTime)
	}
}
