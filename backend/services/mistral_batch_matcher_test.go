package services

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChunkJobSnippets(t *testing.T) {
	snippets := []JobSnippet{
		{JobID: "1"}, {JobID: "2"}, {JobID: "3"}, {JobID: "4"}, {JobID: "5"},
	}

	batches := chunkJobSnippets(snippets, 2)

	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	if len(batches[0]) != 2 || len(batches[1]) != 2 || len(batches[2]) != 1 {
		t.Errorf("unexpected batch sizes: %v, %v, %v", len(batches[0]), len(batches[1]), len(batches[2]))
	}
}

func TestBuildMatchPrompt(t *testing.T) {
	service := &MistralBatchMatchService{}
	profile := UserProfile{
		UserID:           "user-1",
		ParsedExperience: "2 years Go experience",
		TargetRoles:      "Backend Engineer",
		WorkModels:       "Remote",
		MinSalary:        100000,
		Currency:         "USD",
	}
	snippets := []JobSnippet{
		{JobID: "job-1", Title: "Junior Go Engineer", Company: "Acme", Location: "Remote", Description: "Go role"},
	}

	prompt := service.buildMatchPrompt(profile, snippets)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !containsString(prompt, "2 years Go experience") {
		t.Errorf("prompt missing experience context")
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

		resp := mistralAPIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: `{"results": [{"job_id": "job-1", "is_matched": true, "match_score": 85, "seniority": "Junior", "tech_stack": ["Go"], "reasoning": "Fits candidate profile"}]}`,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	service := &MistralBatchMatchService{
		MistralKey: "test-key",
		HTTPClient: mockServer.Client(),
	}

	profile := UserProfile{UserID: "user-1", ParsedExperience: "Go dev"}
	snippets := []JobSnippet{{JobID: "job-1", Title: "Junior Go Dev"}}

	prompt := service.buildMatchPrompt(profile, snippets)
	reqPayload := mistralRequest{
		Model: mistralMatchModel,
		Messages: []mistralRequestMessage{
			{Role: "user", Content: prompt},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0.0,
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

	var apiResp mistralAPIResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)

	var matchResp mistralBatchResponse
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
