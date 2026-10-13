package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/middleware"
	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

/*
TestRequireAuthMissingHeader verifies that requests without Authorization header receive HTTP 401.
*/
func TestRequireAuthMissingHeader(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "test-auth-middleware-secret")
	defer os.Unsetenv("JWT_SECRET")

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireAuth())
	routerEngine.GET("/protected", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodGet, "/protected", nil)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("expected status 401 for missing auth header, got %d", httpRecorder.Code)
	}
}

/*
TestRequireAuthMalformedHeader verifies that headers not matching 'Bearer <token>' receive HTTP 401.
*/
func TestRequireAuthMalformedHeader(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("JWT_SECRET", "test-auth-middleware-secret")
	defer os.Unsetenv("JWT_SECRET")

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireAuth())
	routerEngine.GET("/protected", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	testRequest.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("expected status 401 for malformed header, got %d", httpRecorder.Code)
	}
}

/*
TestRequireAuthExpiredToken verifies that an expired token is rejected with HTTP 401.
*/
func TestRequireAuthExpiredToken(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	testSecretKey := "test-auth-middleware-secret"
	os.Setenv("JWT_SECRET", testSecretKey)
	defer os.Unsetenv("JWT_SECRET")

	expiredClaims := jwt.MapClaims{
		"user_id": "expired-user-id",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, _ := expiredToken.SignedString([]byte(testSecretKey))

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireAuth())
	routerEngine.GET("/protected", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	testRequest.Header.Set("Authorization", "Bearer "+tokenString)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("expected status 401 for expired token, got %d", httpRecorder.Code)
	}
}

/*
TestRequireAuthValidToken verifies that a valid token populates user_id and allows request continuation.
*/
func TestRequireAuthValidToken(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	testSecretKey := "test-auth-middleware-secret"
	os.Setenv("JWT_SECRET", testSecretKey)
	defer os.Unsetenv("JWT_SECRET")

	targetUserID := "valid-user-12345"
	validTokenString, generationError := utils.GenerateToken(targetUserID)
	if generationError != nil {
		testingContext.Fatalf("failed generating valid token: %v", generationError)
	}

	var extractedUserID string
	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireAuth())
	routerEngine.GET("/protected", func(ginContext *gin.Context) {
		extractedUserID = ginContext.GetString("user_id")
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	testRequest.Header.Set("Authorization", "Bearer "+validTokenString)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusOK {
		testingContext.Fatalf("expected status 200 for valid token, got %d", httpRecorder.Code)
	}

	if extractedUserID != targetUserID {
		testingContext.Fatalf("expected extracted user_id %s, got %s", targetUserID, extractedUserID)
	}
}
