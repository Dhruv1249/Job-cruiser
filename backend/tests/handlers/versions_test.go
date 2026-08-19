package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/gin-gonic/gin"
)

func TestListResumeVersionsUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.GET("/api/resume-versions", versionsHandler.ListResumeVersions)

	httpRequest, _ := http.NewRequest(http.MethodGet, "/api/resume-versions", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestDeleteResumeVersionUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.DELETE("/api/resume-versions/:id", versionsHandler.DeleteResumeVersion)

	httpRequest, _ := http.NewRequest(http.MethodDelete, "/api/resume-versions/some-uuid", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestListCoverLetterVersionsReturnsUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.GET("/api/cover-letters", versionsHandler.ListCoverLetterVersions)

	httpRequest, _ := http.NewRequest(http.MethodGet, "/api/cover-letters", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestSetDefaultResumeVersionUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.PUT("/api/resume-versions/:id/default", versionsHandler.SetDefaultResumeVersion)

	httpRequest, _ := http.NewRequest(http.MethodPut, "/api/resume-versions/some-uuid/default", bytes.NewBufferString("{}"))
	httpRequest.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestGetResumeVersionPDFUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.GET("/api/resume-versions/:id/pdf", versionsHandler.GetResumeVersionPDF)

	httpRequest, _ := http.NewRequest(http.MethodGet, "/api/resume-versions/some-uuid/pdf", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestGetCoverLetterPDFUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.GET("/api/cover-letters/:id/pdf", versionsHandler.GetCoverLetterPDF)

	httpRequest, _ := http.NewRequest(http.MethodGet, "/api/cover-letters/some-uuid/pdf", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestDeleteCoverLetterVersionUnauthorizedWithoutUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	versionsHandler := handlers.NewVersionsHandler(nil, make([]byte, 32), "test-secret")

	router := gin.New()
	router.DELETE("/api/cover-letters/:id", versionsHandler.DeleteCoverLetterVersion)

	httpRequest, _ := http.NewRequest(http.MethodDelete, "/api/cover-letters/some-uuid", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", recorder.Code)
	}
}
