// Package services provides business logic for AI job matching, CV parsing, and queue management.
package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultNvidiaNimEndpoint        = "https://integrate.api.nvidia.com/v1/chat/completions"
	defaultNvidiaNimModel           = "nvidia/nemotron-3-super-120b-a12b"
	backgroundWorkerBatchSize       = 20
	backgroundTickerIntervalSeconds = 15
	maxJobDescriptionLength         = 6000
	matchScoreMinThreshold          = 50
	maxTargetTokensPerPrompt        = 200000
)

// NvidiaNimService manages single-key AI job matching, CV parsing, and queue preemption for NVIDIA NIM API.
type NvidiaNimService struct {
	DB                     *pgxpool.Pool
	APIKey                 string
	Endpoint               string
	ModelName              string
	HTTPClient             *http.Client
	queueMutex             sync.RWMutex
	isQueuePaused          bool
	isEvaluationInProgress bool
	rateLimiterMutex       sync.Mutex
	lastRequestTime        time.Time
}

// UserProfileData contains parsed candidate background data for job matching prompts.
type UserProfileData struct {
	UserID             string   `json:"user_id"`
	Email              string   `json:"email"`
	ParsedBio          string   `json:"parsed_bio"`
	MasterCVText       string   `json:"master_cv_text"`
	PreferredRoles     []string `json:"preferred_roles"`
	PreferredLocations []string `json:"preferred_locations"`
	WorkModel          string   `json:"work_model"`
	ExperienceYears    int      `json:"experience_years"`
}

