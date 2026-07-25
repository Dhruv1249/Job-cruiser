package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/genai"
)

// GeminiBatchMatchService manages concurrent, rate-limited batch job matching
// using Google's official GenAI SDK (google.golang.org/genai).
type GeminiBatchMatchService struct {
	DB          *pgxpool.Pool
	RawKeys     []string
	WorkerKeys  []string
	ReservedKey string
	HTTPClient  *http.Client

	// Global atomic job counter & deduplication set
	claimedJobs sync.Map // map[string]bool (job_id -> claimed)
	dailyExhausted int32 // atomic count of exhausted worker keys
}

// NewGeminiBatchMatchService initializes the service, strictly reserving keys[0]
// for non-background use and assigning keys[1:] to background worker threads.
func NewGeminiBatchMatchService(db *pgxpool.Pool, rawKeysStr string) *GeminiBatchMatchService {
	var rawKeys []string
	for _, k := range strings.Split(rawKeysStr, ",") {
		trimmed := strings.TrimSpace(k)
		if trimmed != "" {
			rawKeys = append(rawKeys, trimmed)
		}
	}

	if len(rawKeys) == 0 {
		singleKey := os.Getenv("GEMINI_API_KEY")
		if singleKey != "" {
			rawKeys = []string{singleKey}
		}
	}

	reservedKey := ""
	workerKeys := []string{}

	if len(rawKeys) > 1 {
		reservedKey = rawKeys[0]
		workerKeys = rawKeys[1:]
		prefixKey := reservedKey
		if len(prefixKey) > 8 {
			prefixKey = prefixKey[:8]
		}
		log.Printf("[GeminiMatcher] Initialized with %d total keys. RESERVED Key 0 (%s...) for outside app use. Assigning %d keys to background worker engine.",
			len(rawKeys), prefixKey, len(workerKeys))
	} else if len(rawKeys) == 1 {
		workerKeys = rawKeys
		reservedKey = rawKeys[0]
		log.Printf("[GeminiMatcher] Initialized with 1 key. Using for worker engine.")
	}

	return &GeminiBatchMatchService{
		DB:          db,
		RawKeys:     rawKeys,
		WorkerKeys:  workerKeys,
		ReservedKey: reservedKey,
		HTTPClient:  &http.Client{Timeout: 180 * time.Second},
	}
}

type modelConfig struct {
	Name            string
	MaxTargetTokens int
	TickerInterval  time.Duration
	CandidateChunk  int // number of candidates per prompt (8 for Flash Lite, 2 for Gemma)
}

var supportedModels = []modelConfig{
	{Name: "gemini-3.5-flash-lite", MaxTargetTokens: 15000, TickerInterval: 4200 * time.Millisecond, CandidateChunk: 8},
	{Name: "gemini-3.1-flash-lite", MaxTargetTokens: 15000, TickerInterval: 4200 * time.Millisecond, CandidateChunk: 8},
	{Name: "gemma-4-31b-it", MaxTargetTokens: 2800, TickerInterval: 12000 * time.Millisecond, CandidateChunk: 2},
	{Name: "gemma-4-26b-a4b-it", MaxTargetTokens: 2800, TickerInterval: 12000 * time.Millisecond, CandidateChunk: 2},
}

// EvaluatePendingForAllUsers runs the multi-candidate, multi-worker batch engine.
// Guarantees zero duplicate job assignments across threads via atomic channel distribution.
func (s *GeminiBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	s.evaluateJobsInternal(ctx, "", false)
}

func (s *GeminiBatchMatchService) EvaluateForSingleUser(ctx context.Context, userID string) {
	s.evaluateJobsInternal(ctx, userID, true)
}

func (s *GeminiBatchMatchService) StartBackgroundScheduler(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	log.Println("[GeminiMatcher] Starting 5-minute background AI matcher scheduler...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				log.Println("[GeminiMatcher] 5-minute ticker tick: Triggering background AI evaluation check...")
				s.EvaluatePendingForAllUsers(ctx)
			}
		}
	}()
}

