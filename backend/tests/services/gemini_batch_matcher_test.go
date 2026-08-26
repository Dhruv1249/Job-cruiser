// Package services_test contains unit tests for backend services.
package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestGeminiBatchMatchServiceAlternatingModelsFromEnvironment(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "test-model-alpha,test-model-beta,test-model-gamma")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	firstModel, errFirst := service.GetNextModelNameForTest()
	if errFirst != nil || firstModel != "test-model-alpha" {
		t.Fatalf("expected first model to be test-model-alpha, got: %s (err: %v)", firstModel, errFirst)
	}

	secondModel, errSecond := service.GetNextModelNameForTest()
	if errSecond != nil || secondModel != "test-model-beta" {
		t.Fatalf("expected second model to be test-model-beta, got: %s (err: %v)", secondModel, errSecond)
	}

	thirdModel, errThird := service.GetNextModelNameForTest()
	if errThird != nil || thirdModel != "test-model-gamma" {
		t.Fatalf("expected third model to be test-model-gamma, got: %s (err: %v)", thirdModel, errThird)
	}

	fourthModel, errFourth := service.GetNextModelNameForTest()
	if errFourth != nil || fourthModel != "test-model-alpha" {
		t.Fatalf("expected fourth model to cycle back to test-model-alpha, got: %s (err: %v)", fourthModel, errFourth)
	}
}

func TestGeminiBatchMatchServiceNoModelsConfiguredReturnsError(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "")
	t.Setenv("GEMINI_MODELS", "")
	t.Setenv("GEMINI_MODEL", "")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	modelName, errModel := service.GetNextModelNameForTest()
	if errModel == nil {
		t.Fatalf("expected error when no models configured in environment, got model: %s", modelName)
	}
}

func TestGeminiBatchMatchServiceTokenPackingBudget(t *testing.T) {
	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	sampleJobs := []services.JobSnippetData{
		{JobID: "00000000-0000-0000-0000-000000000001", Title: "Go Developer", Company: "Acme", Location: "Remote", Description: "Building microservices in Go"},
		{JobID: "00000000-0000-0000-0000-000000000002", Title: "Frontend Engineer", Company: "Beta", Location: "Remote", Description: "Building web apps in Flutter"},
		{JobID: "00000000-0000-0000-0000-000000000003", Title: "DevOps Engineer", Company: "Gamma", Location: "Remote", Description: "Managing Kubernetes clusters"},
	}

	batches := service.BuildMultiJobTokenBatchesForTest(context.Background(), sampleJobs, 200)
	if len(batches) < 1 {
		t.Fatalf("expected at least 1 batch generated, got %d", len(batches))
	}

	totalJobCount := 0
	for _, batch := range batches {
		totalJobCount += len(batch)
	}

	if totalJobCount != len(sampleJobs) {
		t.Fatalf("expected all %d jobs to be packed, got %d", len(sampleJobs), totalJobCount)
	}
}

func TestGeminiBatchMatchServiceMockGeneration(t *testing.T) {
	var requestedModel string

	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedModel = request.URL.Path

		responsePayload := map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]string{
							{
								"text": `{"results":[{"job_id":"00000000-0000-0000-0000-000000000001","user_id":"11111111-1111-1111-1111-111111111111","match_score":88,"match_reasoning":"Great Go match and location alignment.","inferred_required_yoe":3,"is_matched":true}]}`,
							},
						},
					},
				},
			},
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(responsePayload)
	}))
	defer mockServer.Close()

	service := services.NewGeminiBatchMatchService(nil, "test-key")
	service.BaseURL = mockServer.URL
	service.RateLimitInterval = 10 * time.Millisecond

	rawJSON, errGenerate := service.GenerateBatchContentForTest(context.Background(), "test-model-alpha", "evaluate prompt")
	if errGenerate != nil {
		t.Fatalf("expected generation to succeed, got error: %v", errGenerate)
	}

	if rawJSON == "" {
		t.Fatalf("expected non-empty JSON response")
	}

	parsedResults, ok := service.ParseAndValidateBatchJSONForTest(rawJSON)
	if !ok {
		t.Fatalf("expected successful JSON unmarshaling")
	}

	if len(parsedResults.Results) != 1 {
		t.Fatalf("expected 1 result object, got %d", len(parsedResults.Results))
	}

	if parsedResults.Results[0].MatchScore != 88 {
		t.Fatalf("expected match score 88, got %d", parsedResults.Results[0].MatchScore)
	}

	if requestedModel == "" {
		t.Fatalf("expected request to hit mock server")
	}
}

