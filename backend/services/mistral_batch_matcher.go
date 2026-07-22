package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	mistralAPIEndpoint       = "https://api.mistral.ai/v1/chat/completions"
	mistralMatchModel        = "mistral-small-2506"
	mistralRequestsPerSecond = 5
	mistralBatchSize         = 100
	descriptionTruncateChars = 600
	matchScoreThreshold      = 50
	maxConcurrentWorkers     = 5
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
	UserID           string
	ParsedExperience string
	TargetRoles      string
	WorkModels       string
	MinSalary        int
	Currency         string
}

// JobSnippet is the minimal job data sent to Mistral per batch entry.
type JobSnippet struct {
	JobID       string `json:"job_id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

// mistralMatchResult is a single result record returned by the Mistral model.
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

// EvaluatePendingForAllUsers fetches all active users, then for each user runs a
// full evaluation pass over every job where ai_evaluated = false. Designed to run
// as a goroutine triggered by the scraper finish-run signal.
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

	if len(pendingJobs) == 0 {
		log.Printf("[MistralMatcher] No unevaluated jobs found, skipping evaluation pass.")
		return
	}

	log.Printf("[MistralMatcher] Starting evaluation: %d jobs × %d users (using %d parallel workers)",
		len(pendingJobs), len(userProfiles), maxConcurrentWorkers)

	evaluatedJobIDs := make(map[string]bool)
	var mu sync.Mutex

	for _, userProfile := range userProfiles {
		s.evaluateJobsForUser(ctx, userProfile, pendingJobs, evaluatedJobIDs, &mu)
	}

	if err := s.markJobsEvaluated(ctx, evaluatedJobIDs); err != nil {
		log.Printf("[MistralMatcher] Failed to mark jobs as evaluated: %v", err)
	}

	log.Printf("[MistralMatcher] Evaluation complete. Marked %d jobs as evaluated.", len(evaluatedJobIDs))
}

func (s *MistralBatchMatchService) evaluateJobsForUser(
	ctx context.Context,
	userProfile UserProfile,
	pendingJobs []JobSnippet,
	evaluatedJobIDs map[string]bool,
	mu *sync.Mutex,
) {
	batches := chunkJobSnippets(pendingJobs, mistralBatchSize)
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

				results, err := s.callMistralBatch(ctx, userProfile, work.items)
				if err != nil {
					log.Printf("[MistralMatcher] Worker %d: Batch %d for user %s failed: %v", workerID, work.index, userProfile.UserID, err)
					continue
				}

				if err := s.upsertMatchResults(ctx, userProfile.UserID, results); err != nil {
					log.Printf("[MistralMatcher] Worker %d: Failed upserting batch %d for user %s: %v", workerID, work.index, userProfile.UserID, err)
					continue
				}

				if mu != nil {
					mu.Lock()
					for _, result := range results {
						evaluatedJobIDs[result.JobID] = true
					}
					mu.Unlock()
				}
			}
		}(w)
	}

	wg.Wait()
}

func (s *MistralBatchMatchService) callMistralBatch(
	ctx context.Context,
	userProfile UserProfile,
	batch []JobSnippet,
) ([]mistralMatchResult, error) {
	prompt := s.buildMatchPrompt(userProfile, batch)

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

	var batchResponse mistralBatchResponse
	if err := json.Unmarshal([]byte(apiResponse.Choices[0].Message.Content), &batchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Mistral batch result JSON: %w", err)
	}

	return batchResponse.Results, nil
}

func (s *MistralBatchMatchService) buildMatchPrompt(userProfile UserProfile, batch []JobSnippet) string {
	batchJSON, _ := json.Marshal(batch)

	return fmt.Sprintf(`You are a precise job-fit evaluator. A candidate is looking for early-career roles (0-3 years experience).

CANDIDATE PROFILE:
%s

Target roles: %s
Preferred work models: %s
Minimum salary: %d %s

EVALUATION RULES:
- Match ONLY if the role targets 0-3 years experience (Junior, New Grad, Intern, Entry Level, SDE I, Associate).
- REJECT roles requiring 4+ years, or titled Senior/Staff/Lead/Principal/Director/Manager/VP.
- REJECT roles restricted to US-only, UK-only, or Europe-only if the candidate is India-based or remote.
- Score 0-100 based on stack overlap with the candidate's experience.
- is_matched = true when match_score >= %d.

For each job in the list below, return exactly one result object.
Return ONLY valid JSON in this exact shape:
{"results": [{"job_id": "...", "is_matched": true|false, "match_score": 0-100, "seniority": "Junior|Mid|Intern|Senior", "tech_stack": ["lang", "framework"], "reasoning": "one sentence"}]}

JOBS:
%s`,
		userProfile.ParsedExperience,
		userProfile.TargetRoles,
		userProfile.WorkModels,
		userProfile.MinSalary,
		userProfile.Currency,
		matchScoreThreshold,
		string(batchJSON),
	)
}

func (s *MistralBatchMatchService) upsertMatchResults(ctx context.Context, userID string, results []mistralMatchResult) error {
	for _, result := range results {
		techStackJSON, _ := json.Marshal(result.TechStack)

		log.Printf("[MistralMatcher] User %s evaluated job %s -> score: %d, matched: %v, reasoning: %s",
			userID, result.JobID, result.MatchScore, result.IsMatched, result.MatchReasoning)

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
		matchReasons, _ := json.Marshal([]string{result.MatchReasoning})
		_, err := s.DB.Exec(ctx, query,
			userID,
			result.JobID,
			result.MatchScore,
			result.IsMatched,
			result.Seniority,
			techStackJSON,
			result.MatchReasoning,
			mistralMatchModel,
			matchReasons,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert match for job %s: %w", result.JobID, err)
		}
	}
	return nil
}

func (s *MistralBatchMatchService) fetchAllUserProfiles(ctx context.Context) ([]UserProfile, error) {
	query := `
		SELECT u.id,
		       COALESCE(u.parsed_experience::text, ''),
		       COALESCE(p.target_roles::text, '[]'),
		       COALESCE(p.work_models::text, '[]'),
		       COALESCE(p.min_salary, 0),
		       COALESCE(p.currency, 'USD')
		FROM users u
		LEFT JOIN user_preferences p ON u.id = p.user_id;
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

func chunkJobSnippets(snippets []JobSnippet, size int) [][]JobSnippet {
	var batches [][]JobSnippet
	for i := 0; i < len(snippets); i += size {
		end := i + size
		if end > len(snippets) {
			end = len(snippets)
		}
		batches = append(batches, snippets[i:end])
	}
	return batches
}
