// Package services_test contains unit tests for backend services.
package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
