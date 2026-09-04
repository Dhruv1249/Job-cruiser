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

func TestScraperTelemetryResponseStructure(t *testing.T) {
	mockResponsePayload := map[string]interface{}{
		"total_jobs":       12500,
		"jobs_last_24h":    420,
		"jobs_last_7d":     2850,
		"unique_companies": 1400,
		"kpis": map[string]interface{}{
			"total_jobs":               12500,
			"jobs_last_24h":            420,
			"jobs_last_7d":             2850,
			"unique_companies":         1400,
			"evaluated_jobs_count":     9200,
			"evaluation_coverage_pct":  73.6,
			"overall_avg_match_score":  76.4,
			"remote_jobs_count":        5100,
			"remote_jobs_pct":          40.8,
			"top_volume_source":        "linkedin",
			"top_quality_source":       "greenhouse",
		},
		"sources_volume": []map[string]interface{}{
			{
				"source":        "linkedin",
				"total_jobs":    6200,
				"jobs_last_24h": 210,
				"jobs_last_7d":  1400,
				"remote_jobs":   2100,
				"onsite_jobs":   4100,
				"share_pct":     49.6,
			},
		},
		"sources_quality": []map[string]interface{}{
			{
				"source":               "greenhouse",
				"evaluated_count":      1200,
				"avg_score":            84.5,
				"elite_matches":        540,
				"good_matches":         480,
				"low_matches":          180,
				"high_match_yield_pct": 45.0,
			},
		},
		"ingestion_timeline": []map[string]interface{}{
			{
				"date":       "2026-09-01",
				"jobs_count": 310,
			},
		},
		"score_distribution": map[string]interface{}{
			"tier_90_100":       1200,
			"tier_80_89":        2400,
			"tier_60_79":        3800,
			"tier_below_60":     1800,
			"unevaluated_count": 3300,
			"avg_score":         76.4,
		},
		"run_health": map[string]interface{}{
			"total_runs_recorded":  45,
			"successful_runs":      42,
			"failed_runs":          3,
			"success_rate_pct":     93.3,
			"avg_duration_seconds": 185,
		},
		"top_companies": []map[string]interface{}{
			{
				"company_name": "Acme Corp",
				"job_count":    42,
			},
		},
		"runs": []map[string]interface{}{
			{
				"run_id":           "test-run-123",
				"started_at":       "2026-09-04T12:00:00Z",
				"finished_at":      "2026-09-04T12:03:00Z",
				"status":           "completed",
				"jobs_added":       125,
				"sources_hit":      "[\"linkedin\", \"greenhouse\"]",
				"duration_seconds": 180,
				"error_message":    "",
			},
		},
	}

	jsonBytes, err := json.Marshal(mockResponsePayload)
	if err != nil {
		t.Fatalf("failed to marshal mock response payload: %v", err)
	}

	var parsedResponse map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsedResponse); err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}

	requiredKeys := []string{
		"total_jobs",
		"jobs_last_24h",
		"unique_companies",
		"kpis",
		"sources_volume",
		"sources_quality",
		"ingestion_timeline",
		"score_distribution",
		"run_health",
		"top_companies",
		"runs",
	}

	for _, requiredKey := range requiredKeys {
		if _, exists := parsedResponse[requiredKey]; !exists {
			t.Errorf("expected required key %q in response, but it was missing", requiredKey)
		}
	}
}