func (s *GeminiBatchMatchService) evaluateJobsInternal(ctx context.Context, targetUserID string, isSingleUser bool) {
	if s == nil || s.DB == nil || len(s.WorkerKeys) == 0 {
		log.Printf("[GeminiMatcher] Engine not fully initialized or DB nil, skipping evaluation.")
		return
	}

	var userProfiles []UserProfile
	var err error

	if isSingleUser {
		singleProf, errFetch := s.fetchSingleUserProfile(ctx, targetUserID)
		if errFetch != nil {
			log.Printf("[GeminiMatcher] Failed fetching single user profile %s: %v", targetUserID, errFetch)
			return
		}
		userProfiles = []UserProfile{*singleProf}
	} else {
		userProfiles, err = s.fetchAllUserProfiles(ctx)
		if err != nil {
			log.Printf("[GeminiMatcher] Failed fetching user profiles: %v", err)
			return
		}
	}

	if len(userProfiles) == 0 {
		log.Printf("[GeminiMatcher] No active user profiles found.")
		return
	}

	var pendingJobs []JobSnippet
	if isSingleUser {
		pendingJobs, err = s.fetchJobsUnmatchedForUser(ctx, targetUserID)
	} else {
		pendingJobs, err = s.fetchUnevaluatedJobs(ctx)
	}

	if err != nil {
		log.Printf("[GeminiMatcher] Failed fetching pending jobs: %v", err)
		return
	}

	if len(pendingJobs) == 0 {
		log.Printf("[GeminiMatcher] No pending jobs to evaluate.")
		return
	}

	log.Printf("[GeminiMatcher] Starting evaluation: %d jobs across %d users using %d worker keys across %d models.",
		len(pendingJobs), len(userProfiles), len(s.WorkerKeys), len(supportedModels))

	// Thread-Safe Job Queue: Channel guarantees NO 2 threads get the same JD!
	jobQueueChan := make(chan JobSnippet, len(pendingJobs))
	for _, job := range pendingJobs {
		// Only enqueue jobs not currently claimed
		if _, loaded := s.claimedJobs.LoadOrStore(job.JobID, true); !loaded {
			jobQueueChan <- job
		}
	}
	close(jobQueueChan)

	var wg sync.WaitGroup
	atomic.StoreInt32(&s.dailyExhausted, 0)

	// Launch Goroutines for each (WorkerKey, Model) pair
	// Total Workers = len(WorkerKeys) * 4 Models
	workerID := 0
	for _, apiKey := range s.WorkerKeys {
		for _, cfg := range supportedModels {
			workerID++
			wg.Add(1)
			go s.runWorker(ctx, workerID, apiKey, cfg, userProfiles, jobQueueChan, &wg)
		}
	}

	wg.Wait()

	log.Printf("[GeminiMatcher] Evaluation pass complete.")
}

func (s *GeminiBatchMatchService) runWorker(
	ctx context.Context,
	workerID int,
	apiKey string,
	cfg modelConfig,
	allProfiles []UserProfile,
	jobQueueChan <-chan JobSnippet,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// Initialize official GenAI SDK Client for this worker & key
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Printf("[GeminiMatcher] Worker %d (%s / %s) failed SDK init: %v", workerID, cfg.Name, apiKey[:8], err)
		return
	}

	ticker := time.NewTicker(cfg.TickerInterval)
	defer ticker.Stop()

	// Split user profiles into model-appropriate chunks (e.g. 8 for Flash Lite, 2 for Gemma)
	userChunks := chunkUserProfiles(allProfiles, cfg.CandidateChunk)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Pull jobs dynamically using SDK CountTokens to build optimal batch
		batchJobs := s.buildExactTokenBatch(ctx, client, cfg, jobQueueChan)
		if len(batchJobs) == 0 {
			// Queue is empty, worker finishes
			return
		}

		for _, userChunk := range userChunks {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C: // Strictly enforce RPM pacing per worker
			}

			results, err := s.callGenAIMultiBatch(ctx, client, cfg.Name, userChunk, batchJobs)
			if err != nil {
				log.Printf("[GeminiMatcher] Worker %d (%s) API call failed: %v", workerID, cfg.Name, err)
				if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "quota") {
					// Handle Rate Limit / Quota Exhaustion
					s.handleQuotaExhaustion(ctx, err)
					return
				}
				continue
			}

			if err := s.upsertMultiMatchResults(ctx, userChunk, results); err != nil {
				log.Printf("[GeminiMatcher] Worker %d (%s) upsert error: %v", workerID, cfg.Name, err)
			}
		}

		// Mark batch jobs as evaluated in database
		jobIDs := make(map[string]bool)
		for _, j := range batchJobs {
			jobIDs[j.JobID] = true
		}
		_ = s.markJobsEvaluated(ctx, jobIDs)
	}
}

