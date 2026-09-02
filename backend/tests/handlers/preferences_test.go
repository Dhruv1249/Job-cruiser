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

func TestPreferencesRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid request with explicit AI matching disabled",
			body: map[string]interface{}{
				"full_name":                 "Jane Doe",
				"target_roles":              []string{"Backend Engineer", "DevOps SRE"},
				"work_models":               []string{"remote"},
				"min_salary":                150000,
				"currency":                  "USD",
				"master_cv_text":            "Experienced Go developer",
				"bio_experience_text":       "5 years building microservices",
				"ai_matching_enabled":       false,
				"target_resume_pages":       2,
				"target_cover_letter_pages": 1,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "valid request with AI matching enabled",
			body: map[string]interface{}{
				"full_name":                 "John Smith",
				"target_roles":              []string{"Fullstack SDE"},
				"work_models":               []string{"hybrid"},
				"min_salary":                120000,
				"currency":                  "USD",
				"master_cv_text":            "Fullstack engineer",
				"bio_experience_text":       "React and Node expert",
				"ai_matching_enabled":       true,
				"target_resume_pages":       1,
				"target_cover_letter_pages": 2,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "invalid request missing required full_name",
			body: map[string]interface{}{
				"target_roles": []string{"Backend Engineer"},
				"work_models":  []string{"remote"},
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(testCase.body)
			if err != nil {
				t.Fatalf("failed to marshal request body: %v", err)
			}

			router := gin.New()
			router.POST("/preferences", func(c *gin.Context) {
				var req handlers.PreferencesRequest
				if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"full_name":                 req.FullName,
					"ai_matching_enabled":       req.AIMatchingEnabled,
					"target_resume_pages":       req.TargetResumePages,
					"target_cover_letter_pages": req.TargetCoverLetterPages,
				})
			})

			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodPost, "/preferences", bytes.NewBuffer(jsonBytes))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != testCase.expectedStatus {
				t.Errorf("expected status %d, got %d", testCase.expectedStatus, recorder.Code)
			}
		})
	}
}

func TestParseCVRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "valid raw CV text",
			body:           map[string]interface{}{"raw_cv_text": "Experienced software engineer specializing in Go and Postgres."},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing raw CV text",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("failed to marshal JSON: %v", err)
			}

			router := gin.New()
			router.POST("/user/parse-cv", func(c *gin.Context) {
				var req handlers.ParseCVRequest
				if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "parsed"})
			})

			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodPost, "/user/parse-cv", bytes.NewBuffer(jsonBytes))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}
		})
	}
}

func TestOverleafConfigRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid request with full fields",
			body: map[string]interface{}{
				"deployment_url": "https://overleaf.example.com",
				"mcp_secret":     "custom_secret_123",
				"project_name":   "my_cvs",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid request with url only",
			body: map[string]interface{}{
				"deployment_url": "https://overleaf.example.com",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request missing url",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, _ := json.Marshal(tc.body)
			router := gin.New()
			router.POST("/preferences/overleaf", func(c *gin.Context) {
				var req handlers.OverleafConfigRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodPost, "/preferences/overleaf", bytes.NewBuffer(jsonBytes))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}
		})
	}
}

func TestPreferencesRequestBindingWithLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := map[string]interface{}{
		"full_name":     "Jane Doe",
		"email":         "jane@example.com",
		"phone":         "+1 555-0199",
		"location":      "Bengaluru, India",
		"github_url":    "https://github.com/janedoe",
		"linkedin_url":  "https://linkedin.com/in/janedoe",
		"portfolio_url": "https://janedoe.dev",
		"target_roles":  []string{"Backend Engineer"},
		"work_models":   []string{"remote"},
		"custom_links": []map[string]string{
			{"label": "Blog", "url": "https://blog.janedoe.dev"},
			{"label": "LeetCode", "url": "https://leetcode.com/janedoe"},
		},
	}

	jsonBytes, err := json.Marshal(testBody)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	router := gin.New()
	router.POST("/preferences", func(c *gin.Context) {
		var req handlers.PreferencesRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
			return
		}

		if req.Email != "jane@example.com" {
			t.Errorf("expected email 'jane@example.com', got '%s'", req.Email)
		}
		if len(req.CustomLinks) != 2 {
			t.Errorf("expected 2 custom links, got %d", len(req.CustomLinks))
		}
		if req.CustomLinks[0].Label != "Blog" || req.CustomLinks[0].URL != "https://blog.janedoe.dev" {
			t.Errorf("unexpected first custom link: %+v", req.CustomLinks[0])
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodPost, "/preferences", bytes.NewBuffer(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestPreferencesRequestBindingWithMatchNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name                        string
		notificationEnabled         bool
		thresholdPercentage         int
		expectedNotificationEnabled bool
		expectedThresholdPercentage int
	}{
		{
			name:                        "default disabled notification with custom threshold",
			notificationEnabled:         false,
			thresholdPercentage:         85,
			expectedNotificationEnabled: false,
			expectedThresholdPercentage: 85,
		},
		{
			name:                        "enabled notification with 80 percent threshold",
			notificationEnabled:         true,
			thresholdPercentage:         80,
			expectedNotificationEnabled: true,
			expectedThresholdPercentage: 80,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testBody := map[string]interface{}{
				"full_name":                            "Jane Doe",
				"target_roles":                         []string{"Backend Engineer"},
				"work_models":                          []string{"remote"},
				"match_threshold_notification_enabled": tc.notificationEnabled,
				"match_threshold_percentage":          tc.thresholdPercentage,
			}

			jsonBytes, err := json.Marshal(testBody)
			if err != nil {
				t.Fatalf("failed to marshal request body: %v", err)
			}

			router := gin.New()
			router.POST("/preferences", func(c *gin.Context) {
				var req handlers.PreferencesRequest
				if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
					return
				}

				if req.MatchThresholdNotificationEnabled != tc.expectedNotificationEnabled {
					t.Errorf("expected notification enabled %v, got %v", tc.expectedNotificationEnabled, req.MatchThresholdNotificationEnabled)
				}
				if req.MatchThresholdPercentage != tc.expectedThresholdPercentage {
					t.Errorf("expected threshold percentage %d, got %d", tc.expectedThresholdPercentage, req.MatchThresholdPercentage)
				}

				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest(http.MethodPost, "/preferences", bytes.NewBuffer(jsonBytes))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}
		})
	}
}
