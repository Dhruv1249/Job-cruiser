package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/gin-gonic/gin"
)

func TestTailorResumeRequiresJobID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/resume", func(ginContext *gin.Context) {
		ginContext.Set("user_id", "test-user")
		tailorHandler.TailorResume(ginContext)
	})

	requestBody := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(requestBody)
	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/resume", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing job_id, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestTailorResumeRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/resume", tailorHandler.TailorResume)

	requestBody := map[string]interface{}{
		"job_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
	}
	jsonBytes, _ := json.Marshal(requestBody)
	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/resume", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", recorder.Code)
	}
}

func TestGenerateCoverLetterRequiresJobID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/cover-letter", func(ginContext *gin.Context) {
		ginContext.Set("user_id", "test-user")
		tailorHandler.GenerateCoverLetter(ginContext)
	})

	requestBody := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(requestBody)
	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/cover-letter", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing job_id, got %d. Body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGenerateCoverLetterRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/cover-letter", tailorHandler.GenerateCoverLetter)

	requestBody := map[string]interface{}{
		"job_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
	}
	jsonBytes, _ := json.Marshal(requestBody)
	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/cover-letter", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", recorder.Code)
	}
}

func TestTailorHandlerNewConstructor(t *testing.T) {
	aesKey := make([]byte, 32)
	for index := range aesKey {
		aesKey[index] = byte(index + 1)
	}
	tailorService := services.NewResumeTailorService("http://gemini.test", "key", nil)
	tailorHandler := handlers.NewTailorHandler(tailorService, nil, aesKey, "mcp-secret")
	if tailorHandler == nil {
		t.Fatal("expected non-nil tailorHandler")
	}
	if len(tailorHandler.AESKey) != 32 {
		t.Fatalf("expected 32 byte AES key, got %d", len(tailorHandler.AESKey))
	}
	if tailorHandler.MCPSecret != "mcp-secret" {
		t.Fatalf("expected mcp-secret, got %s", tailorHandler.MCPSecret)
	}
}

func TestTailorApplicationAsyncRequiresJobID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/application", func(ginContext *gin.Context) {
		ginContext.Set("user_id", "test-user")
		tailorHandler.TailorApplicationAsync(ginContext)
	})

	requestBody := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(requestBody)
	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/application", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing job_id, got %d", recorder.Code)
	}
}

func TestTailorApplicationAsyncRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/application", tailorHandler.TailorApplicationAsync)

	requestBody := map[string]interface{}{
		"job_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
	}
	jsonBytes, _ := json.Marshal(requestBody)
	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/application", bytes.NewBuffer(jsonBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}


func TestListTemplatesRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.GET("/api/tailor/templates", tailorHandler.ListTemplates)

	httpRequest, _ := http.NewRequest(http.MethodGet, "/api/tailor/templates", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", recorder.Code)
	}
}

func TestSeedDefaultTemplatesRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tailorHandler := handlers.NewTailorHandler(nil, nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.POST("/api/tailor/templates/seed", tailorHandler.SeedDefaultTemplates)

	httpRequest, _ := http.NewRequest(http.MethodPost, "/api/tailor/templates/seed", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request, got %d", recorder.Code)
	}
}
