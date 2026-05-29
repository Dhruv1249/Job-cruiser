package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/genai"
)

type AIMatcherService struct {
	DB     *pgxpool.Pool
	APIKey string
}

type MatchResponse struct {
	MatchScore      int      `json:"match_score"`
	MatchReasons    []string `json:"match_reasons"`
	SuggestedAction string   `json:"suggested_action"`
}

func (s *AIMatcherService) ComputePremiumMatch(userID string, jobID string) (*MatchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// 1. Fetch absolutely all candidate preference and history context
	var primaryEmail, userLoc, parsedExp, latexCV, targetRoles, workModels, currencyPref, customAnswers string
	var minSalary int
	userQuery := `
		SELECT u.primary_email, 
		       COALESCE(u.location, 'Unknown'), 
		       COALESCE(u.parsed_experience::text, '[]'), 
		       COALESCE(u.latex_cv, ''),
		       COALESCE(p.target_roles::text, '[]'), 
		       COALESCE(p.work_models::text, '[]'), 
		       COALESCE(p.min_salary, 0), 
		       COALESCE(p.currency, 'USD'), 
		       COALESCE(p.custom_form_answers::text, '{}')
		FROM users u
		JOIN user_preferences p ON u.id = p.user_id
		WHERE u.id = $1;
	`
	err := s.DB.QueryRow(ctx, userQuery, userID).Scan(
		&primaryEmail, &userLoc, &parsedExp, &latexCV, &targetRoles, &workModels, &minSalary, &currencyPref, &customAnswers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed fetching comprehensive user context: %v", err)
	}

	// 2. Fetch absolutely all job telemetry metadata details
	var title, jobLoc, salaryMin, salaryMax, jobCurrency, expReq, jobType, isEasyApply, isRemote, source, url, postedDate, tags, rawDesc string
	jobQuery := `
		SELECT COALESCE(title, ''), 
		       COALESCE(location, 'Unknown'), 
		       COALESCE(salary_min::text, '0'), 
		       COALESCE(salary_max::text, '0'), 
		       COALESCE(currency, 'USD'), 
		       COALESCE(experience_required, 'Unspecified'), 
		       COALESCE(job_type, 'Unspecified'), 
		       COALESCE(is_easy_apply::text, 'false'), 
		       COALESCE(is_remote::text, 'false'), 
		       COALESCE(source, 'Unknown'), 
		       COALESCE(url, ''), 
		       COALESCE(posted_date, ''), 
		       COALESCE(tags::text, '[]'), 
		       COALESCE(raw_desc, '')
		FROM jobs
		WHERE id = $1;
	`
	// Using COALESCE to safely convert nullable types into strings for the prompt injection
	err = s.DB.QueryRow(ctx, jobQuery, jobID).Scan(
		&title, &jobLoc, &salaryMin, &salaryMax, &jobCurrency, &expReq, &jobType, &isEasyApply, &isRemote, &source, &url, &postedDate, &tags, &rawDesc,
	)
	if err != nil {
		return nil, fmt.Errorf("failed fetching comprehensive job data context: %v", err)
	}

	// 3. Construct dense profile payload matching instructions
	prompt := fmt.Sprintf(`You are a precise, data-grounded reverse-ATS match executor. Evaluate the suitability of the job against the complete candidate metrics.

[CANDIDATE METRICS]:
- Email: %s
- Target Roles: %s
- Target Location & Allowed Models: %s (Current Candidate Base: %s)
- Compensation Floor: %d %s
- Core Structured Experience Context: %s
- Raw LaTeX Resume Reference: %s
- Special Custom Answers/Traits: %s

[JOB POSTING TELEMETRY]:
- Title: %s
- Location: %s
- Salary Band Range: %s to %s %s
- Work Framework constraints: Remote=%s, EasyApply=%s, Employment Type=%s
- Target Experience Bracket Required: %s
- Technical Tags/Keywords Extracted: %s
- Source Provider Origin: %s (URL: %s, Posted: %s)
- Complete Raw Markdown/Text Body Description: %s

Execute a rigorous verification. Determine if the requirements fit the candidate experience bracket, stack constraints, and salary boundaries. Output a clean JSON object structure with zero trailing formatting or Markdown wrappers:
{
  "match_score": integer (0 to 100),
  "match_reasons": ["highly detailed explicit contextual point 1", "highly detailed explicit contextual point 2"],
  "suggested_action": "string (must be either 'apply', 'cold_email', 'skip', or 'review')"
}`, primaryEmail, targetRoles, workModels, userLoc, minSalary, currencyPref, parsedExp, latexCV, customAnswers,
		title, jobLoc, salaryMin, salaryMax, jobCurrency, isRemote, isEasyApply, jobType, expReq, tags, source, url, postedDate, rawDesc)

	// 4. Delegate to the SDK handler
	return s.callOfficialGenAISDK(ctx, prompt)
}

func (s *AIMatcherService) callOfficialGenAISDK(ctx context.Context, prompt string) (*MatchResponse, error) {
	// Initialize the official Google GenAI Client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  s.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %v", err)
	}

	// Pull model dynamically from environment variable, fallback to Gemma-4
	modelName := os.Getenv("MATCHER_AI_MODEL")
	if modelName == "" {
		modelName = "gemma-4-31b-it" // Standard instruction-tuned Gemma model
	}

	// Lock the engine into returning strict JSON and remove temperature hallucination
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      genai.Ptr[float32](0.0), // 0 for deterministic matching
	}

	// Execute the request using the SDK's abstraction
	result, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), config)
	if err != nil {
		return nil, fmt.Errorf("ai engine generation failed: %v", err)
	}

	// Extract the raw text from the SDK result object
	rawJSON := result.Text()
	if rawJSON == "" {
		return nil, fmt.Errorf("empty response from model")
	}

	// Unmarshal directly into our Go struct
	var matchResult MatchResponse
	if err := json.Unmarshal([]byte(rawJSON), &matchResult); err != nil {
		return nil, fmt.Errorf("failed parsing structural json output: %v\nRaw Output: %s", err, rawJSON)
	}

	return &matchResult, nil
}
