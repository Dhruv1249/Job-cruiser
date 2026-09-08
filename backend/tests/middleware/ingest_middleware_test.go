package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/middleware"
	"github.com/gin-gonic/gin"
)

/*
TestRequireIngestKeyUnconfiguredEnvironment verifies that requests are rejected with HTTP 401 when INGEST_API_KEY is not configured in the environment.
*/
func TestRequireIngestKeyUnconfiguredEnvironment(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Unsetenv("INGEST_API_KEY")

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireIngestKey())
	routerEngine.POST("/scraper/ingest-raw", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodPost, "/scraper/ingest-raw", nil)
	testRequest.Header.Set("X-Ingest-Key", "dev-ingest-key-12345")

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("Expected status HTTP 401 when INGEST_API_KEY is unset, got %d", httpRecorder.Code)
	}
}

/*
TestRequireIngestKeyMissingHeader verifies that requests lacking the X-Ingest-Key header are rejected with HTTP 401.
*/
func TestRequireIngestKeyMissingHeader(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("INGEST_API_KEY", "configured-secret-production-key")
	defer os.Unsetenv("INGEST_API_KEY")

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireIngestKey())
	routerEngine.POST("/scraper/ingest-raw", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodPost, "/scraper/ingest-raw", nil)

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("Expected status HTTP 401 when X-Ingest-Key is missing, got %d", httpRecorder.Code)
	}
}

/*
TestRequireIngestKeyMismatchedHeader verifies that requests with an incorrect X-Ingest-Key header are rejected with HTTP 401.
*/
func TestRequireIngestKeyMismatchedHeader(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("INGEST_API_KEY", "configured-secret-production-key")
	defer os.Unsetenv("INGEST_API_KEY")

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireIngestKey())
	routerEngine.POST("/scraper/ingest-raw", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodPost, "/scraper/ingest-raw", nil)
	testRequest.Header.Set("X-Ingest-Key", "wrong-key-value")

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusUnauthorized {
		testingContext.Fatalf("Expected status HTTP 401 when X-Ingest-Key does not match, got %d", httpRecorder.Code)
	}
}

/*
TestRequireIngestKeyValidHeader verifies that requests with a valid matching X-Ingest-Key header succeed with HTTP 200.
*/
func TestRequireIngestKeyValidHeader(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("INGEST_API_KEY", "configured-secret-production-key")
	defer os.Unsetenv("INGEST_API_KEY")

	routerEngine := gin.New()
	routerEngine.Use(middleware.RequireIngestKey())
	routerEngine.POST("/scraper/ingest-raw", func(ginContext *gin.Context) {
		ginContext.Status(http.StatusOK)
	})

	httpRecorder := httptest.NewRecorder()
	testRequest, _ := http.NewRequest(http.MethodPost, "/scraper/ingest-raw", nil)
	testRequest.Header.Set("X-Ingest-Key", "configured-secret-production-key")

	routerEngine.ServeHTTP(httpRecorder, testRequest)

	if httpRecorder.Code != http.StatusOK {
		testingContext.Fatalf("Expected status HTTP 200 when X-Ingest-Key matches, got %d", httpRecorder.Code)
	}
}
