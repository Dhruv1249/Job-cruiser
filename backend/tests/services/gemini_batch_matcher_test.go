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

func TestGeminiBatchMatchServicePerModelSixErrorsDisablesModel(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "test-model-1,test-model-2")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	for i := 0; i < 5; i++ {
		service.RecordModelFailureForTest(context.Background(), "test-model-1", "mock fatal error: invalid schema")
	}
	if service.IsModelDisabledForTest("test-model-1") {
		t.Fatalf("expected model to remain enabled after 5 errors")
	}

	service.RecordModelSuccessForTest("test-model-1")
	if service.GetModelConsecutiveErrorsForTest("test-model-1") != 0 {
		t.Fatalf("expected consecutive error count to reset to 0 upon success")
	}

	for i := 0; i < 6; i++ {
		service.RecordModelFailureForTest(context.Background(), "test-model-1", "mock fatal error: unsupported format")
	}
	if !service.IsModelDisabledForTest("test-model-1") {
		t.Fatalf("expected model to be disabled after 6 errors in current run")
	}

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

	for i := 0; i < 6; i++ {
		service.RecordModelFailureForTest(context.Background(), "alpha-model", "mock fatal error: invalid API key")
	}
	for i := 0; i < 6; i++ {
		service.RecordModelFailureForTest(context.Background(), "beta-model", "mock fatal error: invalid API key")
	}

	if !service.IsPipelinePermanentlyStoppedForTest() {
		t.Fatalf("expected pipeline to be permanently stopped when all models are disabled")
	}

	_, errModel := service.GetNextModelNameForTest()
	if errModel == nil {
		t.Fatalf("expected error when attempting to get next model after pipeline shutdown")
	}
}

func TestGeminiBatchMatchServiceTurnBasedRecovery(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "alpha-model,beta-model")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	for i := 0; i < 6; i++ {
		service.RecordModelFailureForTest(context.Background(), "alpha-model", "mock fatal error: unsupported operation")
	}

	if !service.IsModelDisabledForTest("alpha-model") {
		t.Fatalf("expected alpha-model to be disabled for current run")
	}

	selectedModel, errModel := service.GetNextModelNameForTest()
	if errModel != nil || selectedModel != "beta-model" {
		t.Fatalf("expected beta-model to be selected during current run, got %s (err: %v)", selectedModel, errModel)
	}

	service.ResetRunErrorsIfHealthyForTest()

	if service.IsModelDisabledForTest("alpha-model") {
		t.Fatalf("expected alpha-model to recover on next run")
	}

	firstModelNextRun, errNext := service.GetNextModelNameForTest()
	if errNext != nil || (firstModelNextRun != "alpha-model" && firstModelNextRun != "beta-model") {
		t.Fatalf("expected active model selection on next run, got %s (err: %v)", firstModelNextRun, errNext)
	}
}

func TestIsTransientHighDemandError(t *testing.T) {
	transientErrors := []string{
		"503 Service Unavailable",
		"The model is overloaded. Please try again later.",
		"gemini api error: high demand",
		"resource has been exhausted (ResourceExhausted)",
		"429 Too Many Requests: quota exceeded",
		"context deadline exceeded (timeout)",
		"502 Bad Gateway",
		"504 Gateway Timeout",
	}

	for _, errString := range transientErrors {
		if !services.IsTransientHighDemandError(errString) {
			t.Fatalf("expected %q to be identified as transient high demand error", errString)
		}
	}

	nonTransientErrors := []string{
		"400 Bad Request: invalid argument",
		"401 Unauthorized: invalid api key",
		"404 Not Found: model does not exist",
		"invalid character 'x' looking for beginning of value",
	}

	for _, errString := range nonTransientErrors {
		if services.IsTransientHighDemandError(errString) {
			t.Fatalf("expected %q NOT to be identified as transient high demand error", errString)
		}
	}
}

func TestGeminiBatchMatchService503HighDemandDoesNotHaltPipeline(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "gemini-model-1,gemini-model-2")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	for i := 0; i < 10; i++ {
		service.RecordModelFailureForTest(context.Background(), "gemini-model-1", "503 Service Unavailable: The model is overloaded. Please try again later.")
		service.RecordModelFailureForTest(context.Background(), "gemini-model-2", "503 high demand on server")
	}

	if service.IsPipelinePermanentlyStoppedForTest() {
		t.Fatalf("expected pipeline NOT to be permanently stopped on 503 high demand errors")
	}

	service.ResetRunErrorsIfHealthyForTest()

	if service.IsModelDisabledForTest("gemini-model-1") || service.IsModelDisabledForTest("gemini-model-2") {
		t.Fatalf("expected models to be fully active on next run after transient 503 recovery")
	}

	activeModel, errModel := service.GetNextModelNameForTest()
	if errModel != nil || (activeModel != "gemini-model-1" && activeModel != "gemini-model-2") {
		t.Fatalf("expected active model selection on next run, got: %s (err: %v)", activeModel, errModel)
	}
}

