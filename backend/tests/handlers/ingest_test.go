package handlers_test

import (
	"encoding/json"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

func TestExtractTechTagsMatchesKnownKeywords(t *testing.T) {
	testCases := []struct {
		name            string
		title           string
		description     string
		expectedPresent []string
		expectedAbsent  []string
	}{
		{
			name:            "backend engineer with Go and Postgres",
			title:           "Backend Engineer",
			description:     "We use golang, postgres, kubernetes, and redis heavily.",
			expectedPresent: []string{"golang", "postgres", "kubernetes", "redis"},
			expectedAbsent:  []string{"java", "swift"},
		},
		{
			name:            "frontend engineer with React and TypeScript",
			title:           "Frontend Engineer",
			description:     "Strong React and TypeScript skills required. HTML, CSS a plus.",
			expectedPresent: []string{"react", "typescript", "html", "css"},
			expectedAbsent:  []string{"rust", "kafka"},
		},
		{
			name:            "ML platform engineer with Python and TensorFlow",
			title:           "ML Platform Engineer",
			description:     "Experience with Python, TensorFlow, Spark, and Hadoop.",
			expectedPresent: []string{"python", "tensorflow", "spark", "hadoop"},
			expectedAbsent:  []string{"php", "ruby"},
		},
		{
			name:            "empty description and title returns no tags",
			title:           "",
			description:     "",
			expectedPresent: []string{},
			expectedAbsent:  []string{"go", "python", "react"},
		},
		{
			name:            "hyphen-joined compound prevents keyword match",
			title:           "Engineer",
			description:     "Working with golang-adjacent tooling and node-based routing.",
			expectedPresent: []string{},
			expectedAbsent:  []string{"golang", "node"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tags := handlers.ExtractTechTags(tc.title, tc.description)
			tagSet := make(map[string]bool)
			for _, tag := range tags {
				tagSet[tag] = true
			}
			for _, expected := range tc.expectedPresent {
				if !tagSet[expected] {
					t.Errorf("expected tag %q to be present in result %v", expected, tags)
				}
			}
			for _, absent := range tc.expectedAbsent {
				if tagSet[absent] {
					t.Errorf("expected tag %q to be absent from result %v", absent, tags)
				}
			}
		})
	}
}

func TestExtractExperienceRangeFormat(t *testing.T) {
	testCases := []struct {
		name        string
		title       string
		description string
		expected    string
	}{
		{
			name:        "standard range 3-5 years",
			title:       "Backend Engineer",
			description: "We require 3-5 years of relevant experience.",
			expected:    "3-5 years",
		},
		{
			name:        "range with 'to' separator",
			title:       "SDE",
			description: "Candidates with 2 to 4 years of experience preferred.",
			expected:    "2-4 years",
		},
		{
			name:        "plus format 5+ years",
			title:       "Staff Engineer",
			description: "5+ years of software engineering experience required.",
			expected:    "5+ years",
		},
		{
			name:        "minimum prefix at least 3 years",
			title:       "Senior Engineer",
			description: "At least 3 years working with distributed systems.",
			expected:    "3+ years",
		},
		{
			name:        "minimum prefix minimum of 2 years",
			title:       "Junior Dev",
			description: "Minimum of 2 years of professional coding experience.",
			expected:    "2+ years",
		},
		{
			name:        "simple suffix years of experience",
			title:       "Cloud Engineer",
			description: "4 years of experience with cloud platforms.",
			expected:    "4+ years",
		},
		{
			name:        "no experience mentioned returns empty string",
			title:       "Intern",
			description: "No prior experience needed — we will train you.",
			expected:    "",
		},
		{
			name:        "experience in title takes priority when no description match",
			title:       "7+ years Backend Engineer",
			description: "Come join our team.",
			expected:    "7+ years",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handlers.ExtractExperience(tc.title, tc.description)
			if result != tc.expected {
				t.Errorf("ExtractExperience(%q, %q): expected %q, got %q",
					tc.title, tc.description, tc.expected, result)
			}
		})
	}
}

func TestContainsWordBehavior(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		word     string
		expected bool
	}{
		{
			name:     "exact standalone word matches",
			text:     "we use go for our services",
			word:     "go",
			expected: true,
		},
		{
			name:     "word inside larger word does not match",
			text:     "we use golang not go-lang",
			word:     "go",
			expected: false,
		},
		{
			name:     "word at start of string matches",
			text:     "python is required",
			word:     "python",
			expected: true,
		},
		{
			name:     "word at end of string matches",
			text:     "primary language is rust",
			word:     "rust",
			expected: true,
		},
		{
			name:     "hyphen before word prevents match because hyphen is alphanumeric",
			text:     "node-based architecture",
			word:     "node",
			expected: false,
		},
		{
			name:     "empty text never matches",
			text:     "",
			word:     "go",
			expected: false,
		},
		{
			name:     "c++ matches with special character boundary",
			text:     "proficient in c++ and c#",
			word:     "c++",
			expected: true,
		},
		{
			name:     "word with forward slash matches",
			text:     "ci/cd pipeline experience required",
			word:     "ci/cd",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handlers.ContainsWord(tc.text, tc.word)
			if result != tc.expected {
				t.Errorf("ContainsWord(%q, %q): expected %v, got %v",
					tc.text, tc.word, tc.expected, result)
			}
		})
	}
}

