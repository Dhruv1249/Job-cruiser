/*
Package services_test provides unit tests for notification criteria guardrails and deterministic alert filters.
*/
package services_test

import (
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

/*
TestEvaluateNotificationCriteriaGuardrails validates deterministic guardrail rules against candidate criteria prompts.
*/
func TestEvaluateNotificationCriteriaGuardrails(t *testing.T) {
	strictCandidatePrompt := `Only notify if the job satisfies ALL of the following:
1. Candidate Match: Strong alignment with my profile, skills, and experience level.
2. Role Type: Must be a direct Full-Time job OR an Internship with explicit PPO (full-time conversion). Standalone internships without PPO are disqualified.
3. Compensation: If stated, reject internship < ₹30,000/month or full-time role < 10 LPA.`

	testCases := []struct {
		name                   string
		criteriaPrompt         string
		jobTitle               string
		employmentType         string
		salaryMin              int
		salaryMax              int
		salaryPeriod           string
		salaryCurrency         string
		rawDescription         string
		expectedGuardrailPass bool
	}{
		{
			name:                   "rejects standalone internship without PPO",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Software Developer Intern",
			employmentType:         "intern",
			salaryMin:              0,
			salaryMax:              0,
			salaryPeriod:           "",
			salaryCurrency:         "",
			rawDescription:         "Join our engineering team for a summer software internship building cloud systems.",
			expectedGuardrailPass: false,
		},
		{
			name:                   "rejects internship with stipend below threshold even with PPO mention in description",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Web Development Intern",
			employmentType:         "intern",
			salaryMin:              0,
			salaryMax:              0,
			salaryPeriod:           "",
			salaryCurrency:         "",
			rawDescription:         "Duration: 3 months. Stipend: Up to ₹10,000 (performance-based). Pre-placement offer (PPO) available for top performers.",
			expectedGuardrailPass: false,
		},
		{
			name:                   "rejects internship when employment_type is intern and description has no conversion guarantee",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Web Development Intern",
			employmentType:         "intern",
			salaryMin:              0,
			salaryMax:              0,
			salaryPeriod:           "",
			salaryCurrency:         "",
			rawDescription:         "Convert design layouts into functional and user-friendly web interfaces. Stipend: Up to ₹10,000.",
			expectedGuardrailPass: false,
		},
		{
			name:                   "accepts internship with explicit PPO and sufficient stipend in description",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Software Engineering Intern",
			employmentType:         "intern_ppo",
			salaryMin:              40000,
			salaryMax:              50000,
			salaryPeriod:           "monthly",
			salaryCurrency:         "INR",
			rawDescription:         "High performance internship with direct pre-placement offer conversion after 6 months. Stipend ₹45,000/month.",
			expectedGuardrailPass: true,
		},
		{
			name:                   "accepts direct full-time role when compensation is unstated",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Software Engineer I",
			employmentType:         "full_time",
			salaryMin:              0,
			salaryMax:              0,
			salaryPeriod:           "",
			salaryCurrency:         "",
			rawDescription:         "Looking for an entry level software engineer with Golang and distributed systems knowledge.",
			expectedGuardrailPass: true,
		},
		{
			name:                   "rejects full-time role when stated salary in description is below 10 LPA",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Associate Software Engineer",
			employmentType:         "full_time",
			salaryMin:              0,
			salaryMax:              0,
			salaryPeriod:           "",
			salaryCurrency:         "",
			rawDescription:         "Package: 3.5 to 5 LPA. Location: Bengaluru. Immediate joiners preferred.",
			expectedGuardrailPass: false,
		},
		{
			name:                   "rejects full-time role when structured salaryMax is below 10 LPA",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Junior Full Stack Developer",
			employmentType:         "full_time",
			salaryMin:              400000,
			salaryMax:              600000,
			salaryPeriod:           "yearly",
			salaryCurrency:         "INR",
			rawDescription:         "Great opportunity for freshers to learn modern web application development.",
			expectedGuardrailPass: false,
		},
		{
			name:                   "accepts full-time role when structured salary meets 10 LPA threshold",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Backend Engineer",
			employmentType:         "full_time",
			salaryMin:              1200000,
			salaryMax:              1800000,
			salaryPeriod:           "yearly",
			salaryCurrency:         "INR",
			rawDescription:         "Build scalable payment pipelines.",
			expectedGuardrailPass: true,
		},
		{
			name:                   "accepts full-time role when description specifies compensation above 10 LPA",
			criteriaPrompt:         strictCandidatePrompt,
			jobTitle:               "Systems Engineer",
			employmentType:         "full_time",
			salaryMin:              0,
			salaryMax:              0,
			salaryPeriod:           "",
			salaryCurrency:         "",
			rawDescription:         "Annual compensation: 14-18 LPA based on technical interview performance.",
			expectedGuardrailPass: true,
		},
		{
			name:                   "passes any job when criteria prompt is empty",
			criteriaPrompt:         "",
			jobTitle:               "Software Developer Intern",
			employmentType:         "intern",
			salaryMin:              5000,
			salaryMax:              10000,
			salaryPeriod:           "monthly",
			salaryCurrency:         "INR",
			rawDescription:         "Unpaid or minimal stipend internship.",
			expectedGuardrailPass: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualResult := services.EvaluateNotificationCriteriaGuardrails(
				testCase.criteriaPrompt,
				testCase.jobTitle,
				testCase.employmentType,
				testCase.salaryMin,
				testCase.salaryMax,
				testCase.salaryPeriod,
				testCase.salaryCurrency,
				testCase.rawDescription,
			)

			if actualResult != testCase.expectedGuardrailPass {
				t.Fatalf("expected guardrail result %v, got %v for test case: %s",
					testCase.expectedGuardrailPass,
					actualResult,
					testCase.name,
				)
			}
		})
	}
}