func TestGeminiBatchMatchServiceModelTokenBudgetsConfig(t *testing.T) {
	t.Setenv("GEMINI_MODEL_TOKEN_BUDGETS", "gemini-3.5-flash-lite=150000,gemini-3.1-flash-lite=100000,custom-model=300000")
	t.Setenv("GEMINI_BATCH_TOKEN_BUDGET", "120000")

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")

	budget35 := service.GetTargetTokenBudgetForModel("gemini-3.5-flash-lite")
	if budget35 != 150000 {
		t.Fatalf("expected 150000 budget for gemini-3.5-flash-lite, got %d", budget35)
	}

	budget31 := service.GetTargetTokenBudgetForModel("gemini-3.1-flash-lite")
	if budget31 != 100000 {
		t.Fatalf("expected 100000 budget for gemini-3.1-flash-lite, got %d", budget31)
	}

	budgetCustom := service.GetTargetTokenBudgetForModel("custom-model")
	if budgetCustom != 300000 {
		t.Fatalf("expected 300000 budget for custom-model, got %d", budgetCustom)
	}

	budgetFallback := service.GetTargetTokenBudgetForModel("unlisted-future-model")
	if budgetFallback != 120000 {
		t.Fatalf("expected fallback budget 120000 for unlisted-future-model, got %d", budgetFallback)
	}
}

func TestGeminiBatchMatchSchemaJSONIncludesNotificationCriteriaAndSalaryFields(t *testing.T) {
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(services.GeminiBatchJobMatchSchemaJSON), &schemaMap); err != nil {
		t.Fatalf("expected GeminiBatchJobMatchSchemaJSON to be valid JSON, got error: %v", err)
	}

	properties, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected top-level properties map in schema")
	}

	results, ok := properties["results"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected results object in schema properties")
	}

	items, ok := results["items"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected items object in results schema")
	}

	itemProperties, ok := items["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties in item schema")
	}

	expectedFields := []string{
		"job_id",
		"user_id",
		"match_score",
		"match_reasoning",
		"inferred_required_yoe",
		"standardized_location",
		"work_model",
		"is_matched",
		"notification_criteria_met",
		"salary_min",
		"salary_max",
		"salary_currency",
		"salary_period",
		"employment_type",
	}

	for _, field := range expectedFields {
		if _, exists := itemProperties[field]; !exists {
			t.Fatalf("expected item schema property %q to exist in GeminiBatchJobMatchSchemaJSON", field)
		}
	}

	requiredList, ok := items["required"].([]interface{})
	if !ok {
		t.Fatalf("expected required list in items schema")
	}

	requiredMap := make(map[string]bool)
	for _, req := range requiredList {
		if reqStr, isString := req.(string); isString {
			requiredMap[reqStr] = true
		}
	}

	if !requiredMap["notification_criteria_met"] {
		t.Fatalf("expected notification_criteria_met to be in items required list")
	}
}

