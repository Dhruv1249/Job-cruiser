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
		"country":       "India",
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

		if req.Country != "India" {
			t.Errorf("expected country India, got %q", req.Country)
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
		criteriaPrompt              string
		expectedNotificationEnabled bool
		expectedThresholdPercentage int
		expectedCriteriaPrompt      string
	}{
		{
			name:                        "default disabled notification with custom threshold",
			notificationEnabled:         false,
			thresholdPercentage:         85,
			criteriaPrompt:              "",
			expectedNotificationEnabled: false,
			expectedThresholdPercentage: 85,
			expectedCriteriaPrompt:      "",
		},
		{
			name:                        "enabled notification with 80 percent threshold and custom criteria",
			notificationEnabled:         true,
			thresholdPercentage:         80,
			criteriaPrompt:              "Must be 100% remote and Go or Kubernetes",
			expectedNotificationEnabled: true,
			expectedThresholdPercentage: 80,
			expectedCriteriaPrompt:      "Must be 100% remote and Go or Kubernetes",
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
				"notification_prompt_criteria":        tc.criteriaPrompt,
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
				if req.NotificationPromptCriteria != tc.expectedCriteriaPrompt {
					t.Errorf("expected notification prompt criteria %s, got %s", tc.expectedCriteriaPrompt, req.NotificationPromptCriteria)
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

func TestPreferencesRequestBindingWithStructuredResumeDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := map[string]interface{}{
		"full_name":    "Dhruv Dev",
		"target_roles": []string{"Backend Engineer"},
		"work_models":  []string{"remote"},
		"skills":       []string{"Go", "Postgres", "Flutter", "Docker"},
		"projects": []map[string]interface{}{
			{
				"title":          "Job Cruiser",
				"tech_stack":     []string{"Go", "Postgres", "Flutter"},
				"description":    "Full automated job matching platform",
				"link":           "https://github.com/example/job-cruiser",
				"github_url":     "https://github.com/example/job-cruiser",
				"deployment_url": "https://jobcruiser.app",
			},
		},
		"experiences": []map[string]interface{}{
			{
				"company":    "Tech Corp",
				"role":       "Senior Software Engineer",
				"duration":   "2023 - Present",
				"highlights": "Led development of core matching pipelines",
			},
		},
		"education": []map[string]interface{}{
			{
				"institution": "Tech University",
				"degree":      "Bachelor of Technology",
				"year":        "2020 - 2024",
				"grade":       "8.8 CGPA",
			},
		},
		"achievements": []map[string]interface{}{
			{
				"title":   "Hackathon Winner",
				"details": "First place in National Cloud Challenge",
			},
		},
		"certifications": []map[string]interface{}{
			{
				"name":   "AWS Certified Solutions Architect",
				"issuer": "Amazon Web Services",
			},
		},
	}

	jsonBytes, err := json.Marshal(testBody)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	router := gin.New()
	router.POST("/preferences", func(c *gin.Context) {
		var req handlers.PreferencesRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
			return
		}

		if len(req.Projects) != 1 || req.Projects[0].Title != "Job Cruiser" || req.Projects[0].GithubURL != "https://github.com/example/job-cruiser" || req.Projects[0].DeploymentURL != "https://jobcruiser.app" {
			t.Errorf("unexpected projects binding: %+v", req.Projects)
		}
		if len(req.Experiences) != 1 || req.Experiences[0].Company != "Tech Corp" {
			t.Errorf("unexpected experiences binding: %+v", req.Experiences)
		}
		if len(req.Skills) != 4 || req.Skills[0] != "Go" {
			t.Errorf("unexpected skills binding: %+v", req.Skills)
		}
		if len(req.Education) != 1 || req.Education[0].Institution != "Tech University" {
			t.Errorf("unexpected education binding: %+v", req.Education)
		}
		if len(req.Achievements) != 1 || req.Achievements[0].Title != "Hackathon Winner" {
			t.Errorf("unexpected achievements binding: %+v", req.Achievements)
		}
		if len(req.Certifications) != 1 || req.Certifications[0].Name != "AWS Certified Solutions Architect" {
			t.Errorf("unexpected certifications binding: %+v", req.Certifications)
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

func TestProfileUpdateRequestBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := map[string]interface{}{
		"full_name":      "Dhruv Dev",
		"email":          "dhruv@example.com",
		"phone":          "+91 9876543210",
		"location":       "Bangalore, India",
		"country":        "India",
		"bio_summary":    "Passionate backend developer with expertise in Go and cloud systems.",
		"linkedin_url":   "https://linkedin.com/in/dhruv",
		"github_url":     "https://github.com/dhruv",
		"portfolio_url":  "https://dhruv.dev",
		"skills":         []string{"Go", "Postgres", "Redis"},
		"projects": []map[string]interface{}{
			{
				"title":       "Open Overleaf",
				"tech_stack":  []string{"Next.js", "Docker", "TypeScript"},
				"description": "Self-hosted LaTeX editor",
				"link":        "https://github.com/example/overleaf",
			},
		},
	}

	jsonBytes, err := json.Marshal(testBody)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	router := gin.New()
	router.POST("/user/profile", func(c *gin.Context) {
		var req handlers.ProfileUpdateRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": bindErr.Error()})
			return
		}

		if req.FullName != "Dhruv Dev" || req.Email != "dhruv@example.com" || req.Country != "India" {
			t.Errorf("unexpected profile binding: %+v", req)
		}
		if len(req.Projects) != 1 || req.Projects[0].Title != "Open Overleaf" {
			t.Errorf("unexpected projects in profile update: %+v", req.Projects)
		}

		c.JSON(http.StatusOK, gin.H{"status": "profile_updated"})
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodPost, "/user/profile", bytes.NewBuffer(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestExtractStructuredResumeDetails(t *testing.T) {
	masterCVText := "Raw resume text here...\n\n--- STRUCTURED RESUME DETAILS ---\n{\"skills\":[\"Go\",\"Docker\"],\"projects\":[{\"title\":\"Test Project\",\"tech_stack\":[\"Go\"],\"description\":\"Desc\",\"link\":\"\",\"github_url\":\"https://github.com/example/test\",\"deployment_url\":\"https://test.example.com\",\"duration\":\"Jan 2023 - Present\"}],\"achievements\":[{\"title\":\"Hackathon Winner\",\"details\":\"1st place\",\"date\":\"Oct 2024\"}],\"certifications\":[{\"name\":\"AWS SAA\",\"issuer\":\"Amazon\",\"date\":\"May 2023\"}],\"research_patents\":[{\"title\":\"Distributed Consensus\",\"authors\":\"Jane Doe\",\"date\":\"2024\"}],\"open_source_contributions\":[{\"project_name\":\"Kubernetes\",\"contribution_role\":\"Maintainer\",\"tech_stack\":[\"Go\"],\"duration\":\"2022 - Present\"}]}"

	exp, proj, edu, skills, ach, cert, research, openSource := handlers.ExtractStructuredResumeDetails(masterCVText)

	if len(skills) != 2 || skills[0] != "Go" {
		t.Errorf("expected 2 skills, got %v", skills)
	}
	if len(proj) != 1 || proj[0].Title != "Test Project" || proj[0].Duration != "Jan 2023 - Present" || proj[0].GithubURL != "https://github.com/example/test" || proj[0].DeploymentURL != "https://test.example.com" {
		t.Errorf("expected 1 project with duration and URLs, got %v", proj)
	}
	if len(ach) != 1 || ach[0].Title != "Hackathon Winner" || ach[0].Date != "Oct 2024" {
		t.Errorf("expected 1 achievement with date, got %v", ach)
	}
	if len(cert) != 1 || cert[0].Name != "AWS SAA" || cert[0].Date != "May 2023" {
		t.Errorf("expected 1 certification with date, got %v", cert)
	}
	if len(research) != 1 || research[0].Title != "Distributed Consensus" {
		t.Errorf("expected 1 research item, got %v", research)
	}
	if len(openSource) != 1 || openSource[0].ProjectName != "Kubernetes" || openSource[0].Duration != "2022 - Present" {
		t.Errorf("expected 1 open source item, got %v", openSource)
	}
	if len(exp) != 0 || len(edu) != 0 {
		t.Errorf("expected empty slices for unprovided fields")
	}

	emptyText := "No delimiter in this text"
	exp2, proj2, edu2, skills2, ach2, cert2, research2, openSource2 := handlers.ExtractStructuredResumeDetails(emptyText)
	if len(exp2) != 0 || len(proj2) != 0 || len(edu2) != 0 || len(skills2) != 0 || len(ach2) != 0 || len(cert2) != 0 || len(research2) != 0 || len(openSource2) != 0 {
		t.Errorf("expected empty slices for missing delimiter")
	}
}

func TestProfileUpdateBioBinding(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := map[string]interface{}{
		"full_name":           "Dhruv Dev",
		"bio_experience_text": "Experienced software engineer",
	}

	jsonBytes, marshalError := json.Marshal(testBody)
	if marshalError != nil {
		testingContext.Fatalf("failed to marshal JSON: %v", marshalError)
	}

	router := gin.New()
	router.POST("/user/profile", func(ginContext *gin.Context) {
		var requestPayload handlers.ProfileUpdateRequest
		if bindError := ginContext.ShouldBindJSON(&requestPayload); bindError != nil {
			ginContext.JSON(http.StatusBadRequest, gin.H{"error": bindError.Error()})
			return
		}

		effectiveBio := requestPayload.BioSummary
		if effectiveBio == "" {
			effectiveBio = requestPayload.BioExperienceText
		}

		if effectiveBio != "Experienced software engineer" {
			testingContext.Errorf("expected effective bio, got %s", effectiveBio)
		}

		ginContext.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodPost, "/user/profile", bytes.NewBuffer(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		testingContext.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

// TestPreferencesBindingAndBackfill verifies JSON binding for multi-criteria preferences.
func TestPreferencesBindingAndBackfill(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := map[string]interface{}{
		"full_name":         "Dhruv Dev",
		"target_roles":      []string{"Backend Engineer", "Fullstack SDE"},
		"target_industries": []string{"AI / ML", "Fintech"},
		"target_locations":  []string{"India (Remote)", "Global Remote"},
		"work_models":       []string{"remote", "hybrid"},
		"min_salary":        120000,
		"currency":          "USD",
	}

	jsonBytes, marshalError := json.Marshal(testBody)
	if marshalError != nil {
		testingContext.Fatalf("failed to marshal JSON: %v", marshalError)
	}

	router := gin.New()
	router.POST("/preferences", func(ginContext *gin.Context) {
		var requestPayload handlers.PreferencesRequest
		if bindError := ginContext.ShouldBindJSON(&requestPayload); bindError != nil {
			ginContext.JSON(http.StatusBadRequest, gin.H{"error": bindError.Error()})
			return
		}

		if len(requestPayload.TargetRoles) != 2 || requestPayload.TargetRoles[0] != "Backend Engineer" {
			testingContext.Errorf("unexpected target roles: %v", requestPayload.TargetRoles)
		}
		if len(requestPayload.WorkModels) != 2 || requestPayload.WorkModels[0] != "remote" {
			testingContext.Errorf("unexpected work models: %v", requestPayload.WorkModels)
		}
		if requestPayload.MinSalary != 120000 {
			testingContext.Errorf("unexpected min salary: %d", requestPayload.MinSalary)
		}

		ginContext.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodPost, "/preferences", bytes.NewBuffer(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		testingContext.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestPreferencesBindingWithAllOptions(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)

	testBody := map[string]interface{}{
		"full_name":         "Dhruv All Options",
		"target_roles":      []string{},
		"target_industries": []string{},
		"target_locations":  []string{"Any Location"},
		"work_models":       []string{"any"},
		"min_salary":        0,
		"currency":          "USD",
	}

	jsonBytes, marshalError := json.Marshal(testBody)
	if marshalError != nil {
		testingContext.Fatalf("failed to marshal JSON: %v", marshalError)
	}

	router := gin.New()
	router.POST("/preferences", func(ginContext *gin.Context) {
		var requestPayload handlers.PreferencesRequest
		if bindError := ginContext.ShouldBindJSON(&requestPayload); bindError != nil {
			ginContext.JSON(http.StatusBadRequest, gin.H{"error": bindError.Error()})
			return
		}

		if len(requestPayload.TargetRoles) != 0 {
			testingContext.Errorf("expected empty target roles, got: %v", requestPayload.TargetRoles)
		}
		if len(requestPayload.TargetIndustries) != 0 {
			testingContext.Errorf("expected empty target industries, got: %v", requestPayload.TargetIndustries)
		}
		if len(requestPayload.TargetLocations) != 1 || requestPayload.TargetLocations[0] != "Any Location" {
			testingContext.Errorf("expected Any Location, got: %v", requestPayload.TargetLocations)
		}
		if len(requestPayload.WorkModels) != 1 || requestPayload.WorkModels[0] != "any" {
			testingContext.Errorf("expected work models [any], got: %v", requestPayload.WorkModels)
		}
		if requestPayload.MinSalary != 0 {
			testingContext.Errorf("expected min salary 0, got: %d", requestPayload.MinSalary)
		}

		ginContext.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodPost, "/preferences", bytes.NewBuffer(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		testingContext.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestCustomFormAnswersBinding(testingContext *testing.T) {
	gin.SetMode(gin.TestMode)

	rawPayload := map[string]interface{}{
		"full_name": "Alex Mercer",
		"custom_form_answers": map[string]interface{}{
			"visa_sponsorship":   "No",
			"work_authorization": "Authorized in US",
			"notice_period":      "Immediate",
			"custom_fields": []interface{}{
				map[string]interface{}{
					"id":       "field-1",
					"label":    "Why Us?",
					"value":    "Passionate about distributed systems",
					"category": "Short Answers",
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(rawPayload)
	if err != nil {
		testingContext.Fatalf("marshal failed: %v", err)
	}

	router := gin.New()
	router.POST("/preferences", func(ginContext *gin.Context) {
		var requestPayload handlers.PreferencesRequest
		if bindError := ginContext.ShouldBindJSON(&requestPayload); bindError != nil {
			ginContext.JSON(http.StatusBadRequest, gin.H{"error": bindError.Error()})
			return
		}

		if requestPayload.CustomFormAnswers == nil {
			testingContext.Fatalf("expected custom form answers map, got nil")
		}

		if requestPayload.CustomFormAnswers["visa_sponsorship"] != "No" {
			testingContext.Errorf("expected visa_sponsorship 'No', got: %v", requestPayload.CustomFormAnswers["visa_sponsorship"])
		}

		ginContext.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest(http.MethodPost, "/preferences", bytes.NewBuffer(jsonBytes))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		testingContext.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

