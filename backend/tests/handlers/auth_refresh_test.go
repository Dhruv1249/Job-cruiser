package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

/*
TestRefreshTokenUnauthorized verifies that RefreshToken rejects requests without user context with HTTP 401.
*/
func TestRefreshTokenUnauthorized(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	authHandler := &handlers.AuthHandler{DB: nil}

	routerEngine := gin.New()
	routerEngine.POST("/api/auth/refresh", authHandler.RefreshToken)

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("expected HTTP 401 for unauthorized refresh, got %d", httpRecorder.Code)
	}
}

/*
TestRefreshTokenSuccess verifies that RefreshToken produces a renewed token with four-month expiration for authenticated users.
*/
func TestRefreshTokenSuccess(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	testSecretKey := "unit-test-refresh-token-secret"
	os.Setenv("JWT_SECRET", testSecretKey)
	defer os.Unsetenv("JWT_SECRET")

	authHandler := &handlers.AuthHandler{DB: nil}
	targetUserID := "authenticated-user-uuid-12345"

	routerEngine := gin.New()
	routerEngine.POST("/api/auth/refresh", func(ginContext *gin.Context) {
		ginContext.Set("user_id", targetUserID)
		authHandler.RefreshToken(ginContext)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", nil)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusOK {
		testingContext.Fatalf("expected HTTP 200 for successful refresh, got %d", httpRecorder.Code)
	}

	var responseBody map[string]interface{}
	decodeError := json.Unmarshal(httpRecorder.Body.Bytes(), &responseBody)
	if decodeError != nil {
		testingContext.Fatalf("failed decoding response body: %v", decodeError)
	}

	rawToken, hasToken := responseBody["token"].(string)
	if !hasToken || rawToken == "" {
		testingContext.Fatal("expected non-empty token in refresh response")
	}

	rawUserID, hasUserID := responseBody["user_id"].(string)
	if !hasUserID || rawUserID != targetUserID {
		testingContext.Fatalf("expected user_id %s, got %s", targetUserID, rawUserID)
	}

	parsedToken, parseError := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(testSecretKey), nil
	})
	if parseError != nil || !parsedToken.Valid {
		testingContext.Fatalf("expected valid refreshed token: %v", parseError)
	}

	tokenClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		testingContext.Fatal("failed extracting claims from refreshed token")
	}

	rawExpirationTimestamp := tokenClaims["exp"].(float64)
	expirationTime := time.Unix(int64(rawExpirationTimestamp), 0)
	minimumExpectedExpiration := time.Now().Add(119 * 24 * time.Hour)

	if expirationTime.Before(minimumExpectedExpiration) {
		testingContext.Fatalf("expected expiration at least 119 days in future, got %v", expirationTime)
	}
}

/*
TestGetMeRequiresAuthentication verifies that GetMe returns HTTP 401 when user_id is absent from request context.
*/
func TestGetMeRequiresAuthentication(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	authHandler := &handlers.AuthHandler{DB: nil}

	routerEngine := gin.New()
	routerEngine.GET("/api/user/me", authHandler.GetMe)

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodGet, "/api/user/me", nil)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("expected HTTP 401 for GetMe without auth, got %d", httpRecorder.Code)
	}
}