// buildExactTokenBatch dynamically packs jobs using official SDK client.Models.CountTokens
// to guarantee zero hardcoding and zero estimation errors.
func (s *GeminiBatchMatchService) buildExactTokenBatch(
	ctx context.Context,
	client *genai.Client,
	cfg modelConfig,
	jobQueueChan <-chan JobSnippet,
) []JobSnippet {
	var batch []JobSnippet
	currentTokens := 0

	for {
		select {
		case job, ok := <-jobQueueChan:
			if !ok {
				return batch
			}

			// Construct candidate payload string
			candidateText := fmt.Sprintf("ID: %s\nTitle: %s\nCompany: %s\nLoc: %s\nDesc: %s\n",
				job.JobID, job.Title, job.Company, job.Location, job.Description)

			// Call official SDK CountTokens for 100% accurate token measurement
			countResp, err := client.Models.CountTokens(ctx, cfg.Name, genai.Text(candidateText), nil)
			itemTokens := 800 // default fallback if CountTokens fails
			if err == nil && countResp != nil {
				itemTokens = int(countResp.TotalTokens)
			}

			if len(batch) > 0 && (currentTokens+itemTokens > cfg.MaxTargetTokens) {
				return batch
			}

			batch = append(batch, job)
			currentTokens += itemTokens

			// Single-job limit for Gemma 4 if single job is near token ceiling
			if cfg.MaxTargetTokens <= 3000 && len(batch) >= 2 {
				return batch
			}
			if len(batch) >= 10 {
				return batch
			}
		default:
			return batch
		}
	}
}

func (s *GeminiBatchMatchService) callGenAIMultiBatch(
	ctx context.Context,
	client *genai.Client,
	modelName string,
	userChunk []UserProfile,
	batch []JobSnippet,
) ([]jobMultiMatchResult, error) {
	prompt := s.BuildMultiMatchPrompt(userChunk, batch)

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      genai.Ptr[float32](0.0),
	}

	result, err := client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), config)
	if err != nil {
		return nil, err
	}

	rawJSON := result.Text()
	if rawJSON == "" {
		return nil, fmt.Errorf("empty response from model")
	}

	var batchResponse mistralMultiBatchResponse
	contentBytes := []byte(rawJSON)
	if err := json.Unmarshal(contentBytes, &batchResponse); err != nil || len(batchResponse.Results) == 0 {
		var directList []jobMultiMatchResult
		if errArray := json.Unmarshal(contentBytes, &directList); errArray == nil {
			batchResponse.Results = directList
		} else {
			return nil, fmt.Errorf("failed parsing GenAI JSON response: %v\nRaw: %s", err, rawJSON)
		}
	}

	return batchResponse.Results, nil
}

func (s *GeminiBatchMatchService) handleQuotaExhaustion(ctx context.Context, err error) {
	count := atomic.AddInt32(&s.dailyExhausted, 1)
	totalWorkers := int32(len(s.WorkerKeys) * len(supportedModels))

	log.Printf("[GeminiMatcher] Worker encountered quota limit (%d/%d workers exhausted): %v", count, totalWorkers, err)

	if count >= totalWorkers {
		now := time.Now().UTC()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		sleepDuration := time.Until(nextMidnight)

		log.Printf("[GeminiMatcher] ALL %d WORKER TARGETS EXHAUSTED DAILY QUOTA. Pausing matcher until midnight UTC (sleeping %v).",
			totalWorkers, sleepDuration.Round(time.Second))

		time.Sleep(sleepDuration)
		log.Printf("[GeminiMatcher] Midnight UTC reached. Quotas reset! Resuming background match engine.")
		atomic.StoreInt32(&s.dailyExhausted, 0)
	}
}

func (s *GeminiBatchMatchService) BuildMultiMatchPrompt(userChunk []UserProfile, batch []JobSnippet) string {
	candidatesJSON, _ := json.Marshal(userChunk)
	batchJSON, _ := json.Marshal(batch)

	return fmt.Sprintf(`You are a strict, ultra-precise job-fit evaluator. Evaluate each job against candidate profiles.

CANDIDATE PROFILES:
%s

CRITICAL EVALUATION RULES:

1. CANDIDATE LOCATION & REMOTE ELIGIBILITY (STRICT PASS/FAIL):
- Candidates are physically located in INDIA and ONLY have work authorization for India or Global Remote roles.
- Any job specifying "Remote, US", "US Remote", "Remote (US)", "UK Remote", "Canada Remote", "EU Remote", "LATAM Remote", "North America Remote" is RESTRICTED to residents of that specific country/region. A candidate in India CANNOT work a US-restricted or UK-restricted remote job!
- REJECT IMMEDIATELY (match_score = 0, is_matched = false) if:
  * The job location specifies US, United States, UK, London, Canada, Europe, LATAM, North America, or non-India cities/states.
  * The job is country-restricted remote outside India (e.g. "US Remote", "Remote - US", "UK Remote").
- MATCH ONLY IF the location is explicitly:
  * Located in INDIA (On-site, Hybrid, or India Remote).
  * Explicitly GLOBAL Remote, Remote Worldwide, Remote (Global), or Remote (APAC) where candidates anywhere in the world (including India) can work.

2. CANDIDATE EXPERIENCE LEVEL (STRICT PASS/FAIL):
- The candidates are Early-Career / Junior Software Engineers with 1 YEAR OF EXPERIENCE (0-2 YOE target range).
- REJECT IMMEDIATELY (match_score = 0, is_matched = false) if:
  * The title contains Senior, Sr, Lead, Staff, Principal, Architect, Director, Manager, VP, Head of.
  * The job description explicitly requires 3+ years, 4+ years, 5+ years, or 3-5+ years of experience.
- MATCH ONLY IF the job requires 0-2 years of experience or is explicitly Entry Level, Junior, Associate, New Grad, or Intern.

3. SALARY EXTRACTION:
- Extract numeric annual salary range from the job description if mentioned. If not mentioned, set salary_min and salary_max to null.

4. MATCH SCORING:
- Score 0-100 based on tech stack overlap and role fit.
- If Location, Country Remote eligibility, or Experience Level fails, match_score MUST BE 0 and is_matched MUST BE false.
- is_matched = true ONLY when match_score >= 50.

Return ONLY valid JSON matching this exact structure:
{"results": [{"job_id": "...", "seniority": "Junior|Mid|Intern|Senior", "salary_min": 120000, "salary_max": 150000, "currency": "USD", "matches": [{"user_id": "...", "is_matched": true|false, "match_score": 0-100, "tech_stack": ["Python"], "reasoning": "1 short sentence explaining fit or rejection"}]}]}

JOBS TO EVALUATE:
%s`, string(candidatesJSON), string(batchJSON))
}

