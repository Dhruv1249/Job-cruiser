package utils

import (
	"testing"
)

func TestExtractCompanyName(t *testing.T) {
	tests := []struct {
		company  string
		url      string
		title    string
		expected string
	}{
		{
			company:  "Stripe",
			url:      "https://boards.greenhouse.io/stripe/jobs/12345",
			title:    "Software Engineer",
			expected: "Stripe",
		},
		{
			company:  "Unknown",
			url:      "https://boards.greenhouse.io/figma/jobs/67890",
			title:    "Product Designer",
			expected: "Figma",
		},
		{
			company:  "",
			url:      "https://jobs.lever.co/notion/abc123",
			title:    "Fullstack Engineer",
			expected: "Notion",
		},
		{
			company:  "Unknown",
			url:      "https://jobs.ashbyhq.com/linear/def456",
			title:    "Frontend Engineer",
			expected: "Linear",
		},
		{
			company:  "Unknown",
			url:      "https://dover.myworkdaysite.com/recruiting/dover/jobs",
			title:    "Systems Developer",
			expected: "Dover",
		},
		{
			company:  "",
			url:      "https://careers.proxify.io/jobs/123",
			title:    "Senior Engineer at Proxify",
			expected: "Proxify",
		},
	}

	for _, tt := range tests {
		result := ExtractCompanyName(tt.company, tt.url, tt.title)
		if result != tt.expected {
			t.Errorf("ExtractCompanyName(%q, %q, %q) = %q; want %q", tt.company, tt.url, tt.title, result, tt.expected)
		}
	}
}

func TestExtractCompanyDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			url:      "https://careers.google.com/jobs/results/123",
			expected: "google.com",
		},
		{
			url:      "https://proxify.io/careers/123",
			expected: "proxify.io",
		},
	}

	for _, tt := range tests {
		result := ExtractCompanyDomain(tt.url)
		if result != tt.expected {
			t.Errorf("ExtractCompanyDomain(%q) = %q; want %q", tt.url, result, tt.expected)
		}
	}
}
