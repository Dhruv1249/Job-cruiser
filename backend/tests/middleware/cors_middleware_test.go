package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/middleware"
	"github.com/gin-gonic/gin"
)

// TestCORSMiddlewarePreflightRequest verifies that OPTIONS requests return HTTP status 204 with CORS headers.
func TestCORSMiddlewarePreflightRequest(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	routerEngine := gin.New()
	routerEngine.Use(middleware.CORSMiddleware())
	routerEngine.GET("/api/test", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodOptions, "/api/test", nil)
	testRequest.Header.Set("Origin", "https://jobcruiser.web.app")
	testRequest.Header.Set("Access-Control-Request-Method", "GET")

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusNoContent {
		testingContext.Fatalf("Expected status HTTP 204 for OPTIONS preflight, got %d", httpRecorder.Code)
	}

	allowOriginHeader := httpRecorder.Header().Get("Access-Control-Allow-Origin")
	if allowOriginHeader != "https://jobcruiser.web.app" {
		testingContext.Errorf("Expected Access-Control-Allow-Origin to be 'https://jobcruiser.web.app', got '%s'", allowOriginHeader)
	}
}

// TestCORSMiddlewareAllowedOriginsEnvironmentVariable verifies matching against configured allowed origins.
func TestCORSMiddlewareAllowedOriginsEnvironmentVariable(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("ALLOWED_ORIGINS", "https://jobcruiser.web.app, https://jobcruiser.firebaseapp.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	routerEngine := gin.New()
	routerEngine.Use(middleware.CORSMiddleware())
	routerEngine.GET("/api/test", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	testRequest.Header.Set("Origin", "https://jobcruiser.firebaseapp.com")

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusOK {
		testingContext.Fatalf("Expected status HTTP 200, got %d", httpRecorder.Code)
	}

	allowOriginHeader := httpRecorder.Header().Get("Access-Control-Allow-Origin")
	if allowOriginHeader != "https://jobcruiser.firebaseapp.com" {
		testingContext.Errorf("Expected Access-Control-Allow-Origin to be 'https://jobcruiser.firebaseapp.com', got '%s'", allowOriginHeader)
	}
}