// JobSnippetData contains minimal job details sent for AI batch evaluation.
type JobSnippetData struct {
	JobID       string `json:"job_id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

type nvidiaMatchResult struct {
	JobID          string `json:"job_id"`
	UserID         string `json:"user_id"`
	MatchScore     int    `json:"match_score"`
	MatchReasoning string `json:"match_reasoning"`
	IsMatched      bool   `json:"is_matched"`
}

type nvidiaBatchResponse struct {
	Results []nvidiaMatchResult `json:"results"`
}

type nvidiaRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type nvidiaResponseFormat struct {
	Type string `json:"type"`
}

type nvidiaRequest struct {
	Model       string                 `json:"model"`
	Messages    []nvidiaRequestMessage `json:"messages"`
	Temperature float64                `json:"temperature"`
	TopP        float64                `json:"top_p"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Seed        int                    `json:"seed,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
}

type nvidiaTokenizeRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type nvidiaTokenizeResponse struct {
	Tokens []int `json:"tokens"`
	Count  int   `json:"count"`
}

type nvidiaAPIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type nvidiaStreamChunk struct {
	Choices []struct {
		Delta struct {
			ReasoningContent string `json:"reasoning_content"`
			Content          string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// NewNvidiaNimService initializes the single-key NVIDIA NIM service using provided environment values.
func NewNvidiaNimService(db *pgxpool.Pool, rawApiKey string) *NvidiaNimService {
	apiKey := strings.TrimSpace(rawApiKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("NVIDIA_API_KEY"))
	}
	modelName := strings.TrimSpace(os.Getenv("NVIDIA_MODEL"))
	if modelName == "" {
		modelName = defaultNvidiaNimModel
	}
	log.Printf("[NvidiaNimService] Initialized with single NVIDIA API Key (Model: %s)", modelName)
	return &NvidiaNimService{
		DB:         db,
		APIKey:     apiKey,
		Endpoint:   defaultNvidiaNimEndpoint,
		ModelName:  modelName,
		HTTPClient: &http.Client{Timeout: 0},
	}
}

// PauseQueue halts dispatching of new background worker batches during interactive MCP/Overleaf operations.
func (s *NvidiaNimService) PauseQueue() {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.isQueuePaused = true
	log.Println("[NvidiaNimService] Background queue paused for high-priority interactive call.")
}

// ResumeQueue unblocks background worker batch dispatching after interactive operations complete.
func (s *NvidiaNimService) ResumeQueue() {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.isQueuePaused = false
	log.Println("[NvidiaNimService] Background queue resumed.")
}

// IsQueuePaused returns true if background queue dispatching is currently suspended.
func (s *NvidiaNimService) IsQueuePaused() bool {
	s.queueMutex.RLock()
	defer s.queueMutex.RUnlock()
	return s.isQueuePaused
}

// StartBackgroundScheduler starts a 15-second ticker executing up to 5 concurrent worker goroutines.
func (s *NvidiaNimService) StartBackgroundScheduler(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(backgroundTickerIntervalSeconds) * time.Second)
	log.Printf("[NvidiaNimService] Started %d-second background ticker scheduler.", backgroundTickerIntervalSeconds)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if s.IsQueuePaused() {
					log.Println("[NvidiaNimService] Queue is paused, skipping background batch dispatch.")
					continue
				}
				s.EvaluatePendingForAllUsers(ctx)
			}
		}
	}()
}

// EvaluatePendingForAllUsers fetches unevaluated jobs, packs them into multi-job token batches using exact token counts (up to 500k tokens), and dispatches a 5-goroutine worker pool.
func (s *NvidiaNimService) EvaluatePendingForAllUsers(ctx context.Context) {
	s.queueMutex.Lock()
	if s.isEvaluationInProgress {
		s.queueMutex.Unlock()
		return
	}
	s.isEvaluationInProgress = true
	s.queueMutex.Unlock()

	defer func() {
		s.queueMutex.Lock()
		s.isEvaluationInProgress = false
		s.queueMutex.Unlock()
	}()

	userProfiles, errProfiles := s.fetchAllUserProfiles(ctx)
	if errProfiles != nil {
		log.Printf("[NvidiaNimService] Failed to fetch user profiles: %v", errProfiles)
		return
	}
	if len(userProfiles) == 0 {
		log.Println("[NvidiaNimService] No active user profiles found in database to evaluate against.")
		return
	}

	pendingJobs, errJobs := s.fetchUnevaluatedJobs(ctx)
	if errJobs != nil {
		log.Printf("[NvidiaNimService] Failed to fetch unevaluated jobs: %v", errJobs)
		return
	}
	if len(pendingJobs) == 0 {
		log.Println("[NvidiaNimService] Zero unevaluated jobs found in database.")
		return
	}

	log.Printf("[NvidiaNimService] Starting evaluation pass for %d active users across %d pending jobs...", len(userProfiles), len(pendingJobs))

	jobBatches := s.buildMultiJobTokenBatches(ctx, pendingJobs, maxTargetTokensPerPrompt)
	if len(jobBatches) == 0 {
		log.Println("[NvidiaNimService] No valid batches created from pending jobs.")
		return
	}

	limit := backgroundWorkerBatchSize
	if len(jobBatches) < limit {
		limit = len(jobBatches)
	}
	batchesToEvaluate := jobBatches[:limit]
	log.Printf("[NvidiaNimService] Dispatching %d parallel worker goroutines (Evaluating %d batches out of %d total packed)...", limit, limit, len(jobBatches))

	evaluatedJobIDs := make(map[string]bool)
	var syncMutex sync.Mutex
	var workerWaitGroup sync.WaitGroup

	for index := 0; index < len(batchesToEvaluate); index++ {
		workerWaitGroup.Add(1)
		singleBatch := batchesToEvaluate[index]
		workerID := index + 1

		go func(batch []JobSnippetData, id int) {
			defer workerWaitGroup.Done()
			log.Printf("[NvidiaNimWorker-%d] Starting evaluation for batch of %d jobs...", id, len(batch))
			success := s.evaluateJobBatchWithBackoff(ctx, userProfiles, batch, evaluatedJobIDs, &syncMutex, id)
			if !success {
				log.Printf("[NvidiaNimWorker-%d] Batch of %d jobs postponed for next retry pass.", id, len(batch))
			} else {
				log.Printf("[NvidiaNimWorker-%d] Successfully evaluated batch of %d jobs.", id, len(batch))
			}
		}(singleBatch, workerID)
	}

	workerWaitGroup.Wait()

	if len(evaluatedJobIDs) > 0 {
		markErr := s.markJobsEvaluated(ctx, evaluatedJobIDs)
		if markErr != nil {
			log.Printf("[NvidiaNimService] Failed marking jobs as evaluated: %v", markErr)
		} else {
			log.Printf("[NvidiaNimService] Pass complete: Marked %d jobs as evaluated in database across 5 workers.", len(evaluatedJobIDs))
		}
	}
}

// EvaluateForSingleUser evaluates unmatched jobs for a specific user upon initial onboarding or bio update.
func (s *NvidiaNimService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	profile, errProfile := s.fetchSingleUserProfile(ctx, targetUserID)
	if errProfile != nil {
		log.Printf("[NvidiaNimService] Failed to fetch profile for user %s: %v", targetUserID, errProfile)
		return
	}

	unmatchedJobs, errJobs := s.fetchJobsUnmatchedForUser(ctx, targetUserID)
	if errJobs != nil {
		log.Printf("[NvidiaNimService] Failed to fetch unmatched jobs for user %s: %v", targetUserID, errJobs)
		return
	}
	if len(unmatchedJobs) == 0 {
		return
	}

	jobBatches := s.buildMultiJobTokenBatches(ctx, unmatchedJobs, maxTargetTokensPerPrompt)
	evaluatedJobIDs := make(map[string]bool)
	var syncMutex sync.Mutex
	var workerWaitGroup sync.WaitGroup

	for index := 0; index < len(jobBatches); index++ {
		if index > 0 && index%backgroundWorkerBatchSize == 0 {
			workerWaitGroup.Wait()
			time.Sleep(2 * time.Second)
		}

		workerWaitGroup.Add(1)
		singleBatch := jobBatches[index]

		go func(batch []JobSnippetData, id int) {
			defer workerWaitGroup.Done()
			s.evaluateJobBatchWithBackoff(ctx, []UserProfileData{*profile}, batch, evaluatedJobIDs, &syncMutex, id)
		}(singleBatch, index+1)
	}

	workerWaitGroup.Wait()
}

// GenerateCompletion executes a synchronous API request for CV parsing or Overleaf interactive tasks.
func (s *NvidiaNimService) GenerateCompletion(ctx context.Context, promptContent string, systemInstruction string) (string, error) {
	s.PauseQueue()
	defer s.ResumeQueue()

	messages := []nvidiaRequestMessage{}
	if systemInstruction != "" {
		messages = append(messages, nvidiaRequestMessage{Role: "system", Content: systemInstruction})
	}
	messages = append(messages, nvidiaRequestMessage{Role: "user", Content: promptContent})

	payload := nvidiaRequest{
		Model:       s.ModelName,
		Messages:    messages,
		Temperature: 0.3,
		TopP:        0.95,
		Seed:        42,
	}

	rawResponseBody, errCall := s.executeNvidiaAPIWithRetry(ctx, payload, 0)
	if errCall != nil {
		return "", errCall
	}

	var parsedResponse nvidiaAPIResponse
	if errUnmarshal := json.Unmarshal(rawResponseBody, &parsedResponse); errUnmarshal != nil {
		return "", fmt.Errorf("failed unmarshaling NVIDIA response: %w", errUnmarshal)
	}

	if len(parsedResponse.Choices) == 0 {
		return "", fmt.Errorf("empty choice array returned from NVIDIA API")
	}

	return parsedResponse.Choices[0].Message.Content, nil
}

// CountTokens requests exact token count for prompt text via NVIDIA NIM tokenize API with subword tokenization fallback.
func (s *NvidiaNimService) CountTokens(ctx context.Context, text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	tokenizeEndpoint := strings.Replace(s.Endpoint, "/chat/completions", "/tokenize", 1)
	payload := nvidiaTokenizeRequest{
		Model:  s.ModelName,
		Prompt: text,
	}

	jsonBytes, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		return s.calculateExactSubwordTokens(text), nil
	}

	httpRequest, errReq := http.NewRequestWithContext(ctx, http.MethodPost, tokenizeEndpoint, bytes.NewBuffer(jsonBytes))
	if errReq != nil {
		return s.calculateExactSubwordTokens(text), nil
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	if s.APIKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+s.APIKey)
	}

	httpResponse, errDo := s.HTTPClient.Do(httpRequest)
	if errDo != nil {
		return s.calculateExactSubwordTokens(text), nil
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return s.calculateExactSubwordTokens(text), nil
	}

	bodyBytes, errRead := io.ReadAll(httpResponse.Body)
	if errRead != nil {
		return s.calculateExactSubwordTokens(text), nil
	}

	var parsedResp nvidiaTokenizeResponse
	if errUnmarshal := json.Unmarshal(bodyBytes, &parsedResp); errUnmarshal == nil {
		if parsedResp.Count > 0 {
			return parsedResp.Count, nil
		}
		if len(parsedResp.Tokens) > 0 {
			return len(parsedResp.Tokens), nil
		}
	}

	return s.calculateExactSubwordTokens(text), nil
}

func (s *NvidiaNimService) calculateExactSubwordTokens(text string) int {
	words := strings.Fields(text)
	totalTokens := 0
	for _, word := range words {
		subwords := len(word) / 3
		if subwords < 1 {
			subwords = 1
		}
		totalTokens += subwords
	}
	return totalTokens
}

func (s *NvidiaNimService) buildMultiJobTokenBatches(ctx context.Context, allJobs []JobSnippetData, targetTokenBudget int) [][]JobSnippetData {
	var resultBatches [][]JobSnippetData
	var currentBatch []JobSnippetData
	currentTokens := 0

	for _, job := range allJobs {
		snippetText := fmt.Sprintf("ID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription: %s\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))

		itemTokens := s.calculateExactSubwordTokens(snippetText)

		if len(currentBatch) > 0 && (currentTokens+itemTokens > targetTokenBudget) {
			resultBatches = append(resultBatches, currentBatch)
			currentBatch = []JobSnippetData{}
			currentTokens = 0
		}

		currentBatch = append(currentBatch, job)
		currentTokens += itemTokens
	}

	if len(currentBatch) > 0 {
		resultBatches = append(resultBatches, currentBatch)
	}

	return resultBatches
}

func (s *NvidiaNimService) evaluateJobBatchWithBackoff(
	ctx context.Context,
	userProfiles []UserProfileData,
	batch []JobSnippetData,
	evaluatedJobIDs map[string]bool,
	syncMutex *sync.Mutex,
	workerID int,
) bool {
	promptText := s.buildMultiJobPrompt(userProfiles, batch)
	payload := nvidiaRequest{
		Model:       s.ModelName,
		Messages:    []nvidiaRequestMessage{{Role: "user", Content: promptText}},
		Temperature: 1.0,
		TopP:        1.0,
		MaxTokens:   500000,
		Seed:        42,
		Stream:      true,
	}

	responseBytes, errCall := s.executeNvidiaAPIWithRetry(ctx, payload, workerID)
	if errCall != nil {
		log.Printf("[Worker-%d] API call failed for batch of %d jobs: %v", workerID, len(batch), errCall)
		return false
	}

	var parsedResponse nvidiaAPIResponse
	if errUnmarshal := json.Unmarshal(responseBytes, &parsedResponse); errUnmarshal != nil {
		log.Printf("[Worker-%d] Unmarshal error for batch of %d jobs: %v", workerID, len(batch), errUnmarshal)
		return false
	}

	if len(parsedResponse.Choices) == 0 {
		log.Printf("[Worker-%d] Empty choices array for batch of %d jobs", workerID, len(batch))
		return false
	}

	rawText := parsedResponse.Choices[0].Message.Content
	cleanJSON := sanitizeJSONResponse(rawText)

	var batchResult nvidiaBatchResponse
	if errJSON := json.Unmarshal([]byte(cleanJSON), &batchResult); errJSON != nil {
		log.Printf("[Worker-%d] JSON parse error for batch of %d jobs: %v", workerID, len(batch), errJSON)
		return false
	}

	for _, res := range batchResult.Results {
		if res.JobID == "" || res.UserID == "" || !isValidUUIDString(res.JobID) || !isValidUUIDString(res.UserID) {
			continue
		}
		isMatch := res.MatchScore >= matchScoreMinThreshold
		upsertErr := s.upsertUserMatch(ctx, res.UserID, res.JobID, res.MatchScore, res.MatchReasoning, isMatch)
		if upsertErr != nil {
			log.Printf("[Worker-%d] Upsert error for user %s job %s: %v", workerID, res.UserID, res.JobID, upsertErr)
		}
		syncMutex.Lock()
		evaluatedJobIDs[res.JobID] = true
		syncMutex.Unlock()
	}

	return true
}

func (s *NvidiaNimService) executeNvidiaAPIWithRetry(ctx context.Context, payload nvidiaRequest, workerID int) ([]byte, error) {
	if s.APIKey == "" {
		return nil, fmt.Errorf("NVIDIA_API_KEY environment variable is not configured")
	}

	jsonBytes, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		return nil, fmt.Errorf("failed marshaling request payload: %w", errMarshal)
	}

	maxAttempts := 3
	backoffDuration := 3 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		s.enforceMinimumRequestSpacing()

		log.Printf("[Worker-%d] Posting request to %s (Model: %s, Payload: %d bytes, Attempt: %d/%d)...",
			workerID, s.Endpoint, payload.Model, len(jsonBytes), attempt, maxAttempts)
		requestStart := time.Now()

		httpRequest, errReq := http.NewRequestWithContext(ctx, http.MethodPost, s.Endpoint, bytes.NewBuffer(jsonBytes))
		if errReq != nil {
			return nil, fmt.Errorf("failed creating HTTP request: %w", errReq)
		}

		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("Accept", "application/json")
		httpRequest.Header.Set("Authorization", "Bearer "+s.APIKey)

		httpResponse, errDo := s.HTTPClient.Do(httpRequest)
		if errDo != nil {
			log.Printf("[Worker-%d] Request error after %v: %v", workerID, time.Since(requestStart), errDo)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("HTTP request failed after %d attempts: %w", maxAttempts, errDo)
			}
			time.Sleep(backoffDuration)
			backoffDuration *= 2
			continue
		}

		if payload.Stream {
			if httpResponse.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(httpResponse.Body)
				httpResponse.Body.Close()
				return nil, fmt.Errorf("NVIDIA API error HTTP %d: %s", httpResponse.StatusCode, string(bodyBytes))
			}

			scanner := bufio.NewScanner(httpResponse.Body)
			var accumulatedBuilder strings.Builder
			reasoningTokenCount := 0
			firstTokenReceived := false
			lastLogTime := time.Now()
			lastChunkTime := time.Now()

			for scanner.Scan() {
				if time.Since(lastChunkTime) > 5*time.Minute {
					log.Printf("[Worker-%d] Stream idle for 5 minutes with zero tokens received. Terminating connection.", workerID)
					httpResponse.Body.Close()
					return nil, fmt.Errorf("streaming connection idle timeout: no data received for 5 minutes")
				}

				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, ":") {
					continue
				}

				if strings.HasPrefix(line, "data: ") {
					dataStr := strings.TrimPrefix(line, "data: ")
					if dataStr == "[DONE]" {
						break
					}

					var chunk nvidiaStreamChunk
					if errJson := json.Unmarshal([]byte(dataStr), &chunk); errJson == nil {
						if len(chunk.Choices) > 0 {
							delta := chunk.Choices[0].Delta
							if delta.ReasoningContent != "" || delta.Content != "" {
								lastChunkTime = time.Now()
							}
							if delta.ReasoningContent != "" {
								reasoningTokenCount += len(delta.ReasoningContent)
								if !firstTokenReceived {
									firstTokenReceived = true
									log.Printf("[Worker-%d] Streaming first reasoning token received after %v", workerID, time.Since(requestStart))
								}
							}
							if delta.Content != "" {
								if !firstTokenReceived {
									firstTokenReceived = true
									log.Printf("[Worker-%d] Streaming first content token received after %v", workerID, time.Since(requestStart))
								}
								accumulatedBuilder.WriteString(delta.Content)
							}

							if time.Since(lastLogTime) >= 10*time.Second {
								log.Printf("[Worker-%d] Live Stream: Reasoning bytes: %d, Response bytes: %d (Elapsed: %v)",
									workerID, reasoningTokenCount, accumulatedBuilder.Len(), time.Since(requestStart).Truncate(time.Second))
								lastLogTime = time.Now()
							}
						}
					}
				}
			}
			httpResponse.Body.Close()

			accumulatedJSON := accumulatedBuilder.String()
			log.Printf("[Worker-%d] Streaming finished in %v (Reasoning bytes: %d, Response size: %d bytes)",
				workerID, time.Since(requestStart).Truncate(time.Millisecond), reasoningTokenCount, len(accumulatedJSON))

			syntheticResponse := nvidiaAPIResponse{}
			syntheticResponse.Choices = make([]struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}, 1)
			syntheticResponse.Choices[0].Message.Content = accumulatedJSON

			return json.Marshal(syntheticResponse)
		}

		bodyBytes, errRead := io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()
		if errRead != nil {
			return nil, fmt.Errorf("failed reading response body: %w", errRead)
		}

		log.Printf("[NvidiaNimHTTP] Received HTTP %d in %v (Response size: %d bytes)",
			httpResponse.StatusCode, time.Since(requestStart), len(bodyBytes))

		if httpResponse.StatusCode == http.StatusTooManyRequests {
			log.Printf("[NvidiaNimService] Rate limited (429). Retrying in %v...", backoffDuration)
			time.Sleep(backoffDuration)
			backoffDuration *= 2
			continue
		}

		if httpResponse.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("NVIDIA API error HTTP %d: %s", httpResponse.StatusCode, string(bodyBytes))
		}

		return bodyBytes, nil
	}

	return nil, fmt.Errorf("NVIDIA API call exceeded max retries")
}

func (s *NvidiaNimService) enforceMinimumRequestSpacing() {
	s.rateLimiterMutex.Lock()
	defer s.rateLimiterMutex.Unlock()

	minimumInterval := 500 * time.Millisecond
	elapsed := time.Since(s.lastRequestTime)
	if elapsed < minimumInterval {
		time.Sleep(minimumInterval - elapsed)
	}
	s.lastRequestTime = time.Now()
}

func (s *NvidiaNimService) buildMultiJobPrompt(userProfiles []UserProfileData, jobsBatch []JobSnippetData) string {
	var builder strings.Builder

	builder.WriteString("Evaluate the following batch of job listings against candidate profiles and assign match scores (0 to 100).\n\n")
	builder.WriteString("STRICT SCORING RULES YOU MUST ENFORCE WITHOUT EXCEPTION:\n")
	builder.WriteString("1. LOCATION & WORK AUTHORIZATION MISMATCH:\n")
	builder.WriteString("   - Candidate target locations are India (Onsite/Hybrid), India (Remote), or Global Remote.\n")
	builder.WriteString("   - If job is US Onsite, US Hybrid, or US-only Remote (e.g. Remote Texas, Remote SF, Remote NYC, US-based), assign MAX MATCH SCORE OF 0 to 15.\n")
	builder.WriteString("   - Candidates in India CANNOT work US-restricted remote or US onsite jobs.\n\n")
	builder.WriteString("2. SENIORITY & EXPERIENCE GAP MISMATCH:\n")
	builder.WriteString("   - Candidate is early-career / junior engineer (~1 Year of Experience).\n")
	builder.WriteString("   - If job title contains 'Senior', 'Sr', 'Staff', 'Lead', 'Principal', 'Architect', 'Director', 'Head', or requires 5+ years of experience, assign MAX MATCH SCORE OF 15 to 30.\n")
	builder.WriteString("   - Early career candidates DO NOT match Senior/Staff/Architect positions.\n\n")
	builder.WriteString("3. HIGH MATCH QUALIFICATION (80-100):\n")
	builder.WriteString("   - High scores (80 to 100) are reserved EXCLUSIVELY for Entry-Level, Junior, Associate, or Mid-Level Software Engineer, Full Stack, Backend, Frontend, or AI/ML Engineer roles matching candidate's stack in India or Global Remote.\n\n")

	builder.WriteString("Return ONLY a raw JSON object formatted as follows:\n")
	builder.WriteString("{\n  \"results\": [\n    {\n      \"job_id\": \"string\",\n      \"user_id\": \"string\",\n      \"match_score\": 85,\n      \"match_reasoning\": \"short bullet reasoning\",\n      \"is_matched\": true\n    }\n  ]\n}\n\n")

	builder.WriteString("### Candidate Profiles\n")
	for _, profile := range userProfiles {
		combinedProfileText := profile.ParsedBio
		if profile.MasterCVText != "" {
			combinedProfileText += "\n\nMaster CV / Full Experience Context:\n" + profile.MasterCVText
		}
		fmt.Fprintf(&builder, "User ID: %s\nProfile & Resume Context:\n%s\nPreferred Roles: %s\nPreferred Locations: %s\nWork Model: %s\n\n",
			profile.UserID, combinedProfileText, strings.Join(profile.PreferredRoles, ", "), strings.Join(profile.PreferredLocations, ", "), profile.WorkModel)
	}

	builder.WriteString("### Job Listings to Evaluate\n")
	for _, job := range jobsBatch {
		fmt.Fprintf(&builder, "---\nID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription:\n%s\n\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))
	}

	return builder.String()
}

func (s *NvidiaNimService) fetchAllUserProfiles(ctx context.Context) ([]UserProfileData, error) {
	sqlQuery := `
		SELECT u.id, u.primary_email, COALESCE(up.bio_experience_text, ''), COALESCE(up.master_cv_text, ''), COALESCE(up.target_roles, '[]'),
		       COALESCE(up.target_locations, '[]'), COALESCE(up.work_models->>0, ''), 0
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id;
	`
	rows, errQuery := s.DB.Query(ctx, sqlQuery)
	if errQuery != nil {
		return nil, errQuery
	}
	defer rows.Close()

	var profiles []UserProfileData
	for rows.Next() {
		var item UserProfileData
		scanErr := rows.Scan(&item.UserID, &item.Email, &item.ParsedBio, &item.MasterCVText, &item.PreferredRoles, &item.PreferredLocations, &item.WorkModel, &item.ExperienceYears)
		if scanErr == nil {
			profiles = append(profiles, item)
		}
	}
	return profiles, nil
}

func (s *NvidiaNimService) fetchSingleUserProfile(ctx context.Context, targetUserID string) (*UserProfileData, error) {
	sqlQuery := `
		SELECT u.id, u.primary_email, COALESCE(up.bio_experience_text, ''), COALESCE(up.master_cv_text, ''), COALESCE(up.target_roles, '[]'),
		       COALESCE(up.target_locations, '[]'), COALESCE(up.work_models->>0, ''), 0
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.id = $1;
	`
	var item UserProfileData
	scanErr := s.DB.QueryRow(ctx, sqlQuery, targetUserID).Scan(&item.UserID, &item.Email, &item.ParsedBio, &item.MasterCVText, &item.PreferredRoles, &item.PreferredLocations, &item.WorkModel, &item.ExperienceYears)
	if scanErr != nil {
		return nil, scanErr
	}
	return &item, nil
}

func (s *NvidiaNimService) fetchUnevaluatedJobs(ctx context.Context) ([]JobSnippetData, error) {
	sqlQuery := `
		SELECT j.id, j.title, COALESCE(c.name, ''), COALESCE(j.location, ''), COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE j.ai_evaluated = false
		ORDER BY j.scraped_at DESC;
	`
	rows, errQuery := s.DB.Query(ctx, sqlQuery)
	if errQuery != nil {
		return nil, errQuery
	}
	defer rows.Close()

	var snippets []JobSnippetData
	for rows.Next() {
		var snippet JobSnippetData
		scanErr := rows.Scan(&snippet.JobID, &snippet.Title, &snippet.Company, &snippet.Location, &snippet.Description)
		if scanErr == nil {
			snippets = append(snippets, snippet)
		}
	}
	return snippets, nil
}

func (s *NvidiaNimService) fetchJobsUnmatchedForUser(ctx context.Context, targetUserID string) ([]JobSnippetData, error) {
	sqlQuery := `
		SELECT j.id, j.title, COALESCE(c.name, ''), COALESCE(j.location, ''), COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE NOT EXISTS (
			SELECT 1 FROM user_job_matches ujm
			WHERE ujm.job_id = j.id AND ujm.user_id = $1
		)
		ORDER BY j.scraped_at DESC;
	`
	rows, errQuery := s.DB.Query(ctx, sqlQuery, targetUserID)
	if errQuery != nil {
		return nil, errQuery
	}
	defer rows.Close()

	var snippets []JobSnippetData
	for rows.Next() {
		var snippet JobSnippetData
		scanErr := rows.Scan(&snippet.JobID, &snippet.Title, &snippet.Company, &snippet.Location, &snippet.Description)
		if scanErr == nil {
			snippets = append(snippets, snippet)
		}
	}
	return snippets, nil
}

func (s *NvidiaNimService) upsertUserMatch(ctx context.Context, userID, jobID string, score int, reasoning string, isMatch bool) error {
	sqlQuery := `
		INSERT INTO user_job_matches (user_id, job_id, match_score, match_reasoning, is_ai_matched, ai_model, evaluated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, job_id) DO UPDATE SET
			match_score = EXCLUDED.match_score,
			match_reasoning = EXCLUDED.match_reasoning,
			is_ai_matched = EXCLUDED.is_ai_matched,
			ai_model = EXCLUDED.ai_model,
			evaluated_at = NOW();
	`
	_, errExec := s.DB.Exec(ctx, sqlQuery, userID, jobID, score, reasoning, isMatch, s.ModelName)
	return errExec
}

func (s *NvidiaNimService) markJobsEvaluated(ctx context.Context, jobIDsMap map[string]bool) error {
	if len(jobIDsMap) == 0 {
		return nil
	}

	keysList := make([]string, 0, len(jobIDsMap))
	for keyID := range jobIDsMap {
		keysList = append(keysList, keyID)
	}

	sqlQuery := `
		UPDATE jobs
		SET ai_evaluated = true
		WHERE id = ANY($1);
	`
	_, errExec := s.DB.Exec(ctx, sqlQuery, keysList)
	return errExec
}

func sanitizeJSONResponse(textContent string) string {
	clean := strings.TrimSpace(textContent)
	if strings.HasPrefix(clean, "```json") {
		clean = strings.TrimPrefix(clean, "```json")
		clean = strings.TrimSuffix(clean, "```")
	} else if strings.HasPrefix(clean, "```") {
		clean = strings.TrimPrefix(clean, "```")
		clean = strings.TrimSuffix(clean, "```")
	}
	return strings.TrimSpace(clean)
}

func truncateTextString(sourceText string, maxLength int) string {
	if len(sourceText) <= maxLength {
		return sourceText
	}
	return sourceText[:maxLength]
}

func isValidUUIDString(u string) bool {
	if len(u) != 36 {
		return false
	}
	for index, char := range u {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
		} else {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	return true
}

func IsValidUUIDStringForTest(u string) bool {
	return isValidUUIDString(u)
}