func TestGeminiBatchMatchParsingNotificationCriteriaAndSalary(t *testing.T) {
	sampleJSON := `{
		"results": [
			{
				"job_id": "00000000-0000-0000-0000-000000000001",
				"user_id": "11111111-1111-1111-1111-111111111111",
				"match_score": 85,
				"match_reasoning": "Strong match for backend Go internship.",
				"inferred_required_yoe": 0,
				"standardized_location": "Bengaluru, India",
				"work_model": "onsite",
				"is_matched": true,
				"notification_criteria_met": true,
				"salary_min": 25000,
				"salary_max": 25000,
				"salary_currency": "INR",
				"salary_period": "monthly",
				"employment_type": "intern"
			},
			{
				"job_id": "00000000-0000-0000-0000-000000000002",
				"user_id": "11111111-1111-1111-1111-111111111111",
				"match_score": 92,
				"match_reasoning": "Direct match for 6-month internship with pre-placement offer.",
				"inferred_required_yoe": 0,
				"standardized_location": "Hyderabad, India",
				"work_model": "hybrid",
				"is_matched": true,
				"notification_criteria_met": true,
				"salary_min": 40000,
				"salary_max": 40000,
				"salary_currency": "INR",
				"salary_period": "monthly",
				"employment_type": "intern_ppo"
			},
			{
				"job_id": "00000000-0000-0000-0000-000000000003",
				"user_id": "11111111-1111-1111-1111-111111111111",
				"match_score": 75,
				"match_reasoning": "Full-time senior role.",
				"inferred_required_yoe": 5,
				"standardized_location": "San Francisco, CA, USA",
				"work_model": "remote",
				"is_matched": true,
				"notification_criteria_met": false,
				"salary_min": 140000,
				"salary_max": 180000,
				"salary_currency": "USD",
				"salary_period": "yearly",
				"employment_type": "full_time"
			},
			{
				"job_id": "00000000-0000-0000-0000-000000000004",
				"user_id": "11111111-1111-1111-1111-111111111111",
				"match_score": 60,
				"match_reasoning": "Unpaid research internship with open source contribution.",
				"inferred_required_yoe": 0,
				"standardized_location": "Remote (Global)",
				"work_model": "remote",
				"is_matched": true,
				"notification_criteria_met": true,
				"salary_min": null,
				"salary_max": null,
				"salary_currency": null,
				"salary_period": null,
				"employment_type": "intern"
			}
		]
	}`

	service := services.NewGeminiBatchMatchService(nil, "test-api-key")
	parsedResponse, ok := service.ParseAndValidateBatchJSONForTest(sampleJSON)
	if !ok {
		t.Fatalf("expected successful parsing of batch JSON with salary and notification fields")
	}

	if len(parsedResponse.Results) != 4 {
		t.Fatalf("expected 4 parsed results, got %d", len(parsedResponse.Results))
	}

	firstItem := parsedResponse.Results[0]
	if !firstItem.NotificationCriteriaMet {
		t.Fatalf("expected item 0 notification_criteria_met to be true")
	}
	if firstItem.SalaryMin == nil || *firstItem.SalaryMin != 25000 {
		t.Fatalf("expected item 0 salary_min to be 25000")
	}
	if firstItem.SalaryCurrency == nil || *firstItem.SalaryCurrency != "INR" {
		t.Fatalf("expected item 0 salary_currency to be INR")
	}
	if firstItem.SalaryPeriod == nil || *firstItem.SalaryPeriod != "monthly" {
		t.Fatalf("expected item 0 salary_period to be monthly")
	}
	if firstItem.EmploymentType == nil || *firstItem.EmploymentType != "intern" {
		t.Fatalf("expected item 0 employment_type to be intern")
	}

	secondItem := parsedResponse.Results[1]
	if secondItem.EmploymentType == nil || *secondItem.EmploymentType != "intern_ppo" {
		t.Fatalf("expected item 1 employment_type to be intern_ppo")
	}
	if secondItem.SalaryMin == nil || *secondItem.SalaryMin != 40000 {
		t.Fatalf("expected item 1 salary_min to be 40000")
	}

	thirdItem := parsedResponse.Results[2]
	if thirdItem.NotificationCriteriaMet {
		t.Fatalf("expected item 2 notification_criteria_met to be false")
	}
	if thirdItem.SalaryMax == nil || *thirdItem.SalaryMax != 180000 {
		t.Fatalf("expected item 2 salary_max to be 180000")
	}
	if thirdItem.SalaryCurrency == nil || *thirdItem.SalaryCurrency != "USD" {
		t.Fatalf("expected item 2 salary_currency to be USD")
	}
	if thirdItem.SalaryPeriod == nil || *thirdItem.SalaryPeriod != "yearly" {
		t.Fatalf("expected item 2 salary_period to be yearly")
	}

	fourthItem := parsedResponse.Results[3]
	if fourthItem.SalaryMin != nil {
		t.Fatalf("expected item 3 salary_min to be nil for unpaid job, got %v", *fourthItem.SalaryMin)
	}
	if fourthItem.SalaryMax != nil {
		t.Fatalf("expected item 3 salary_max to be nil for unpaid job, got %v", *fourthItem.SalaryMax)
	}
	if fourthItem.SalaryCurrency != nil {
		t.Fatalf("expected item 3 salary_currency to be nil for unpaid job, got %v", *fourthItem.SalaryCurrency)
	}
	if fourthItem.SalaryPeriod != nil {
		t.Fatalf("expected item 3 salary_period to be nil for unpaid job, got %v", *fourthItem.SalaryPeriod)
	}
}