func TestGeminiBatchMatchServiceLocationMismatchScoreCap(t *testing.T) {
	service := services.NewGeminiBatchMatchService(nil, "test-key")

	userProfile := services.UserProfileData{
		UserID:             "11111111-1111-1111-1111-111111111111",
		Email:              "candidate@example.com",
		PreferredLocations: []string{"India (Remote)", "India (On-site)"},
		ExperienceYears:    3,
	}

	matchResult := services.GeminiMatchResult{
		JobID:               "00000000-0000-0000-0000-000000000001",
		UserID:              "11111111-1111-1111-1111-111111111111",
		MatchScore:          90,
		MatchReasoning:      "Great skill match",
		InferredRequiredYoE: 7,
		IsMatched:           true,
	}

	service.ApplyExperienceAndLocationCapsForTest(&matchResult, &userProfile)

	if matchResult.MatchScore > 25 {
		t.Fatalf("expected match score to be capped at 25 due to 4-year experience deficit, got %d", matchResult.MatchScore)
	}
}

func TestGeminiBatchMatchServicePerModelFiveErrorsDisablesModel(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "test-model-1,test-model-2")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	// 1. Four errors do not disable model
	for i := 0; i < 4; i++ {
		service.RecordModelFailureForTest(context.Background(), "test-model-1", "mock timeout error")
	}
	if service.IsModelDisabledForTest("test-model-1") {
		t.Fatalf("expected model to remain enabled after 4 errors")
	}

	// 2. Success resets consecutive error count
	service.RecordModelSuccessForTest("test-model-1")
	if service.GetModelConsecutiveErrorsForTest("test-model-1") != 0 {
		t.Fatalf("expected consecutive error count to reset to 0 upon success")
	}

	// 3. Five consecutive errors permanently disables model
	for i := 0; i < 5; i++ {
		service.RecordModelFailureForTest(context.Background(), "test-model-1", "mock 429 quota error")
	}
	if !service.IsModelDisabledForTest("test-model-1") {
		t.Fatalf("expected model to be permanently disabled after 5 consecutive errors")
	}

	// 4. Remaining model-2 continues to be used exclusively
	selectedModel1, err1 := service.GetNextModelNameForTest()
	if err1 != nil || selectedModel1 != "test-model-2" {
		t.Fatalf("expected only test-model-2 to be selected, got: %s (err: %v)", selectedModel1, err1)
	}

	selectedModel2, err2 := service.GetNextModelNameForTest()
	if err2 != nil || selectedModel2 != "test-model-2" {
		t.Fatalf("expected test-model-2 to continue being selected, got: %s (err: %v)", selectedModel2, err2)
	}
}

func TestGeminiBatchMatchServiceAllModelsDisabledShutsDownPipeline(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "alpha-model,beta-model")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	// Disable both models with 5 consecutive errors each
	for i := 0; i < 5; i++ {
		service.RecordModelFailureForTest(context.Background(), "alpha-model", "mock 500 error")
	}
	for i := 0; i < 5; i++ {
		service.RecordModelFailureForTest(context.Background(), "beta-model", "mock 500 error")
	}

	if !service.IsPipelinePermanentlyStoppedForTest() {
		t.Fatalf("expected pipeline to be permanently stopped when all models are disabled")
	}

	_, errModel := service.GetNextModelNameForTest()
	if errModel == nil {
		t.Fatalf("expected error when attempting to get next model after pipeline shutdown")
	}
}
