package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
)

/*
TestMatchedJobResponseSerialization verifies that MatchedJobResponse includes scraped_at field when marshaled to JSON.
*/
func TestMatchedJobResponseSerialization(t *testing.T) {
	response := handlers.MatchedJobResponse{
		JobID:             "test-job-uuid",
		Title:             "Senior Backend Engineer",
		Company:           "Acme Corp",
		Location:          "Remote",
		IsRemote:          true,
		Source:            "Greenhouse",
		URL:               "https://example.com/job/123",
		PostedDate:        "2026-08-10",
		ScrapedAt:         "2026-08-12 10:00:00",
		Seniority:         "Senior",
		Summary:           "Great Go role",
		RawDescription:    "Full Job Description",
		MatchScore:        85,
		MatchReasoning:    "Matches Go and backend skills",
		TechStack:         []string{"Go", "PostgreSQL"},
		IsMatched:         true,
		Currency:          "USD",
		IsViewed:          false,
		ApplicationStatus: "unapplied",
		IsNew:             true,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal MatchedJobResponse: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	scrapedAtValue, exists := parsed["scraped_at"]
	if !exists {
		t.Fatalf("expected scraped_at field in JSON output")
	}

	if scrapedAtValue != "2026-08-12 10:00:00" {
		t.Errorf("expected scraped_at '2026-08-12 10:00:00', got '%v'", scrapedAtValue)
	}
}

/*
TestMatchedJobResponseUnmarshaling verifies roundtrip unmarshaling of MatchedJobResponse fields.
*/
func TestMatchedJobResponseUnmarshaling(t *testing.T) {
	rawJSON := []byte(`{
		"job_id": "job-abc-456",
		"title": "Staff Infrastructure Engineer",
		"company": "CloudScale Inc",
		"location": "San Francisco, CA",
		"is_remote": false,
		"source": "Lever",
		"url": "https://example.com/job/456",
		"posted_date": "2026-08-26",
		"scraped_at": "2026-08-27 08:30:00",
		"seniority": "Staff",
		"summary": "Kubernetes and platform infrastructure role",
		"raw_description": "Managing multi-region Kubernetes clusters.",
		"match_score": 94,
		"match_reasoning": "Deep experience in Kubernetes and Go",
		"tech_stack": ["Kubernetes", "Go", "Terraform"],
		"is_matched": true,
		"salary_min": 210000,
		"salary_max": 250000,
		"currency": "USD",
		"is_viewed": true,
		"application_status": "applied",
		"is_new": false
	}`)

	var parsedResponse handlers.MatchedJobResponse
	unmarshalError := json.Unmarshal(rawJSON, &parsedResponse)
	if unmarshalError != nil {
		t.Fatalf("failed to unmarshal JSON into MatchedJobResponse: %v", unmarshalError)
	}

	if parsedResponse.JobID != "job-abc-456" {
		t.Errorf("expected JobID 'job-abc-456', got '%s'", parsedResponse.JobID)
	}
	if parsedResponse.MatchScore != 94 {
		t.Errorf("expected MatchScore 94, got %d", parsedResponse.MatchScore)
	}
	if !parsedResponse.IsViewed {
		t.Errorf("expected IsViewed to be true")
	}
	if len(parsedResponse.TechStack) != 3 {
		t.Errorf("expected 3 items in TechStack, got %d", len(parsedResponse.TechStack))
	}
}
/*
TestUnmarshalStringJSONValidArray verifies that a valid JSON array string is correctly decoded into a Go string slice.
*/
func TestUnmarshalStringJSONValidArray(t *testing.T) {
	raw := `["Go","PostgreSQL","Docker"]`
	var result []string
	if err := handlers.UnmarshalStringJSON(raw, &result); err != nil {
		t.Fatalf("expected no error for valid JSON array, got: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != "Go" || result[1] != "PostgreSQL" || result[2] != "Docker" {
		t.Errorf("unexpected values: %v", result)
	}
}

/*
TestUnmarshalStringJSONEmptyArray verifies that an empty JSON array is decoded to an empty slice without error.
*/
func TestUnmarshalStringJSONEmptyArray(t *testing.T) {
	raw := `[]`
	var result []string
	if err := handlers.UnmarshalStringJSON(raw, &result); err != nil {
		t.Fatalf("expected no error for empty JSON array, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 items, got %d", len(result))
	}
}

/*
TestUnmarshalStringJSONInvalidInput verifies that malformed JSON returns a non-nil error.
*/
func TestUnmarshalStringJSONInvalidInput(t *testing.T) {
	raw := `not valid json`
	var result []string
	if err := handlers.UnmarshalStringJSON(raw, &result); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

/*
TestMatchedJobResponseApplicationStatusField verifies that the application_status field is correctly serialized.
*/
func TestMatchedJobResponseApplicationStatusField(t *testing.T) {
	statuses := []string{"unapplied", "bookmarked", "applied", "interviewing", "offer", "rejected"}
	for _, status := range statuses {
		response := handlers.MatchedJobResponse{
			JobID:             "test-id",
			ApplicationStatus: status,
			TechStack:         []string{},
		}
		jsonBytes, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("failed to marshal for status %q: %v", status, err)
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("failed to unmarshal for status %q: %v", status, err)
		}
		got, ok := parsed["application_status"]
		if !ok {
			t.Errorf("application_status field missing for status %q", status)
			continue
		}
		if got != status {
			t.Errorf("expected application_status %q, got %q", status, got)
		}
	}
}
