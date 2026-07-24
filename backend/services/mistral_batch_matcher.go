package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var yoeRe = regexp.MustCompile(`(?i)\b([3-9]|\d{2,})\s*(\+)?\s*(years?|yrs?)\b|\b([3-9]|\d{2,})\s*-\s*\d+\s*(years?|yrs?)\b`)

func hasExcessiveYOE(text string) bool {
	return yoeRe.MatchString(text)
}

func isNonTargetHybridOrOnsite(location string, rawDesc string, isRemote bool, targetLocationsStr string) bool {
	combined := strings.ToLower(location + " " + rawDesc)
	hasHybridOrOnsite := strings.Contains(combined, "hybrid") || strings.Contains(combined, "onsite") || strings.Contains(combined, "on-site") || strings.Contains(combined, "in-person") || strings.Contains(combined, "day/week") || strings.Contains(combined, "days/week")
	
	if hasHybridOrOnsite {
		if strings.Contains(targetLocationsStr, "India") {
			nonIndiaCities := []string{"nyc", "new york", "san francisco", "sf ", "london", "seattle", "austin", "chicago", "boston", "toronto"}
			for _, city := range nonIndiaCities {
				if strings.Contains(combined, city) {
					return true
				}
			}
		}
	}
	return false
}

const (
	mistralAPIEndpoint       = "https://api.mistral.ai/v1/chat/completions"
	mistralMatchModel        = "mistral-small-2506"
	mistralRequestsPerSecond = 5
	maxEvaluationsPerPrompt  = 40
	descriptionTruncateChars = 6000
	maxCharBudgetPerBatch    = 64000
	matchScoreThreshold      = 50
	maxConcurrentWorkers     = 5
	maxCandidatesPerPrompt   = 4
)

// MistralBatchMatchService evaluates batches of unevaluated jobs against each
// user's parsed CV using the Mistral mistral-small-2506 API. Results are written
// to user_job_matches with a 0-100 match_score and is_matched flag.
type MistralBatchMatchService struct {
	DB         *pgxpool.Pool
	MistralKey string
	HTTPClient *http.Client
}

// UserProfile holds the CV data needed to build a personalised Mistral prompt.
type UserProfile struct {
	UserID           string `json:"user_id"`
	ParsedExperience string `json:"parsed_experience"`
	TargetRoles      string `json:"target_roles"`
	TargetIndustries string `json:"target_industries"`
	TargetLocations  string `json:"target_locations"`
	WorkModels       string `json:"work_models"`
	MinSalary        int    `json:"min_salary"`
	Currency         string `json:"currency"`
}

