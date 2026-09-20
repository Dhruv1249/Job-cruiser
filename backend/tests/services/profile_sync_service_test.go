package services_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestFormatProfileJSONGeneratesValidPayload(testingContext *testing.T) {
	candidateProfile := services.CandidateProfileSyncPayload{
		FullName:             "Dhruv Test",
		ProfessionalHeadline: "Systems Engineer & Full-Stack Architect",
		Email:                "dhruv@example.com",
		Phone:                "+1234567890",
		Location:             "Bengaluru, India",
		Country:              "India",
		LinkedInURL:          "https://linkedin.com/in/dhruv",
		GithubURL:            "https://github.com/dhruv",
		PortfolioURL:         "https://dhruv.dev",
		BioSummary:           "Passionate systems engineer.",
		Skills:               []string{"Rust", "Go", "PostgreSQL"},
		Experiences: []services.ProfileExperienceItem{
			{
				Company:    "TechCorp",
				Role:       "Senior Engineer",
				Duration:   "2023 - Present",
				Highlights: "Built microservices; Optimized SQL queries.",
				TechStack:  []string{"Go", "PostgreSQL"},
			},
		},
		Projects: []services.ProfileProjectItem{
			{
				Title:       "KernelOS",
				TechStack:   []string{"Rust", "x86_64"},
				Description: "Lightweight operating system kernel.",
				GithubURL:   "https://github.com/dhruv/kernelos",
			},
		},
		Education: []services.ProfileEducationItem{
			{
				Institution: "State University",
				Degree:      "Bachelor of Technology in Computer Science",
				Year:        "2024",
				Grade:       "8.8 CGPA",
			},
		},
	}

	formattedJSONBytes, formattingError := services.FormatProfileJSON(candidateProfile)
	if formattingError != nil {
		testingContext.Fatalf("FormatProfileJSON returned unexpected error: %v", formattingError)
	}

	var parsedOutput map[string]interface{}
	if unmarshalError := json.Unmarshal(formattedJSONBytes, &parsedOutput); unmarshalError != nil {
		testingContext.Fatalf("formatted JSON failed to parse: %v", unmarshalError)
	}

	if parsedOutput["name"] != "Dhruv Test" {
		testingContext.Fatalf("expected name 'Dhruv Test', got %v", parsedOutput["name"])
	}
	if parsedOutput["professional_headline"] != "Systems Engineer & Full-Stack Architect" {
		testingContext.Fatalf("expected headline 'Systems Engineer & Full-Stack Architect', got %v", parsedOutput["professional_headline"])
	}
	if parsedOutput["synced_by"] != "Job-cruiser" {
		testingContext.Fatalf("expected synced_by 'Job-cruiser', got %v", parsedOutput["synced_by"])
	}
	if parsedOutput["last_synced_at"] == nil || parsedOutput["last_synced_at"] == "" {
		testingContext.Fatalf("expected valid last_synced_at timestamp, got %v", parsedOutput["last_synced_at"])
	}
}

func TestFormatProfileMarkdownGeneratesReadableDocument(testingContext *testing.T) {
	candidateProfile := services.CandidateProfileSyncPayload{
		FullName:             "Dhruv Test",
		ProfessionalHeadline: "Distributed Systems Architect",
		Email:                "dhruv@example.com",
		Skills:               []string{"Go", "Rust", "Postgres"},
		Experiences: []services.ProfileExperienceItem{
			{
				Company:    "CloudCorp",
				Role:       "Lead Backend Engineer",
				Duration:   "2022 - 2024",
				Highlights: "Architected distributed queue.",
				TechStack:  []string{"Go", "Kafka"},
			},
		},
		Projects: []services.ProfileProjectItem{
			{
				Title:       "Overleaf Sync",
				TechStack:   []string{"Go", "Next.js"},
				Description: "Syncs profiles seamlessly.",
			},
		},
	}

	markdownDocument := services.FormatProfileMarkdown(candidateProfile)

	if !strings.Contains(markdownDocument, "# Candidate Profile & Experience Bank") {
		testingContext.Fatalf("markdown missing primary header")
	}
	if !strings.Contains(markdownDocument, "Last Synced") {
		testingContext.Fatalf("markdown missing Last Synced metadata banner")
	}
	if !strings.Contains(markdownDocument, "Dhruv Test") {
		testingContext.Fatalf("markdown missing candidate name")
	}
	if !strings.Contains(markdownDocument, "Distributed Systems Architect") {
		testingContext.Fatalf("markdown missing headline")
	}
	if !strings.Contains(markdownDocument, "CloudCorp") {
		testingContext.Fatalf("markdown missing experience company")
	}
	if !strings.Contains(markdownDocument, "Overleaf Sync") {
		testingContext.Fatalf("markdown missing project title")
	}
}

func TestFormatProfileLaTeXVariablesEscapesSpecialCharacters(testingContext *testing.T) {
	candidateProfile := services.CandidateProfileSyncPayload{
		FullName:             "Dhruv & Co_Special%Name#1",
		ProfessionalHeadline: "R&D Engineer $100k",
		Email:                "dhruv_test%user@example.com",
		Phone:                "+123456",
		Location:             "New York & San Francisco",
		LinkedInURL:          "https://linkedin.com/in/dhruv_profile",
		GithubURL:            "https://github.com/dhruv_user",
		PortfolioURL:         "https://dhruv.dev?ref=portfolio&user=1",
	}

	latexVariablesOutput := services.FormatProfileLaTeXVariables(candidateProfile)

	if !strings.Contains(latexVariablesOutput, `\newcommand{\candidateLastSyncedAt}`) {
		testingContext.Fatalf("LaTeX missing candidateLastSyncedAt macro: %s", latexVariablesOutput)
	}
	if !strings.Contains(latexVariablesOutput, `\newcommand{\candidateName}{Dhruv \& Co\_Special\%Name\#1}`) {
		testingContext.Fatalf("LaTeX variable candidateName not properly escaped: %s", latexVariablesOutput)
	}
	if !strings.Contains(latexVariablesOutput, `\newcommand{\candidateHeadline}{R\&D Engineer \$100k}`) {
		testingContext.Fatalf("LaTeX variable candidateHeadline not properly escaped: %s", latexVariablesOutput)
	}
	if !strings.Contains(latexVariablesOutput, `\newcommand{\candidateEmail}{dhruv\_test\%user@example.com}`) {
		testingContext.Fatalf("LaTeX variable candidateEmail not properly escaped: %s", latexVariablesOutput)
	}
}

func TestMergeProjectTechIntoSkillsDeduplication(testingContext *testing.T) {
	existingSkills := []string{"Go", "Docker"}
	projects := []services.ProfileProjectItem{
		{
			Title:     "P1",
			TechStack: []string{"go", "Rust", "PostgreSQL"},
		},
		{
			Title:     "P2",
			TechStack: []string{"docker", "K8s", "rust"},
		},
	}

	mergedSkills := services.MergeProjectTechIntoSkills(existingSkills, projects)
	if len(mergedSkills) != 5 {
		testingContext.Fatalf("expected 5 unique skills, got %d: %v", len(mergedSkills), mergedSkills)
	}
}