func TestIsAlphanumericClassifiesCorrectly(t *testing.T) {
	alphanumericChars := []byte{'a', 'z', '0', '9', '+', '#', '/', '-'}
	nonAlphanumericChars := []byte{' ', '.', ',', '!', '(', ')', '_'}

	for _, ch := range alphanumericChars {
		if !handlers.IsAlphanumeric(ch) {
			t.Errorf("expected character %q to be classified as alphanumeric", ch)
		}
	}

	for _, ch := range nonAlphanumericChars {
		if handlers.IsAlphanumeric(ch) {
			t.Errorf("expected character %q to be classified as non-alphanumeric", ch)
		}
	}
}

func TestEnrichJobDescriptionsPayloadSerialization(t *testing.T) {
	requestPayload := handlers.EnrichJobDescriptionsRequest{
		Updates: []handlers.EnrichJobDescriptionItem{
			{
				ID:              "job-uuid-123",
				DescriptionText: "Experienced backend engineer required.",
			},
		},
	}

	if len(requestPayload.Updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(requestPayload.Updates))
	}
	if requestPayload.Updates[0].ID != "job-uuid-123" {
		t.Errorf("expected ID 'job-uuid-123', got %s", requestPayload.Updates[0].ID)
	}
	if requestPayload.Updates[0].DescriptionText != "Experienced backend engineer required." {
		t.Errorf("expected DescriptionText match, got %s", requestPayload.Updates[0].DescriptionText)
	}
}

func TestJobWithoutDescriptionItemSerialization(t *testing.T) {
	item := handlers.JobWithoutDescriptionItem{
		ID:    "job-uuid-456",
		URL:   "https://www.linkedin.com/jobs/view/456",
		Title: "Staff Software Engineer",
	}

	if item.ID != "job-uuid-456" {
		t.Errorf("expected ID 'job-uuid-456', got %s", item.ID)
	}
	if item.URL != "https://www.linkedin.com/jobs/view/456" {
		t.Errorf("expected URL match, got %s", item.URL)
	}
	if item.Title != "Staff Software Engineer" {
		t.Errorf("expected Title match, got %s", item.Title)
	}
}

func TestFinishRequestSourceDistributionSerialization(t *testing.T) {
	rawJSON := []byte(`{
		"run_id": "run-xyz-789",
		"status": "success",
		"error_message": "",
		"sources_hit": {
			"greenhouse": 120,
			"lever": 85,
			"ashby": 45
		}
	}`)

	var finishRequest handlers.FinishRequest
	unmarshalErr := json.Unmarshal(rawJSON, &finishRequest)
	if unmarshalErr != nil {
		t.Fatalf("unexpected unmarshal error: %v", unmarshalErr)
	}

	if finishRequest.RunID != "run-xyz-789" {
		t.Errorf("expected run_id 'run-xyz-789', got %s", finishRequest.RunID)
	}
	if finishRequest.Status != "success" {
		t.Errorf("expected status 'success', got %s", finishRequest.Status)
	}
	if len(finishRequest.SourcesHit) == 0 {
		t.Fatalf("expected sources_hit to not be empty")
	}

	var sourceDistributionMap map[string]int
	distributionErr := json.Unmarshal(finishRequest.SourcesHit, &sourceDistributionMap)
	if distributionErr != nil {
		t.Fatalf("unexpected sources_hit map unmarshal error: %v", distributionErr)
	}

	if sourceDistributionMap["greenhouse"] != 120 {
		t.Errorf("expected greenhouse count 120, got %d", sourceDistributionMap["greenhouse"])
	}
	if sourceDistributionMap["lever"] != 85 {
		t.Errorf("expected lever count 85, got %d", sourceDistributionMap["lever"])
	}
	if sourceDistributionMap["ashby"] != 45 {
		t.Errorf("expected ashby count 45, got %d", sourceDistributionMap["ashby"])
	}
}

func TestFinishRequestGranularSourcesAggregation(t *testing.T) {
	rawJSON := []byte(`{
		"run_id": "run-telemetry-456",
		"status": "success",
		"sources_hit": {
			"linkedin:sde i:india": {"jobs_found": 200, "duration_seconds": 15.0},
			"linkedin:frontend:india": {"jobs_found": 100, "duration_seconds": 10.0},
			"indeed:backend:remote": {"jobs_found": 50, "duration_seconds": 4.5}
		}
	}`)

	var finishRequest handlers.FinishRequest
	unmarshalErr := json.Unmarshal(rawJSON, &finishRequest)
	if unmarshalErr != nil {
		t.Fatalf("unexpected unmarshal error: %v", unmarshalErr)
	}

	aggregatedPayload, aggErr := utils.AggregateSourceStatistics(finishRequest.SourcesHit)
	if aggErr != nil {
		t.Fatalf("unexpected aggregation error: %v", aggErr)
	}

	var parsedMetrics map[string]utils.ScraperPlatformMetric
	if parseErr := json.Unmarshal(aggregatedPayload, &parsedMetrics); parseErr != nil {
		t.Fatalf("unexpected metric parse error: %v", parseErr)
	}

	if parsedMetrics["linkedin"].JobsFound != 300 {
		t.Errorf("expected 300 jobs for linkedin, got %d", parsedMetrics["linkedin"].JobsFound)
	}
	if parsedMetrics["linkedin"].QueryCount != 2 {
		t.Errorf("expected query count 2 for linkedin, got %d", parsedMetrics["linkedin"].QueryCount)
	}
	if parsedMetrics["indeed"].JobsFound != 50 {
		t.Errorf("expected 50 jobs for indeed, got %d", parsedMetrics["indeed"].JobsFound)
	}
}

