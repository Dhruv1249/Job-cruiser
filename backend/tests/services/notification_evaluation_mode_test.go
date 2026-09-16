/*
Package services_test provides comprehensive unit tests for notification evaluation modes and criteria filtering.
*/
package services_test

import (
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

/*
TestEvaluateNotificationEligibility verifies all four notification evaluation modes across truth tables and edge cases.
*/
func TestEvaluateNotificationEligibility(t *testing.T) {
	testCases := []struct {
		name                    string
		evaluationMode          string
		matchScore              int
		targetThreshold         int
		criteriaPrompt          string
		notificationCriteriaMet bool
		expectedResult          bool
	}{
		{
			name:                    "score_only mode passes when score meets threshold regardless of prompt",
			evaluationMode:          "score_only",
			matchScore:              85,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: false,
			expectedResult:          true,
		},
		{
			name:                    "score_only mode fails when score is below threshold",
			evaluationMode:          "score_only",
			matchScore:              75,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: true,
			expectedResult:          false,
		},
		{
			name:                    "score_only mode passes with empty prompt when score meets threshold",
			evaluationMode:          "score_only",
			matchScore:              80,
			targetThreshold:         80,
			criteriaPrompt:          "",
			notificationCriteriaMet: false,
			expectedResult:          true,
		},
		{
			name:                    "prompt_only mode passes when criteria met regardless of low score",
			evaluationMode:          "prompt_only",
			matchScore:              40,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: true,
			expectedResult:          true,
		},
		{
			name:                    "prompt_only mode fails when criteria not met even with high score",
			evaluationMode:          "prompt_only",
			matchScore:              95,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: false,
			expectedResult:          false,
		},
		{
			name:                    "prompt_only mode strictly fails when criteria prompt is empty with no fallback",
			evaluationMode:          "prompt_only",
			matchScore:              95,
			targetThreshold:         80,
			criteriaPrompt:          "   ",
			notificationCriteriaMet: true,
			expectedResult:          false,
		},
		{
			name:                    "both mode passes when score meets threshold and criteria met",
			evaluationMode:          "both",
			matchScore:              85,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: true,
			expectedResult:          true,
		},
		{
			name:                    "both mode fails when score below threshold even if criteria met",
			evaluationMode:          "both",
			matchScore:              75,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: true,
			expectedResult:          false,
		},
		{
			name:                    "both mode fails when score meets threshold but criteria not met",
			evaluationMode:          "both",
			matchScore:              90,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: false,
			expectedResult:          false,
		},
		{
			name:                    "both mode falls back to score threshold when criteria prompt is empty",
			evaluationMode:          "both",
			matchScore:              80,
			targetThreshold:         80,
			criteriaPrompt:          "",
			notificationCriteriaMet: false,
			expectedResult:          true,
		},
		{
			name:                    "both mode fails when criteria prompt is empty and score below threshold",
			evaluationMode:          "both",
			matchScore:              79,
			targetThreshold:         80,
			criteriaPrompt:          "",
			notificationCriteriaMet: true,
			expectedResult:          false,
		},
		{
			name:                    "either mode passes when score meets threshold even if criteria not met",
			evaluationMode:          "either",
			matchScore:              85,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: false,
			expectedResult:          true,
		},
		{
			name:                    "either mode passes when criteria met even if score below threshold",
			evaluationMode:          "either",
			matchScore:              50,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: true,
			expectedResult:          true,
		},
		{
			name:                    "either mode fails when neither score nor criteria met",
			evaluationMode:          "either",
			matchScore:              60,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: false,
			expectedResult:          false,
		},
		{
			name:                    "either mode with empty prompt relies on score threshold passing",
			evaluationMode:          "either",
			matchScore:              82,
			targetThreshold:         80,
			criteriaPrompt:          "",
			notificationCriteriaMet: false,
			expectedResult:          true,
		},
		{
			name:                    "either mode with empty prompt relies on score threshold failing",
			evaluationMode:          "either",
			matchScore:              70,
			targetThreshold:         80,
			criteriaPrompt:          "",
			notificationCriteriaMet: true,
			expectedResult:          false,
		},
		{
			name:                    "unknown mode defaults to both mode behavior",
			evaluationMode:          "invalid_mode_name",
			matchScore:              85,
			targetThreshold:         80,
			criteriaPrompt:          "Only remote roles",
			notificationCriteriaMet: true,
			expectedResult:          true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualResult := services.EvaluateNotificationEligibility(
				testCase.evaluationMode,
				testCase.matchScore,
				testCase.targetThreshold,
				testCase.criteriaPrompt,
				testCase.notificationCriteriaMet,
			)

			if actualResult != testCase.expectedResult {
				t.Fatalf("expected eligibility %v, got %v for mode %s, score %d, threshold %d, prompt '%s', criteriaMet %v",
					testCase.expectedResult,
					actualResult,
					testCase.evaluationMode,
					testCase.matchScore,
					testCase.targetThreshold,
					testCase.criteriaPrompt,
					testCase.notificationCriteriaMet,
				)
			}
		})
	}
}