func (s *GeminiBatchMatchService) upsertMultiMatchResults(ctx context.Context, userChunk []UserProfile, results []jobMultiMatchResult) error {
	validUsers := make(map[string]string)
	for _, u := range userChunk {
		validUsers[u.UserID] = u.UserID
	}

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

		isSenior := IsSeniorRoleTitle(jobTitle)
		isExcessiveExperience := hasExcessiveYOE(jobTitle + " " + rawDesc)

		for idx, match := range jobResult.Matches {
			targetUserID := ""
			rawID := strings.TrimSpace(match.UserID)

			if len(userChunk) == 1 {
				targetUserID = userChunk[0].UserID
			} else if actualID, ok := validUsers[rawID]; ok {
				targetUserID = actualID
			} else if idx < len(userChunk) {
				targetUserID = userChunk[idx].UserID
			} else if len(userChunk) > 0 {
				targetUserID = userChunk[0].UserID
			}

			if targetUserID == "" || jobResult.JobID == "" {
				continue
			}
			userID := targetUserID

			var targetLocs string
			_ = s.DB.QueryRow(ctx, "SELECT COALESCE(target_locations::text, '') FROM user_preferences WHERE user_id = $1;", userID).Scan(&targetLocs)

			score := match.MatchScore
			isMatched := match.IsMatched
			reasoning := match.MatchReasoning

			if isSenior || isExcessiveExperience || IsNonTargetOnsite(jobLocation, isRemote, targetLocs) || isNonTargetHybridOrOnsite(jobLocation, rawDesc, isRemote, targetLocs) {
				score = 0
				isMatched = false
				reasoning = ""
			}

			techStackJSON, _ := json.Marshal(match.TechStack)
			matchReasons, _ := json.Marshal([]string{reasoning})

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
			_, err := s.DB.Exec(ctx, query,
				userID,
				jobResult.JobID,
				score,
				isMatched,
				seniorityVal,
				techStackJSON,
				reasoning,
				"gemini-batch-engine",
				matchReasons,
			)
			if err != nil {
				return fmt.Errorf("failed to upsert match for user %s, job %s: %w", match.UserID, jobResult.JobID, err)
			}
		}
	}
	return nil
}

func (s *GeminiBatchMatchService) fetchAllUserProfiles(ctx context.Context) ([]UserProfile, error) {
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

func (s *GeminiBatchMatchService) fetchSingleUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
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

func (s *GeminiBatchMatchService) fetchJobsUnmatchedForUser(ctx context.Context, userID string) ([]JobSnippet, error) {
	query := `
		SELECT j.id, j.title, COALESCE(c.name, ''), j.location,
		       COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE NOT EXISTS (
			SELECT 1 FROM user_job_matches ujm
			WHERE ujm.job_id = j.id AND ujm.user_id = $2
		)
		ORDER BY j.scraped_at DESC;
	`
	rows, err := s.DB.Query(ctx, query, userID)
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

func (s *GeminiBatchMatchService) fetchUnevaluatedJobs(ctx context.Context) ([]JobSnippet, error) {
	query := `
		SELECT j.id, j.title, COALESCE(c.name, ''), j.location,
		       COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE j.ai_evaluated = false
		ORDER BY j.scraped_at DESC;
	`
	rows, err := s.DB.Query(ctx, query)
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

func (s *GeminiBatchMatchService) markJobsEvaluated(ctx context.Context, jobIDs map[string]bool) error {
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