// JobSnippet is the minimal job data sent to Mistral per batch entry.
type JobSnippet struct {
	JobID       string `json:"job_id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

// mistralMatchResult is a single result record returned by the Mistral model for a single candidate.
type mistralMatchResult struct {
	JobID          string   `json:"job_id"`
	IsMatched      bool     `json:"is_matched"`
	MatchScore     int      `json:"match_score"`
	Seniority      string   `json:"seniority"`
	TechStack      []string `json:"tech_stack"`
	MatchReasoning string   `json:"reasoning"`
}

type mistralBatchResponse struct {
	Results []mistralMatchResult `json:"results"`
}

type candidateMatchResult struct {
	UserID         string   `json:"user_id"`
	IsMatched      bool     `json:"is_matched"`
	MatchScore     int      `json:"match_score"`
	TechStack      []string `json:"tech_stack"`
	MatchReasoning string   `json:"reasoning"`
}

type jobMultiMatchResult struct {
	JobID     string                 `json:"job_id"`
	Seniority string                 `json:"seniority"`
	SalaryMin *float64               `json:"salary_min"`
	SalaryMax *float64               `json:"salary_max"`
	Currency  string                 `json:"currency"`
	Matches   []candidateMatchResult `json:"matches"`
}

type mistralMultiBatchResponse struct {
	Results []jobMultiMatchResult `json:"results"`
}

type mistralRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralRequest struct {
	Model          string                  `json:"model"`
	Messages       []mistralRequestMessage `json:"messages"`
	ResponseFormat map[string]string       `json:"response_format"`
	Temperature    float32                 `json:"temperature"`
}

type mistralAPIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// EvaluatePendingForAllUsers fetches all active users, groups them into chunks of up to 4 candidates,
// and evaluates every job where ai_evaluated = false against all candidates in a single prompt pass.
func (s *MistralBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	userProfiles, err := s.fetchAllUserProfiles(ctx)
	if err != nil {
		log.Printf("[MistralMatcher] Failed to fetch user profiles: %v", err)
		return
	}

	pendingJobs, err := s.fetchUnevaluatedJobs(ctx)
	if err != nil {
		log.Printf("[MistralMatcher] Failed to fetch unevaluated jobs: %v", err)
		return
	}

	if len(pendingJobs) == 0 || len(userProfiles) == 0 {
		log.Printf("[MistralMatcher] No unevaluated jobs or user profiles found, skipping evaluation pass.")
		return
	}

	userChunks := chunkUserProfiles(userProfiles, maxCandidatesPerPrompt)

	log.Printf("[MistralMatcher] Starting multi-candidate evaluation: %d jobs × %d users (%d user chunks, %d parallel workers)",
		len(pendingJobs), len(userProfiles), len(userChunks), maxConcurrentWorkers)

	evaluatedJobIDs := make(map[string]bool)
	var mu sync.Mutex

	for chunkIndex, userChunk := range userChunks {
		s.evaluateJobsForUserChunk(ctx, userChunk, pendingJobs, evaluatedJobIDs, &mu)
		log.Printf("[MistralMatcher] Finished evaluation pass for user chunk %d/%d", chunkIndex+1, len(userChunks))
	}

	if err := s.markJobsEvaluated(ctx, evaluatedJobIDs); err != nil {
		log.Printf("[MistralMatcher] Failed to mark jobs as evaluated: %v", err)
	}

	log.Printf("[MistralMatcher] Evaluation complete. Marked %d jobs as evaluated.", len(evaluatedJobIDs))
}

// EvaluateForSingleUser scores all jobs that have not yet been matched against the given user.
// Called when a user enables AI matching for the first time — at that point all previously
// ingested and ai_evaluated jobs still have no user_job_matches row for this user.
func (s *MistralBatchMatchService) EvaluateForSingleUser(ctx context.Context, userID string) {
	profile, err := s.fetchSingleUserProfile(ctx, userID)
	if err != nil {
		log.Printf("[MistralMatcher] Failed to fetch profile for user %s: %v", userID, err)
		return
	}

	jobs, err := s.fetchJobsUnmatchedForUser(ctx, userID)
	if err != nil {
		log.Printf("[MistralMatcher] Failed to fetch unmatched jobs for user %s: %v", userID, err)
		return
	}

	if len(jobs) == 0 {
		log.Printf("[MistralMatcher] No unmatched jobs found for user %s, nothing to evaluate.", userID)
		return
	}

	log.Printf("[MistralMatcher] Evaluating %d jobs for newly enabled user %s", len(jobs), userID)

	evaluatedJobIDs := make(map[string]bool)
	var mu sync.Mutex
	s.evaluateJobsForUserChunk(ctx, []UserProfile{*profile}, jobs, evaluatedJobIDs, &mu)

	log.Printf("[MistralMatcher] Single-user evaluation complete for %s: %d jobs scored.", userID, len(evaluatedJobIDs))
}


func (s *MistralBatchMatchService) evaluateJobsForUserChunk(
	ctx context.Context,
	userChunk []UserProfile,
	pendingJobs []JobSnippet,
	evaluatedJobIDs map[string]bool,
	mu *sync.Mutex,
) {
	candidateCount := len(userChunk)
	if candidateCount == 0 {
		return
	}

	batchSize := maxEvaluationsPerPrompt / candidateCount
	if batchSize < 10 {
		batchSize = 10
	}

	batches := chunkJobSnippets(pendingJobs, batchSize)
	if len(batches) == 0 {
		return
	}

	rateLimiter := time.NewTicker(time.Second / time.Duration(mistralRequestsPerSecond))
	defer rateLimiter.Stop()

	type batchWork struct {
		index int
		items []JobSnippet
	}

	workChan := make(chan batchWork, len(batches))
	for i, b := range batches {
		workChan <- batchWork{index: i, items: b}
	}
	close(workChan)

	numWorkers := maxConcurrentWorkers
	if len(batches) < numWorkers {
		numWorkers = len(batches)
	}

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for work := range workChan {
				select {
				case <-ctx.Done():
					return
				case <-rateLimiter.C:
				}

				results, err := s.callMistralMultiBatch(ctx, userChunk, work.items)
				if err != nil {
					log.Printf("[MistralMatcher] Worker %d: Multi-batch %d failed: %v", workerID, work.index, err)
					continue
				}

				if err := s.upsertMultiMatchResults(ctx, results); err != nil {
					log.Printf("[MistralMatcher] Worker %d: Failed upserting multi-batch %d: %v", workerID, work.index, err)
					continue
				}

				if mu != nil {
					mu.Lock()
					for _, jobRes := range results {
						evaluatedJobIDs[jobRes.JobID] = true
					}
					mu.Unlock()
				}
			}
		}(w)
	}

	wg.Wait()
}

func (s *MistralBatchMatchService) evaluateJobsForUser(
	ctx context.Context,
	userProfile UserProfile,
	pendingJobs []JobSnippet,
	evaluatedJobIDs map[string]bool,
	mu *sync.Mutex,
) {
	s.evaluateJobsForUserChunk(ctx, []UserProfile{userProfile}, pendingJobs, evaluatedJobIDs, mu)
}

func (s *MistralBatchMatchService) callMistralMultiBatch(
	ctx context.Context,
	userChunk []UserProfile,
	batch []JobSnippet,
) ([]jobMultiMatchResult, error) {
	prompt := s.buildMultiMatchPrompt(userChunk, batch)
	start := time.Now()

	requestPayload := mistralRequest{
		Model: mistralMatchModel,
		Messages: []mistralRequestMessage{
			{Role: "user", Content: prompt},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
		Temperature:    0.0,
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Mistral request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, mistralAPIEndpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create Mistral HTTP request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+s.MistralKey)
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := s.HTTPClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("Mistral HTTP request failed: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Mistral response body: %w", err)
	}

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mistral API returned status %d: %s", httpResponse.StatusCode, string(responseBody))
	}

	var apiResponse mistralAPIResponse
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Mistral API response envelope: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("Mistral returned no choices")
	}

	var batchResponse mistralMultiBatchResponse
	contentBytes := []byte(apiResponse.Choices[0].Message.Content)
	if err := json.Unmarshal(contentBytes, &batchResponse); err != nil || len(batchResponse.Results) == 0 {
		var directList []jobMultiMatchResult
		if errArray := json.Unmarshal(contentBytes, &directList); errArray == nil {
			batchResponse.Results = directList
		} else {
			return nil, fmt.Errorf("failed to parse Mistral multi-batch result JSON: %w", err)
		}
	}

	log.Printf("[MistralMatcher] Multi-batch of %d jobs × %d candidates finished in %v", len(batch), len(userChunk), time.Since(start))
	return batchResponse.Results, nil
}

func (s *MistralBatchMatchService) callMistralBatch(
	ctx context.Context,
	userProfile UserProfile,
	batch []JobSnippet,
) ([]mistralMatchResult, error) {
	multiResults, err := s.callMistralMultiBatch(ctx, []UserProfile{userProfile}, batch)
	if err != nil {
		return nil, err
	}

	var singleResults []mistralMatchResult
	for _, jobRes := range multiResults {
		for _, m := range jobRes.Matches {
			singleResults = append(singleResults, mistralMatchResult{
				JobID:          jobRes.JobID,
				IsMatched:      m.IsMatched,
				MatchScore:     m.MatchScore,
				Seniority:      jobRes.Seniority,
				TechStack:      m.TechStack,
				MatchReasoning: m.MatchReasoning,
			})
		}
	}
	return singleResults, nil
}

func (s *MistralBatchMatchService) buildMultiMatchPrompt(userChunk []UserProfile, batch []JobSnippet) string {
	candidatesJSON, _ := json.Marshal(userChunk)
	batchJSON, _ := json.Marshal(batch)

	return fmt.Sprintf(`You are a strict, ultra-precise job-fit evaluator. Evaluate each job against candidate profiles.

CANDIDATE PROFILES:
%s

CRITICAL EVALUATION RULES:

1. CANDIDATE LOCATION & REMOTE ELIGIBILITY (STRICT PASS/FAIL):
- Candidates are physically located in INDIA.
- A job specified as "US Remote", "UK Remote", "Canada Remote", "EU Remote", or "Remote (US)" requires being physically located or authorized in that specific country. A candidate in India CANNOT work a US-restricted or UK-restricted remote job!
- REJECT IMMEDIATELY (match_score = 0, is_matched = false) if:
  * The job is On-site or Hybrid outside India (e.g. NYC, San Francisco, London, Berlin, Austin).
  * The job is country-restricted Remote outside India (e.g. "US Remote", "Remote - US", "UK Remote", "Remote (US/Canada)").
- MATCH ONLY IF the location is explicitly:
  * Located in INDIA (On-site, Hybrid, or India Remote).
  * Explicitly GLOBAL Remote, Remote Worldwide, Remote (Global), or Remote (APAC) where anyone in the world (including India) can work.

2. CANDIDATE EXPERIENCE LEVEL (STRICT PASS/FAIL):
- The candidates are Early-Career / Junior Software Engineers with 1 YEAR OF EXPERIENCE (0-2 YOE target range).
- REJECT IMMEDIATELY (match_score = 0, is_matched = false) if:
  * The title contains Senior, Sr, Lead, Staff, Principal, Architect, Director, Manager, VP, Head of.
  * The job description explicitly requires 3+ years, 4+ years, 5+ years, or 3-5+ years of experience.
- MATCH ONLY IF the job requires 0-2 years of experience or is explicitly Entry Level, Junior, Associate, New Grad, or Intern.

3. SALARY EXTRACTION:
- Extract numeric annual salary range from the job description if mentioned (e.g., "$120,000 - $150,000" or "$120k-$150k" -> salary_min: 120000, salary_max: 150000, currency: "USD"). If not mentioned, set salary_min and salary_max to null.

4. MATCH SCORING:
- Score 0-100 based on tech stack overlap and role fit.
- If Location, Country Remote eligibility, or Experience Level fails, match_score MUST BE 0 and is_matched MUST BE false.
- is_matched = true ONLY when match_score >= %d.

Return ONLY valid JSON matching this exact structure:
{"results": [{"job_id": "...", "seniority": "Junior|Mid|Intern|Senior", "salary_min": 120000, "salary_max": 150000, "currency": "USD", "matches": [{"user_id": "...", "is_matched": true|false, "match_score": 0-100, "tech_stack": ["Python"], "reasoning": "1 short sentence explaining fit or rejection"}]}]}

JOBS TO EVALUATE:
%s`,
		string(candidatesJSON),
		matchScoreThreshold,
		string(batchJSON),
	)
}

func (s *MistralBatchMatchService) buildMatchPrompt(userProfile UserProfile, batch []JobSnippet) string {
	return s.buildMultiMatchPrompt([]UserProfile{userProfile}, batch)
}

func isSeniorRoleTitle(title string) bool {
	t := strings.ToLower(title)
	keywords := []string{
		"senior", "sr.", "sr ", "lead", "principal", "staff", "architect",
		"director", "head of", "vp", "vice president", "manager", "management",
	}
	for _, kw := range keywords {
		if strings.Contains(t, kw) {
			return true
		}
	}
	return false
}

func isNonTargetOnsite(location string, isRemote bool, targetLocationsStr string) bool {
	if isRemote {
		return false
	}
	loc := strings.ToLower(location)
	if loc == "" || strings.Contains(loc, "remote") || strings.Contains(loc, "anywhere") || strings.Contains(loc, "worldwide") || strings.Contains(loc, "global") {
		return false
	}
	// If candidate targets India, reject physical non-India locations (e.g. CA, US, UK, London, NY)
	if strings.Contains(targetLocationsStr, "India") {
		if !strings.Contains(loc, "india") && !strings.Contains(loc, ", in") && !strings.Contains(loc, "in,") && loc != "india" {
			return true
		}
	}
	return false
}

func (s *MistralBatchMatchService) upsertMultiMatchResults(ctx context.Context, results []jobMultiMatchResult) error {
	for _, jobResult := range results {
		var jobTitle, jobLocation, rawDesc string
		var isRemote bool
		_ = s.DB.QueryRow(ctx, "SELECT title, COALESCE(location, ''), is_remote, COALESCE(raw_desc, '') FROM jobs WHERE id = $1::uuid;", jobResult.JobID).Scan(&jobTitle, &jobLocation, &isRemote, &rawDesc)

		if jobResult.SalaryMin != nil || jobResult.SalaryMax != nil {
			var minInt, maxInt *int
			if jobResult.SalaryMin != nil {
				v := int(*jobResult.SalaryMin)
				minInt = &v
			}
			if jobResult.SalaryMax != nil {
				v := int(*jobResult.SalaryMax)
				maxInt = &v
			}
			_, _ = s.DB.Exec(ctx, `
				UPDATE jobs 
				SET salary_min = COALESCE($1, salary_min), 
				    salary_max = COALESCE($2, salary_max), 
				    currency   = CASE WHEN $3 != '' THEN $3 ELSE currency END 
				WHERE id = $4::uuid;
			`, minInt, maxInt, jobResult.Currency, jobResult.JobID)
		}

		isSenior := isSeniorRoleTitle(jobTitle)
		isExcessiveExperience := hasExcessiveYOE(jobTitle + " " + rawDesc)

		for _, match := range jobResult.Matches {
			userID := strings.TrimSpace(match.UserID)
			if len(userID) > 36 {
				userID = userID[:36]
			}
			if userID == "" || jobResult.JobID == "" {
				continue
			}

			var targetLocs string
			_ = s.DB.QueryRow(ctx, "SELECT COALESCE(target_locations::text, '') FROM user_preferences WHERE user_id = $1;", userID).Scan(&targetLocs)

			score := match.MatchScore
			isMatched := match.IsMatched
			reasoning := match.MatchReasoning

			if isSenior || isExcessiveExperience || isNonTargetOnsite(jobLocation, isRemote, targetLocs) || isNonTargetHybridOrOnsite(jobLocation, rawDesc, isRemote, targetLocs) {
				score = 0
				isMatched = false
				reasoning = ""
			}

			techStackJSON, _ := json.Marshal(match.TechStack)

			if isMatched {
				log.Printf("[MistralMatcher] User %s evaluated job %s (%s, %s) -> score: %d, matched: true, reasoning: %s",
					userID, jobResult.JobID, jobTitle, jobLocation, score, reasoning)
			}

			seniorityVal := jobResult.Seniority
			if len(seniorityVal) > 50 {
				seniorityVal = seniorityVal[:50]
			}

			query := `
				INSERT INTO user_job_matches (user_id, job_id, match_score, is_ai_matched, seniority, tech_stack, match_reasoning, ai_model, evaluated_at, match_reasons)
				VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, $9)
				ON CONFLICT (user_id, job_id) DO UPDATE SET
					match_score     = EXCLUDED.match_score,
					is_ai_matched   = EXCLUDED.is_ai_matched,
					seniority       = EXCLUDED.seniority,
					tech_stack      = EXCLUDED.tech_stack,
					match_reasoning = EXCLUDED.match_reasoning,
					ai_model        = EXCLUDED.ai_model,
					evaluated_at    = CURRENT_TIMESTAMP;
			`
			matchReasons, _ := json.Marshal([]string{reasoning})
			_, err := s.DB.Exec(ctx, query,
				userID,
				jobResult.JobID,
				score,
				isMatched,
				seniorityVal,
				techStackJSON,
				reasoning,
				mistralMatchModel,
				matchReasons,
			)
			if err != nil {
				return fmt.Errorf("failed to upsert match for user %s, job %s: %w", match.UserID, jobResult.JobID, err)
			}
		}
	}
	return nil
}

func (s *MistralBatchMatchService) upsertMatchResults(ctx context.Context, userID string, results []mistralMatchResult) error {
	var multiResults []jobMultiMatchResult
	for _, r := range results {
		multiResults = append(multiResults, jobMultiMatchResult{
			JobID:     r.JobID,
			Seniority: r.Seniority,
			Matches: []candidateMatchResult{
				{
					UserID:         userID,
					IsMatched:      r.IsMatched,
					MatchScore:     r.MatchScore,
					TechStack:      r.TechStack,
					MatchReasoning: r.MatchReasoning,
				},
			},
		})
	}
	return s.upsertMultiMatchResults(ctx, multiResults)
}

func (s *MistralBatchMatchService) fetchAllUserProfiles(ctx context.Context) ([]UserProfile, error) {
	query := `
		SELECT u.id,
		       COALESCE(u.parsed_experience::text, ''),
		       COALESCE(p.target_roles::text, '[]'),
		       COALESCE(p.target_industries::text, '[]'),
		       COALESCE(p.target_locations::text, '["India (On-site & Hybrid)", "India (Remote)", "Global Remote"]'),
		       COALESCE(p.work_models::text, '[]'),
		       COALESCE(p.min_salary, 0),
		       COALESCE(p.currency, 'USD')
		FROM users u
		LEFT JOIN user_preferences p ON u.id = p.user_id
		WHERE u.ai_matching_enabled = true;
	`
	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query user profiles: %w", err)
	}
	defer rows.Close()

	var profiles []UserProfile
	for rows.Next() {
		var profile UserProfile
		if err := rows.Scan(
			&profile.UserID,
			&profile.ParsedExperience,
			&profile.TargetRoles,
			&profile.TargetIndustries,
			&profile.TargetLocations,
			&profile.WorkModels,
			&profile.MinSalary,
			&profile.Currency,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user profile row: %w", err)
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

// fetchSingleUserProfile retrieves the CV and preference data for one specific user.
func (s *MistralBatchMatchService) fetchSingleUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	query := `
		SELECT u.id,
		       COALESCE(u.parsed_experience::text, ''),
		       COALESCE(p.target_roles::text, '[]'),
		       COALESCE(p.target_industries::text, '[]'),
		       COALESCE(p.target_locations::text, '["India (On-site & Hybrid)", "India (Remote)", "Global Remote"]'),
		       COALESCE(p.work_models::text, '[]'),
		       COALESCE(p.min_salary, 0),
		       COALESCE(p.currency, 'USD')
		FROM users u
		LEFT JOIN user_preferences p ON u.id = p.user_id
		WHERE u.id = $1;
	`
	var profile UserProfile
	err := s.DB.QueryRow(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.ParsedExperience,
		&profile.TargetRoles,
		&profile.TargetIndustries,
		&profile.TargetLocations,
		&profile.WorkModels,
		&profile.MinSalary,
		&profile.Currency,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch single user profile: %w", err)
	}
	return &profile, nil
}

// fetchJobsUnmatchedForUser retrieves all jobs that have no user_job_matches row for the given user.
// This covers both newly ingested jobs and jobs ingested before the user enabled AI matching.
func (s *MistralBatchMatchService) fetchJobsUnmatchedForUser(ctx context.Context, userID string) ([]JobSnippet, error) {
	query := `
		SELECT j.id, j.title, COALESCE(c.name, ''), j.location,
		       LEFT(COALESCE(j.raw_desc, ''), $1)
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE NOT EXISTS (
			SELECT 1 FROM user_job_matches ujm
			WHERE ujm.job_id = j.id AND ujm.user_id = $2
		)
		ORDER BY j.scraped_at DESC;
	`
	rows, err := s.DB.Query(ctx, query, descriptionTruncateChars, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unmatched jobs for user: %w", err)
	}
	defer rows.Close()

	var snippets []JobSnippet
	for rows.Next() {
		var snippet JobSnippet
		if err := rows.Scan(&snippet.JobID, &snippet.Title, &snippet.Company, &snippet.Location, &snippet.Description); err != nil {
			return nil, fmt.Errorf("failed to scan job snippet row: %w", err)
		}
		snippets = append(snippets, snippet)
	}
	return snippets, nil
}



func (s *MistralBatchMatchService) fetchUnevaluatedJobs(ctx context.Context) ([]JobSnippet, error) {
	query := `
		SELECT j.id, j.title, COALESCE(c.name, ''), j.location,
		       LEFT(COALESCE(j.raw_desc, ''), $1)
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE j.ai_evaluated = false
		ORDER BY j.scraped_at DESC;
	`
	rows, err := s.DB.Query(ctx, query, descriptionTruncateChars)
	if err != nil {
		return nil, fmt.Errorf("failed to query unevaluated jobs: %w", err)
	}
	defer rows.Close()

	var snippets []JobSnippet
	for rows.Next() {
		var snippet JobSnippet
		if err := rows.Scan(&snippet.JobID, &snippet.Title, &snippet.Company, &snippet.Location, &snippet.Description); err != nil {
			return nil, fmt.Errorf("failed to scan job snippet row: %w", err)
		}
		snippets = append(snippets, snippet)
	}
	return snippets, nil
}

func (s *MistralBatchMatchService) markJobsEvaluated(ctx context.Context, jobIDs map[string]bool) error {
	if len(jobIDs) == 0 {
		return nil
	}

	ids := make([]string, 0, len(jobIDs))
	for id := range jobIDs {
		ids = append(ids, id)
	}

	query := `
		UPDATE jobs SET ai_evaluated = true, ai_evaluated_at = CURRENT_TIMESTAMP
		WHERE id = ANY($1::uuid[]);
	`
	_, err := s.DB.Exec(ctx, query, ids)
	return err
}

func chunkUserProfiles(profiles []UserProfile, size int) [][]UserProfile {
	var chunks [][]UserProfile
	for i := 0; i < len(profiles); i += size {
		end := i + size
		if end > len(profiles) {
			end = len(profiles)
		}
		chunks = append(chunks, profiles[i:end])
	}
	return chunks
}

func chunkJobSnippets(snippets []JobSnippet, maxBatchCount int) [][]JobSnippet {
	var batches [][]JobSnippet
	var currentBatch []JobSnippet
	currentChars := 0

	for _, snippet := range snippets {
		snippetLen := len(snippet.Description) + len(snippet.Title) + len(snippet.Company)
		if len(currentBatch) > 0 && (currentChars+snippetLen > maxCharBudgetPerBatch || len(currentBatch) >= maxBatchCount) {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentChars = 0
		}
		currentBatch = append(currentBatch, snippet)
		currentChars += snippetLen
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}
