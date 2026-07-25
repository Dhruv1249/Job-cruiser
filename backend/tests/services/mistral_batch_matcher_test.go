package services_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestChunkJobSnippets(t *testing.T) {
	snippets := []services.JobSnippet{
		{JobID: "1"}, {JobID: "2"}, {JobID: "3"}, {JobID: "4"}, {JobID: "5"},
	}

	batches := services.ChunkJobSnippets(snippets, 2)

	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	if len(batches[0]) != 2 || len(batches[1]) != 2 || len(batches[2]) != 1 {
		t.Errorf("unexpected batch sizes: %v, %v, %v", len(batches[0]), len(batches[1]), len(batches[2]))
	}
}

func TestBuildMatchPrompt(t *testing.T) {
	service := &services.MistralBatchMatchService{}
	profile := services.UserProfile{
		UserID:           "user-1",
		ParsedExperience: "2 years Go experience",
		TargetRoles:      "Backend Engineer",
		WorkModels:       "Remote",
		MinSalary:        100000,
		Currency:         "USD",
	}
	snippets := []services.JobSnippet{
		{JobID: "job-1", Title: "Junior Go Engineer", Company: "Acme", Location: "Remote", Description: "Go role"},
	}

	prompt := service.BuildMatchPrompt(profile, snippets)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !containsString(prompt, "user-1") {
		t.Errorf("prompt missing user ID")
	}
	if !containsString(prompt, "job-1") {
		t.Errorf("prompt missing job snippet ID")
	}
}

func TestCallMistralBatchSuccess(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{"results": [{"job_id": "job-1", "is_matched": true, "match_score": 85, "seniority": "Junior", "tech_stack": ["Go"], "reasoning": "Fits candidate profile"}]}`,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	service := &services.MistralBatchMatchService{
		MistralKey: "test-key",
		HTTPClient: mockServer.Client(),
	}

	profile := services.UserProfile{UserID: "user-1", ParsedExperience: "Go dev"}
	snippets := []services.JobSnippet{{JobID: "job-1", Title: "Junior Go Dev"}}

	prompt := service.BuildMatchPrompt(profile, snippets)
	reqPayload := map[string]interface{}{
		"model": "mistral-small-2506",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.0,
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, mockServer.URL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	req.Header.Set("Content-Type", "application/json")

	resp, err := mockServer.Client().Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&apiResp)

	var matchResp struct {
		Results []struct {
			JobID      string `json:"job_id"`
			MatchScore int    `json:"match_score"`
		} `json:"results"`
	}
	json.Unmarshal([]byte(apiResp.Choices[0].Message.Content), &matchResp)

	if len(matchResp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(matchResp.Results))
	}
	if matchResp.Results[0].MatchScore != 85 {
		t.Errorf("expected score 85, got %d", matchResp.Results[0].MatchScore)
	}
}

func containsString(s, substr string) bool {
	return searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
