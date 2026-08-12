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
