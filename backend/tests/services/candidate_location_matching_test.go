/*
Package services_test verifies behavior of AI candidate matching prompts and country-level filtering.
*/
package services_test

import (
	"strings"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

/*
TestCandidateProfileBatchPromptContainsCountryNotCity ensures the AI matching prompt includes candidate country
and excludes candidate city or regional location.
*/
func TestCandidateProfileBatchPromptContainsCountryNotCity(t *testing.T) {
	candidateProfiles := []services.UserProfileData{
		{
			UserID:             "b113a51b-19d1-4a49-9c3e-6873f8e03cec",
			Email:              "dhruv@example.com",
			Country:            "India",
			CurrentLocation:    "kangra, HP",
			PreferredLocations: []string{"Global Remote", "India (Remote)", "India (On-site & Hybrid)"},
			WorkModel:          "any",
			ExperienceYears:    1,
			ParsedBio:          "Experienced software engineer working on cloud systems.",
		},
	}

	jobSnippets := []services.JobSnippetData{
		{
			JobID:       "00000000-0000-0000-0000-000000000001",
			Title:       "Backend Engineer",
			Company:     "Acme Corp",
			Location:    "Bengaluru, India",
			Description: "Looking for a Go developer in Bengaluru.",
		},
	}

	renderedPrompt := services.BuildBatchMatchUserContentForTest(candidateProfiles, jobSnippets, 1)

	if !strings.Contains(renderedPrompt, "Candidate Country: India") {
		t.Fatalf("expected prompt to contain 'Candidate Country: India', got:\n%s", renderedPrompt)
	}

	if strings.Contains(renderedPrompt, "kangra, HP") {
		t.Fatalf("expected prompt to omit granular city 'kangra, HP', but it was found in:\n%s", renderedPrompt)
	}

	if strings.Contains(renderedPrompt, "Candidate Current Location:") {
		t.Fatalf("expected prompt to not contain 'Candidate Current Location:', but it was found in:\n%s", renderedPrompt)
	}
}

/*
TestResolveCandidateCountry verifies fallback and trimming logic for country extraction.
*/
func TestResolveCandidateCountry(t *testing.T) {
	testCases := []struct {
		inputCountry     string
		inputLocation    string
		expectedResolved string
	}{
		{
			inputCountry:     "India",
			inputLocation:    "kangra, HP",
			expectedResolved: "India",
		},
		{
			inputCountry:     "  United States  ",
			inputLocation:    "San Francisco, CA",
			expectedResolved: "United States",
		},
		{
			inputCountry:     "",
			inputLocation:    "Bengaluru, Karnataka, India",
			expectedResolved: "India",
		},
		{
			inputCountry:     "",
			inputLocation:    "Austin, TX, USA",
			expectedResolved: "USA",
		},
		{
			inputCountry:     "",
			inputLocation:    "",
			expectedResolved: "Not specified",
		},
	}

	for _, testCase := range testCases {
		actualResolved := services.ResolveCandidateCountryForTest(testCase.inputCountry, testCase.inputLocation)
		if actualResolved != testCase.expectedResolved {
			t.Fatalf("resolveCandidateCountry(%q, %q) = %q; expected %q",
				testCase.inputCountry, testCase.inputLocation, actualResolved, testCase.expectedResolved)
		}
	}
}

/*
TestBatchMatchSystemInstructionEmphasizesCountryLevelMatching verifies that system instructions instruct
the AI to evaluate location strictly at the Country level.
*/
func TestBatchMatchSystemInstructionEmphasizesCountryLevelMatching(t *testing.T) {
	systemInstruction := services.BuildBatchMatchSystemInstructionForTest(1)

	if !strings.Contains(systemInstruction, "Candidate location is evaluated strictly at the Country level") {
		t.Fatalf("expected system instruction to emphasize Country-level matching, got:\n%s", systemInstruction)
	}

	if !strings.Contains(systemInstruction, "Check the candidate's Country") {
		t.Fatalf("expected system instruction to instruct checking candidate's Country, got:\n%s", systemInstruction)
	}
}
