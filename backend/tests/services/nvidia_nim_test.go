// Package services_test contains unit tests for backend services.
package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

// TestNvidiaNimQueuePauseResume verifies that PauseQueue and ResumeQueue update the queue state cleanly.
func TestNvidiaNimQueuePauseResume(t *testing.T) {
	service := services.NewNvidiaNimService(nil, "test-nvidia-key")

	if service.IsQueuePaused() {
		t.Fatalf("expected queue to be unpaused initially")
	}

	service.PauseQueue()
	if !service.IsQueuePaused() {
		t.Fatalf("expected queue to be paused after calling PauseQueue")
	}

	service.ResumeQueue()
	if service.IsQueuePaused() {
		t.Fatalf("expected queue to be unpaused after calling ResumeQueue")
	}
}

// TestNvidiaNimGenerateCompletionSuccess verifies HTTP headers, payload, and response extraction targeting NVIDIA NIM.
func TestNvidiaNimGenerateCompletionSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-nvidia-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		responsePayload := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Tailored resume bullet generated successfully",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responsePayload)
	}))
	defer mockServer.Close()

	service := services.NewNvidiaNimService(nil, "test-nvidia-key")
	service.Endpoint = mockServer.URL
	service.HTTPClient = mockServer.Client()

	completionResult, errCall := service.GenerateCompletion(context.Background(), "Summarize experience", "You are an expert ATS writer")
	if errCall != nil {
		t.Fatalf("expected successful completion, got error: %v", errCall)
	}

	if completionResult != "Tailored resume bullet generated successfully" {
		t.Errorf("unexpected completion content: %s", completionResult)
	}
}

// TestNvidiaNimRateLimitRetry verifies exponential retry handling upon receiving HTTP 429 response.
func TestNvidiaNimRateLimitRetry(t *testing.T) {
	attemptCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		responsePayload := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Success after rate limit retry",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responsePayload)
	}))
	defer mockServer.Close()

	service := services.NewNvidiaNimService(nil, "test-nvidia-key")
	service.Endpoint = mockServer.URL
	service.HTTPClient = mockServer.Client()

	completionResult, errCall := service.GenerateCompletion(context.Background(), "Test prompt", "")
	if errCall != nil {
		t.Fatalf("expected successful recovery after retry, got error: %v", errCall)
	}

	if attemptCount != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", attemptCount)
	}

	if completionResult != "Success after rate limit retry" {
		t.Errorf("unexpected output: %s", completionResult)
	}
}

// TestNvidiaNimCountTokensAPI verifies exact token counting via the NVIDIA NIM tokenize API endpoint.
func TestNvidiaNimCountTokensAPI(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responsePayload := map[string]interface{}{
			"count": 42,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responsePayload)
	}))
	defer mockServer.Close()

	service := services.NewNvidiaNimService(nil, "test-nvidia-key")
	service.Endpoint = mockServer.URL + "/chat/completions"
	service.HTTPClient = mockServer.Client()

	tokenCount, errCount := service.CountTokens(context.Background(), "Sample prompt text for token counting")
	if errCount != nil {
		t.Fatalf("expected successful token count, got error: %v", errCount)
	}

	if tokenCount != 42 {
		t.Errorf("expected exact token count 42, got %d", tokenCount)
	}
}

// TestIsValidUUIDValidation verifies that valid 36-char UUIDs pass and malformed LLM UUIDs are rejected.
func TestIsValidUUIDValidation(t *testing.T) {
	validUUID := "96b83816-1389-4399-0ff5-cf5f135249b0"
	invalidShortUUID := "96b83816-1389-4399-ff5-cf5f135249b0"

	if !services.IsValidUUIDStringForTest(validUUID) {
		t.Errorf("expected valid UUID %s to pass validation", validUUID)
	}
	if services.IsValidUUIDStringForTest(invalidShortUUID) {
		t.Errorf("expected truncated UUID %s to fail validation", invalidShortUUID)
	}
}

