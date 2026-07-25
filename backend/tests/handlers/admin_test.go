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

func TestToggleAIMatchingRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		expectedState  bool
	}{
		{
			name:           "enable AI matching for user",
			body:           map[string]interface{}{"enabled": true},
			expectedStatus: http.StatusOK,
			expectedState:  true,
		},
		{
			name:           "disable AI matching for user",
			body:           map[string]interface{}{"enabled": false},
			expectedStatus: http.StatusOK,
			expectedState:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("failed to marshal JSON: %v", err)
			}

			router := gin.New()
			router.PUT("/admin/users/:id/ai-matching", func(c *gin.Context) {
				var req handlers.ToggleAIMatchingRequest
				if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"enabled": req.Enabled})
			})

			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodPut, "/admin/users/123/ai-matching", bytes.NewBuffer(jsonBytes))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response JSON: %v", err)
			}

			if response["enabled"] != tc.expectedState {
				t.Errorf("expected enabled state %v, got %v", tc.expectedState, response["enabled"])
			}
		})
	}
}
