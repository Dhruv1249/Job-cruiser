package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/gin-gonic/gin"
)

/*
TestUpdateStatusRequestBinding verifies validation logic when updating application status.
*/
func TestUpdateStatusRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid_status_update",
			payload: map[string]interface{}{
				"status": "applied",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing_status_field",
			payload:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.PUT("/applications/:id/status", func(c *gin.Context) {
				var req handlers.UpdateStatusRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": req.Status})
			})

			bodyBytes, _ := json.Marshal(tt.payload)
			request := httptest.NewRequest(http.MethodPut, "/applications/app-123/status", bytes.NewBuffer(bodyBytes))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected HTTP status %d, got %d", tt.expectedStatus, recorder.Code)
			}
		})
	}
}

/*
TestCreateApplicationRequestBinding verifies validation when creating a new application entry.
*/
func TestCreateApplicationRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid_create_application",
			payload: map[string]interface{}{
				"job_id": "job-456",
				"status": "bookmarked",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing_job_id",
			payload: map[string]interface{}{
				"status": "bookmarked",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/applications", func(c *gin.Context) {
				var req struct {
					JobID  string `json:"job_id" binding:"required"`
					Status string `json:"status"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"job_id": req.JobID, "status": req.Status})
			})

			bodyBytes, _ := json.Marshal(tt.payload)
			request := httptest.NewRequest(http.MethodPost, "/applications", bytes.NewBuffer(bodyBytes))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected HTTP status %d, got %d", tt.expectedStatus, recorder.Code)
			}
		})
	}
}