// TestNvidiaNimGenerateCompletionWithJSONSchema verifies that guided_json schema is cleanly passed in payload.nvext.
func TestNvidiaNimGenerateCompletionWithJSONSchema(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestPayload map[string]interface{}
		if errDecode := json.NewDecoder(r.Body).Decode(&requestPayload); errDecode != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}

		nvExtMap, exists := requestPayload["nvext"].(map[string]interface{})
		if !exists {
			http.Error(w, "missing nvext in payload", http.StatusBadRequest)
			return
		}

		guidedJSON, hasJSON := nvExtMap["guided_json"].(map[string]interface{})
		if !hasJSON || guidedJSON["type"] != "object" {
			http.Error(w, "missing or invalid guided_json in nvext payload", http.StatusBadRequest)
			return
		}

		responsePayload := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{"title":"Inception","rating":4.0}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responsePayload)
	}))
	defer mockServer.Close()

	service := services.NewNvidiaNimService(nil, "test-nvidia-key")
	service.Endpoint = mockServer.URL
	service.HTTPClient = mockServer.Client()

	movieSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":  map[string]any{"type": "string"},
			"rating": map[string]any{"type": "number"},
		},
		"required": []string{"title", "rating"},
	}

	result, errCall := service.GenerateCompletionWithSchema(context.Background(), "Rate Inception", "", movieSchema)
	if errCall != nil {
		t.Fatalf("expected successful schema completion, got error: %v", errCall)
	}

	if result != `{"title":"Inception","rating":4.0}` {
		t.Errorf("unexpected content: %s", result)
	}
}

// TestNvidiaNimGenerateCompletionWithNvExt verifies guided_regex transmission in nvext payload.
func TestNvidiaNimGenerateCompletionWithNvExt(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestPayload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestPayload)

		nvExtMap, exists := requestPayload["nvext"].(map[string]interface{})
		if !exists {
			http.Error(w, "missing nvext", http.StatusBadRequest)
			return
		}

		regex, _ := nvExtMap["guided_regex"].(string)
		if regex != "[1-5]" {
			http.Error(w, "invalid guided_regex", http.StatusBadRequest)
			return
		}

		responsePayload := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "4",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responsePayload)
	}))
	defer mockServer.Close()

	service := services.NewNvidiaNimService(nil, "test-nvidia-key")
	service.Endpoint = mockServer.URL
	service.HTTPClient = mockServer.Client()

	nvExt := &services.NvidiaNvExt{
		GuidedRegex: "[1-5]",
	}

	result, errCall := service.GenerateCompletionWithNvExt(context.Background(), "Rate 1-5", "", nvExt)
	if errCall != nil {
		t.Fatalf("expected successful guided_regex completion, got error: %v", errCall)
	}

	if result != "4" {
		t.Errorf("unexpected result: %s", result)
	}
}

// TestSanitizeJSONResponseWithPreambleText verifies that preamble reasoning text starting with 'L' is cleanly trimmed before parsing JSON.
func TestSanitizeJSONResponseWithPreambleText(t *testing.T) {
	rawLLMOutput := "Looking at the candidate profiles and job listings, here is the structured evaluation:\n\n" +
		`{"results":[{"job_id":"96b83816-1389-4399-0ff5-cf5f135249b0","user_id":"96b83816-1389-4399-0ff5-cf5f135249b0","match_score":85,"match_reasoning":"Good match","is_matched":true}]}`

	sanitized := services.SanitizeJSONResponseForTest(rawLLMOutput)
	expectedPrefix := "{\"results\":"
	if !strings.HasPrefix(sanitized, expectedPrefix) {
		t.Fatalf("expected sanitized JSON to start with %s, got: %s", expectedPrefix, sanitized)
	}

	var parsed map[string]interface{}
	if errUnmarshal := json.Unmarshal([]byte(sanitized), &parsed); errUnmarshal != nil {
		t.Fatalf("expected clean unmarshal of sanitized JSON, got error: %v", errUnmarshal)
	}
}
