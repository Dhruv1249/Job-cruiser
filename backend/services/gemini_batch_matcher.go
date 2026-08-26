// Package services provides business logic for AI job matching, CV parsing, and queue management.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/genai"
)

// GeminiMatchResult encapsulates an evaluated job match output for a specific candidate.
type GeminiMatchResult struct {
	JobID               string `json:"job_id"`
	UserID              string `json:"user_id"`
	MatchScore          int    `json:"match_score"`
	MatchReasoning      string `json:"match_reasoning"`
	InferredRequiredYoE int    `json:"inferred_required_yoe"`
	IsMatched           bool   `json:"is_matched"`
}

// GeminiBatchResponse defines the structured JSON array envelope emitted by Gemini models.
type GeminiBatchResponse struct {
	Results []GeminiMatchResult `json:"results"`
}

// GeminiBatchJobMatchSchemaJSON defines the structured schema for Gemini batch match evaluations.
var GeminiBatchJobMatchSchemaJSON = `{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "job_id": {"type": "string"},
          "user_id": {"type": "string"},
          "match_score": {"type": "integer"},
          "match_reasoning": {"type": "string"},
          "inferred_required_yoe": {"type": "integer"},
          "is_matched": {"type": "boolean"}
        },
        "required": ["job_id", "user_id", "match_score", "match_reasoning", "inferred_required_yoe", "is_matched"],
        "propertyOrdering": ["job_id", "user_id", "match_score", "match_reasoning", "inferred_required_yoe", "is_matched"]
      }
    }
  },
  "required": ["results"],
  "propertyOrdering": ["results"]
}`

// GeminiBatchMatchService manages batch AI job evaluations sequentially across alternating Gemini models.
type GeminiBatchMatchService struct {
	DB                     *pgxpool.Pool
	APIKey                 string
	BaseURL                string
	HTTPClient             *http.Client
	Models                 []string
	currentModelIndex      atomic.Int64
	TargetTokenBudget      int
	RateLimitInterval      time.Duration
	lastRequestTime        time.Time
	rateLimitMutex         sync.Mutex
	queueMutex                    sync.RWMutex
	isQueuePaused                 bool
	isEvaluationInProgress        bool
	isPipelinePermanentlyStopped  bool
	OnPipelineShutdown            func()
	modelErrorMutex               sync.RWMutex
	modelConsecutiveErrors        map[string]int
	disabledModels                map[string]bool
}

const maxConsecutiveModelErrors = 5

// NewGeminiBatchMatchService constructs a new GeminiBatchMatchService configured via environment variables.
func NewGeminiBatchMatchService(databasePool *pgxpool.Pool, apiKey string) *GeminiBatchMatchService {
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}

	configuredModels := parseConfiguredGeminiModels()
	tokenBudget := parseConfiguredTokenBudget()
	rateLimitDuration := parseConfiguredRateLimitDuration()

	return &GeminiBatchMatchService{
		DB:                     databasePool,
		APIKey:                 apiKey,
		HTTPClient:             &http.Client{Timeout: 15 * time.Minute},
		Models:                 configuredModels,
		TargetTokenBudget:      tokenBudget,
		RateLimitInterval:      rateLimitDuration,
		modelConsecutiveErrors: make(map[string]int),
		disabledModels:         make(map[string]bool),
	}
}

func parseConfiguredGeminiModels() []string {
	environmentModels := strings.TrimSpace(os.Getenv("GEMINI_BATCH_MODELS"))
	if environmentModels == "" {
		environmentModels = strings.TrimSpace(os.Getenv("GEMINI_MODELS"))
	}
	if environmentModels == "" {
		environmentModels = strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	}

	if environmentModels == "" {
		return nil
	}

	rawParts := strings.Split(environmentModels, ",")
	var parsedModels []string
	for _, rawPart := range rawParts {
		trimmedModel := strings.TrimSpace(rawPart)
		if trimmedModel != "" {
			parsedModels = append(parsedModels, trimmedModel)
		}
	}
	return parsedModels
}

func parseConfiguredTokenBudget() int {
	rawBudget := strings.TrimSpace(os.Getenv("GEMINI_BATCH_TOKEN_BUDGET"))
	if rawBudget != "" {
		if parsedBudget, err := strconv.Atoi(rawBudget); err == nil && parsedBudget > 0 {
			return parsedBudget
		}
	}
	return 0
}

func parseConfiguredRateLimitDuration() time.Duration {
	rawSeconds := strings.TrimSpace(os.Getenv("GEMINI_BATCH_RATE_LIMIT_SECONDS"))
	if rawSeconds != "" {
		if parsedSeconds, err := strconv.Atoi(rawSeconds); err == nil && parsedSeconds > 0 {
			return time.Duration(parsedSeconds) * time.Second
		}
	}
	return 0
}

// PauseQueue halts background batch evaluation dispatches.
func (s *GeminiBatchMatchService) PauseQueue() {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.isQueuePaused = true
	log.Println("[GeminiBatchMatchService] Background queue paused.")
}

// ResumeQueue unblocks background batch evaluation dispatches.
func (s *GeminiBatchMatchService) ResumeQueue() {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.isQueuePaused = false
	log.Println("[GeminiBatchMatchService] Background queue resumed.")
}

// IsQueuePaused returns whether background batch evaluation is paused.
func (s *GeminiBatchMatchService) IsQueuePaused() bool {
	s.queueMutex.RLock()
	defer s.queueMutex.RUnlock()
	return s.isQueuePaused
}

// StartBackgroundScheduler starts a periodic ticker triggering batch evaluations every 10 minutes.
func (s *GeminiBatchMatchService) StartBackgroundScheduler(ctx context.Context) {
	tickerDuration := 10 * time.Minute
	ticker := time.NewTicker(tickerDuration)
	log.Printf("[GeminiBatchMatchService] Started %v background ticker scheduler.", tickerDuration)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if s.IsPipelinePermanentlyStopped() {
					log.Println("[GeminiBatchMatchService] AI matching pipeline is permanently stopped, skipping background batch dispatch.")
					continue
				}
				if s.IsQueuePaused() {
					log.Println("[GeminiBatchMatchService] Queue is paused, skipping background batch dispatch.")
					continue
				}
				s.EvaluatePendingForAllUsers(ctx)
			}
		}
	}()
}

// EvaluatePendingForAllUsers fetches unevaluated jobs and evaluates them in sequential batches.
func (s *GeminiBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	s.queueMutex.Lock()
	if s.isPipelinePermanentlyStopped {
		s.queueMutex.Unlock()
		log.Println("[GeminiBatchMatchService] AI matching pipeline is permanently stopped. Skipping evaluation.")
		return
	}
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

	if s.DB == nil {
		return
	}

	userProfiles, errProfiles := s.fetchAllUserProfiles(ctx)
	if errProfiles != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch user profiles: %v", errProfiles)
		return
	}
	if len(userProfiles) == 0 {
		log.Println("[GeminiBatchMatchService] No active user profiles found.")
		return
	}

	pendingJobs, errJobs := s.fetchUnevaluatedJobs(ctx)
	if errJobs != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch unevaluated jobs: %v", errJobs)
		return
	}
	if len(pendingJobs) == 0 {
		log.Println("[GeminiBatchMatchService] Zero unevaluated jobs found.")
		return
	}

	jobBatches := s.buildMultiJobTokenBatches(ctx, pendingJobs, s.TargetTokenBudget)
	if len(jobBatches) == 0 {
		return
	}

	log.Printf("[GeminiBatchMatchService] Dispatching sequential Gemini pipeline across %d initial batches...", len(jobBatches))

	batchIteration := 0
	for len(jobBatches) > 0 {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.IsPipelinePermanentlyStopped() {
			log.Println("[GeminiBatchMatchService] AI matching pipeline is permanently stopped. Aborting remaining batches.")
			return
		}

		singleBatch := jobBatches[0]
		jobBatches = jobBatches[1:]
		batchIteration++

		selectedModel, errModel := s.getNextModelName()
		if errModel != nil {
			log.Printf("[GeminiBatchMatchService] Configuration/Shutdown error: %v", errModel)
			return
		}

		log.Printf("[GeminiBatchMatchService] Processing batch %d (%d jobs, %d remaining in queue) using model: %s...",
			batchIteration, len(singleBatch), len(jobBatches), selectedModel)

		s.enforceRateLimitPacing()

		currentBatchEvaluatedIDs := make(map[string]bool)
		success := s.evaluateJobBatch(ctx, userProfiles, singleBatch, currentBatchEvaluatedIDs, selectedModel)
		if !success {
			log.Printf("[GeminiBatchMatchService] Batch of %d jobs failed with model %s — re-queuing batch at the end of queue.", len(singleBatch), selectedModel)
			if !s.IsPipelinePermanentlyStopped() {
				jobBatches = append(jobBatches, singleBatch)
			}
		} else {
			if len(currentBatchEvaluatedIDs) > 0 {
				markErr := s.markJobsEvaluated(ctx, currentBatchEvaluatedIDs)
				if markErr != nil {
					log.Printf("[GeminiBatchMatchService] Failed marking batch of %d jobs as evaluated: %v", len(currentBatchEvaluatedIDs), markErr)
				} else {
					log.Printf("[GeminiBatchMatchService] Live progress: marked %d jobs as evaluated in database.", len(currentBatchEvaluatedIDs))
				}
			}
		}
	}
}

// EvaluateForSingleUser evaluates unmatched jobs for a specific user.
func (s *GeminiBatchMatchService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	s.queueMutex.RLock()
	if s.isPipelinePermanentlyStopped {
		s.queueMutex.RUnlock()
		log.Printf("[GeminiBatchMatchService] AI matching pipeline is permanently stopped. Skipping evaluation for user %s.", targetUserID)
		return
	}
	s.queueMutex.RUnlock()

	if s.DB == nil {
		return
	}

	profile, errProfile := s.fetchSingleUserProfile(ctx, targetUserID)
	if errProfile != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch profile for user %s: %v", targetUserID, errProfile)
		return
	}

	unmatchedJobs, errJobs := s.fetchJobsUnmatchedForUser(ctx, targetUserID)
	if errJobs != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch unmatched jobs for user %s: %v", targetUserID, errJobs)
		return
	}
	if len(unmatchedJobs) == 0 {
		return
	}

	jobBatches := s.buildMultiJobTokenBatches(ctx, unmatchedJobs, s.TargetTokenBudget)
	userProfiles := []UserProfileData{*profile}

	batchIteration := 0
	for len(jobBatches) > 0 {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.IsPipelinePermanentlyStopped() {
			log.Printf("[GeminiBatchMatchService] Pipeline stopped. Aborting single-user batches for user %s.", targetUserID)
			return
		}

		singleBatch := jobBatches[0]
		jobBatches = jobBatches[1:]
		batchIteration++

		selectedModel, errModel := s.getNextModelName()
		if errModel != nil {
			log.Printf("[GeminiBatchMatchService] Configuration error for user %s: %v", targetUserID, errModel)
			return
		}

		s.enforceRateLimitPacing()
		currentBatchEvaluatedIDs := make(map[string]bool)
		success := s.evaluateJobBatch(ctx, userProfiles, singleBatch, currentBatchEvaluatedIDs, selectedModel)
		if !success {
			log.Printf("[GeminiBatchMatchService] Single-user batch of %d jobs failed with model %s — re-queuing batch at the end of queue.", len(singleBatch), selectedModel)
			if !s.IsPipelinePermanentlyStopped() {
				jobBatches = append(jobBatches, singleBatch)
			}
		} else {
			log.Printf("[GeminiBatchMatchService] Evaluated single-user batch %d (%d jobs) for user %s.", batchIteration, len(singleBatch), targetUserID)
		}
	}
}

func (s *GeminiBatchMatchService) getNextModelName() (string, error) {
	s.modelErrorMutex.RLock()
	if s.isPipelinePermanentlyStopped {
		s.modelErrorMutex.RUnlock()
		return "", fmt.Errorf("gemini matching pipeline is permanently stopped because all models exceeded error limit")
	}
	s.modelErrorMutex.RUnlock()

	if len(s.Models) == 0 {
		s.Models = parseConfiguredGeminiModels()
	}
	if len(s.Models) == 0 {
		return "", fmt.Errorf("no Gemini models configured in GEMINI_BATCH_MODELS or GEMINI_MODELS")
	}

	s.modelErrorMutex.RLock()
	var activeModels []string
	for _, model := range s.Models {
		if !s.disabledModels[model] {
			activeModels = append(activeModels, model)
		}
	}
	s.modelErrorMutex.RUnlock()

	if len(activeModels) == 0 {
		s.modelErrorMutex.Lock()
		s.isPipelinePermanentlyStopped = true
		if s.OnPipelineShutdown != nil {
			s.OnPipelineShutdown()
		}
		s.modelErrorMutex.Unlock()
		return "", fmt.Errorf("all configured Gemini models are permanently stopped")
	}

	nextIndex := s.currentModelIndex.Add(1) - 1
	modelIndex := int(nextIndex) % len(activeModels)
	if modelIndex < 0 {
		modelIndex = -modelIndex
	}
	return activeModels[modelIndex], nil
}

// IsPipelinePermanentlyStopped returns true if the matching pipeline has been permanently shut down.
func (s *GeminiBatchMatchService) IsPipelinePermanentlyStopped() bool {
	s.modelErrorMutex.RLock()
	defer s.modelErrorMutex.RUnlock()
	return s.isPipelinePermanentlyStopped
}

func (s *GeminiBatchMatchService) recordModelSuccess(modelName string) {
	s.modelErrorMutex.Lock()
	defer s.modelErrorMutex.Unlock()

	s.modelConsecutiveErrors[modelName] = 0
}

func (s *GeminiBatchMatchService) recordModelFailure(ctx context.Context, modelName string, failureReason string) {
	s.modelErrorMutex.Lock()
	defer s.modelErrorMutex.Unlock()

	s.modelConsecutiveErrors[modelName]++
	consecutiveErrors := s.modelConsecutiveErrors[modelName]

	log.Printf("[GeminiBatchMatchService] Model '%s' consecutive error count: %d/%d (Reason: %s)",
		modelName, consecutiveErrors, maxConsecutiveModelErrors, failureReason)

	if consecutiveErrors >= maxConsecutiveModelErrors && !s.disabledModels[modelName] {
		s.disabledModels[modelName] = true
		log.Printf("[GeminiBatchMatchService] Model '%s' exceeded %d consecutive errors. Permanently disabling model.",
			modelName, maxConsecutiveModelErrors)

		go s.notifyAdminModelDisabled(modelName, failureReason)

		allDisabled := true
		for _, model := range s.Models {
			if !s.disabledModels[model] {
				allDisabled = false
				break
			}
		}

		if allDisabled && !s.isPipelinePermanentlyStopped {
			s.isPipelinePermanentlyStopped = true
			log.Println("[GeminiBatchMatchService] All configured Gemini models permanently disabled. Shutting down AI evaluation pipeline.")
			if s.OnPipelineShutdown != nil {
				s.OnPipelineShutdown()
			}
			go s.notifyAdminPipelineShutdown()
		}
	}
}

func (s *GeminiBatchMatchService) notifyAdminModelDisabled(modelName string, reason string) {
	if s.DB == nil {
		return
	}

	title := fmt.Sprintf("Alert: Gemini Model '%s' Permanently Stopped", modelName)
	message := fmt.Sprintf("The Gemini matching engine has permanently disabled model '%s' after reaching %d consecutive errors. Last error: %s. Remaining models in cascade will continue processing.",
		modelName, maxConsecutiveModelErrors, reason)

	masterAdminEmail := os.Getenv("MASTER_ADMIN_EMAIL")
	query := `
		INSERT INTO notifications (user_id, title, message, is_read)
		SELECT id, $1, $2, false
		FROM users
		WHERE is_master_admin = true OR (primary_email = $3 AND $3 != '');
	`
	_, _ = s.DB.Exec(context.Background(), query, title, message, masterAdminEmail)
}

func (s *GeminiBatchMatchService) notifyAdminPipelineShutdown() {
	if s.DB == nil {
		return
	}

	title := "Critical Alert: AI Matching Pipeline Permanently Stopped"
	message := fmt.Sprintf("All configured Gemini models have exceeded %d consecutive errors. The AI batch evaluation pipeline has been permanently stopped. Please check model availability or API quotas.",
		maxConsecutiveModelErrors)

	masterAdminEmail := os.Getenv("MASTER_ADMIN_EMAIL")
	query := `
		INSERT INTO notifications (user_id, title, message, is_read)
		SELECT id, $1, $2, false
		FROM users
		WHERE is_master_admin = true OR (primary_email = $3 AND $3 != '');
	`
	_, _ = s.DB.Exec(context.Background(), query, title, message, masterAdminEmail)
}

func (s *GeminiBatchMatchService) enforceRateLimitPacing() {
	s.rateLimitMutex.Lock()
	defer s.rateLimitMutex.Unlock()

	if !s.lastRequestTime.IsZero() {
		elapsed := time.Since(s.lastRequestTime)
		if elapsed < s.RateLimitInterval {
			waitDuration := s.RateLimitInterval - elapsed
			log.Printf("[GeminiBatchMatchService] Pacing rate limit: waiting %v before next request...", waitDuration.Truncate(time.Millisecond))
			time.Sleep(waitDuration)
		}
	}
	s.lastRequestTime = time.Now()
}

// CountTokens requests exact token count for prompt text using the official Google GenAI token counter API.
func (s *GeminiBatchMatchService) CountTokens(ctx context.Context, modelName string, text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	client, errClient := s.createGenAIClient(ctx)
	if errClient != nil {
		return s.calculateExactSubwordTokens(text), nil
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: text},
			},
		},
	}

	response, errCount := client.Models.CountTokens(ctx, modelName, contents, nil)
	if errCount != nil || response == nil {
		return s.calculateExactSubwordTokens(text), nil
	}

	return int(response.TotalTokens), nil
}

func (s *GeminiBatchMatchService) calculateExactSubwordTokens(text string) int {
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

func (s *GeminiBatchMatchService) buildMultiJobTokenBatches(ctx context.Context, allJobs []JobSnippetData, targetTokenBudget int) [][]JobSnippetData {
	var resultBatches [][]JobSnippetData
	var currentBatch []JobSnippetData
	currentTokens := 0

	for _, job := range allJobs {
		snippetText := fmt.Sprintf("ID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription: %s\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))

		itemTokens := s.calculateExactSubwordTokens(snippetText)

		tokenBudgetExceeded := len(currentBatch) > 0 && currentTokens+itemTokens > targetTokenBudget
		jobCountExceeded := len(currentBatch) >= maxJobsPerBatch
		if tokenBudgetExceeded || jobCountExceeded {
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

func (s *GeminiBatchMatchService) evaluateJobBatch(
	ctx context.Context,
	userProfiles []UserProfileData,
	batch []JobSnippetData,
	evaluatedJobIDs map[string]bool,
	modelName string,
) bool {
	expectedResultCount := len(batch) * len(userProfiles)
	promptText := s.buildMultiJobPrompt(userProfiles, batch, expectedResultCount)

	requestStart := time.Now()
	rawOutput, errGenerate := s.generateBatchContent(ctx, modelName, promptText)
	if errGenerate != nil {
		log.Printf("[GeminiBatchMatchService] API error with model %s: %v", modelName, errGenerate)
		utils.LogRawAIResponse("GeminiBatch-ERROR", modelName, promptText, errGenerate.Error(), time.Since(requestStart), true)
		s.recordModelFailure(ctx, modelName, errGenerate.Error())
		return false
	}

	cleanJSON := sanitizeJSONResponse(rawOutput)
	var batchResult GeminiBatchResponse
	if errJSON := json.Unmarshal([]byte(cleanJSON), &batchResult); errJSON != nil {
		log.Printf("[GeminiBatchMatchService] JSON parse error from model %s: %v", modelName, errJSON)
		utils.LogRawAIResponse("GeminiBatch-PARSE-ERROR", modelName, promptText, rawOutput, time.Since(requestStart), true)
		s.recordModelFailure(ctx, modelName, "JSON parse failure: "+errJSON.Error())
		return false
	}

	utils.LogRawAIResponse("GeminiBatch", modelName, promptText, rawOutput, time.Since(requestStart), false)
	s.recordModelSuccess(modelName)
	log.Printf("[GeminiBatchMatchService] Answer received from model '%s' in %v (Response size: %d bytes). Successfully parsed %d job match results for %d candidate profile(s).",
		modelName, time.Since(requestStart).Truncate(time.Millisecond), len(rawOutput), len(batchResult.Results), len(userProfiles))

	for _, resultItem := range batchResult.Results {
		if resultItem.JobID == "" || resultItem.UserID == "" || !isValidUUIDString(resultItem.JobID) || !isValidUUIDString(resultItem.UserID) {
			continue
		}

		var matchedProfile *UserProfileData
		for profileIndex := range userProfiles {
			if userProfiles[profileIndex].UserID == resultItem.UserID {
				matchedProfile = &userProfiles[profileIndex]
				break
			}
		}

		if matchedProfile != nil {
			s.applyExperienceAndLocationCaps(&resultItem, matchedProfile)
		}

		isMatch := resultItem.MatchScore >= matchScoreMinThreshold
		upsertErr := s.upsertUserMatch(ctx, resultItem.UserID, resultItem.JobID, resultItem.MatchScore, resultItem.MatchReasoning, isMatch, modelName)
		if upsertErr != nil {
			log.Printf("[GeminiBatchMatchService] Upsert error for user %s job %s: %v", resultItem.UserID, resultItem.JobID, upsertErr)
		}

		evaluatedJobIDs[resultItem.JobID] = true
	}

	return true
}

// IsModelDisabledForTest checks if a model is currently disabled.
func (s *GeminiBatchMatchService) IsModelDisabledForTest(modelName string) bool {
	s.modelErrorMutex.RLock()
	defer s.modelErrorMutex.RUnlock()
	return s.disabledModels[modelName]
}

// IsPipelinePermanentlyStoppedForTest checks if the pipeline is permanently stopped.
func (s *GeminiBatchMatchService) IsPipelinePermanentlyStoppedForTest() bool {
	s.modelErrorMutex.RLock()
	defer s.modelErrorMutex.RUnlock()
	return s.isPipelinePermanentlyStopped
}

// GetModelConsecutiveErrorsForTest returns current consecutive failure count for a model.
func (s *GeminiBatchMatchService) GetModelConsecutiveErrorsForTest(modelName string) int {
	s.modelErrorMutex.RLock()
	defer s.modelErrorMutex.RUnlock()
	return s.modelConsecutiveErrors[modelName]
}

// RecordModelFailureForTest records a model failure for testing.
func (s *GeminiBatchMatchService) RecordModelFailureForTest(ctx context.Context, modelName string, reason string) {
	s.recordModelFailure(ctx, modelName, reason)
}

// RecordModelSuccessForTest records a model success for testing.
func (s *GeminiBatchMatchService) RecordModelSuccessForTest(modelName string) {
	s.recordModelSuccess(modelName)
}

func (s *GeminiBatchMatchService) generateBatchContent(ctx context.Context, modelName string, prompt string) (string, error) {
	client, errClient := s.createGenAIClient(ctx)
	if errClient != nil {
		return "", errClient
	}

	var responseSchema genai.Schema
	if errUnmarshal := json.Unmarshal([]byte(GeminiBatchJobMatchSchemaJSON), &responseSchema); errUnmarshal != nil {
		return "", fmt.Errorf("failed unmarshaling structured response schema: %w", errUnmarshal)
	}

	config := &genai.GenerateContentConfig{
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: genai.ThinkingLevelMinimal,
		},
		ResponseMIMEType: "application/json",
		ResponseSchema:   &responseSchema,
		Temperature:      genai.Ptr[float32](0.0),
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}

	result, errGen := client.Models.GenerateContent(ctx, modelName, contents, config)
	if errGen != nil {
		return "", fmt.Errorf("gemini generation failed for model %s: %w", modelName, errGen)
	}

	return result.Text(), nil
}

func (s *GeminiBatchMatchService) createGenAIClient(ctx context.Context) (*genai.Client, error) {
	clientConfig := &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  s.APIKey,
	}
	if s.HTTPClient != nil {
		clientConfig.HTTPClient = s.HTTPClient
	}
	if s.BaseURL != "" {
		clientConfig.HTTPOptions = genai.HTTPOptions{
			BaseURL: s.BaseURL,
		}
	}
	return genai.NewClient(ctx, clientConfig)
}

func (s *GeminiBatchMatchService) applyExperienceAndLocationCaps(result *GeminiMatchResult, profile *UserProfileData) {
	candidateYears := profile.ExperienceYears
	jobMinimumYears := result.InferredRequiredYoE

	if jobMinimumYears > 0 {
		if jobMinimumYears > (candidateYears + 2) {
			if result.MatchScore > 25 {
				result.MatchScore = 25
				result.MatchReasoning = fmt.Sprintf("[Experience Gap Cap] Job requires %d+ YoE, candidate has %d YoE. Tech fit scored %d%%. %s",
					jobMinimumYears, candidateYears, result.MatchScore, result.MatchReasoning)
			}
		} else if jobMinimumYears > candidateYears {
			if result.MatchScore > 45 {
				result.MatchScore = 45
				result.MatchReasoning = fmt.Sprintf("[Stretch Role Cap] Job requires %d+ YoE, candidate has %d YoE. Tech fit scored %d%%. %s",
					jobMinimumYears, candidateYears, result.MatchScore, result.MatchReasoning)
			}
		}
	}
}

func (s *GeminiBatchMatchService) buildMultiJobPrompt(userProfiles []UserProfileData, jobsBatch []JobSnippetData, expectedResultCount int) string {
	var builder strings.Builder
	currentTimeText := time.Now().Format("January 2006")

	builder.WriteString("Return ONLY a raw JSON object formatted according to the schema:\n")
	builder.WriteString("{\n  \"results\": [\n    {\n      \"job_id\": \"<job_id verbatim>\",\n      \"user_id\": \"<user_id verbatim>\",\n      \"match_score\": 85,\n      \"match_reasoning\": \"Detailed 2-3 line reasoning specifying exact tech stack overlap, candidate base location vs job requirement, and YoE comparison.\",\n      \"inferred_required_yoe\": 4,\n      \"is_matched\": true\n    }\n  ]\n}\n\n")
	fmt.Fprintf(&builder, "IMPORTANT: You MUST return exactly %d entries in the \"results\" array — one for every (job × candidate) pair listed below.\n\n", expectedResultCount)

	builder.WriteString("SCORING RULES:\n")
	builder.WriteString("1. LOCATION MISMATCH (HARD CAP): Candidate is India-based. Any job that is US Onsite, US Hybrid, or US-only Remote (explicitly excludes non-US applicants) -> cap score at 0-15.\n")
	builder.WriteString("2. EXPERIENCE GAP (HARD CAP):\n")
	fmt.Fprintf(&builder, "   - Use Candidate YoE calculated from resume relative to %s. Infer job minimum required YoE from the JD.\n", currentTimeText)
	builder.WriteString("   - If required YoE > (candidate YoE + 3): cap score at 0-25.\n")
	builder.WriteString("   - If required YoE is (candidate YoE + 1) to (candidate YoE + 2): cap score at 45-65.\n")
	builder.WriteString("   - If candidate YoE meets or exceeds required YoE: bonus +15 to +20 points if tech stack matches.\n")
	builder.WriteString("3. HIGH MATCH (75-100): ONLY when YoE meets requirements, location matches, and tech stack strongly overlaps.\n\n")

	builder.WriteString("### CANDIDATE PROFILES\n")
	for _, profile := range userProfiles {
		combinedProfileText := profile.ParsedBio
		if profile.MasterCVText != "" {
			combinedProfileText += "\n\nMaster CV / Full Experience Context:\n" + profile.MasterCVText
		}
		fmt.Fprintf(&builder, "User ID: %s\nCandidate YoE: %d\nProfile & Resume Context:\n%s\nPreferred Roles: %s\nPreferred Locations: %s\nWork Model: %s\n\n",
			profile.UserID, profile.ExperienceYears, combinedProfileText, strings.Join(profile.PreferredRoles, ", "), strings.Join(profile.PreferredLocations, ", "), profile.WorkModel)
	}

	builder.WriteString("### JOB LISTINGS TO EVALUATE\n")
	for _, job := range jobsBatch {
		fmt.Fprintf(&builder, "---\nID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription:\n%s\n\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))
	}

	fmt.Fprintf(&builder, "NOW OUTPUT THE STRUCTURED JSON. Exactly %d entries.\n", expectedResultCount)
	return builder.String()
}

func (s *GeminiBatchMatchService) fetchAllUserProfiles(ctx context.Context) ([]UserProfileData, error) {
	sqlQuery := `
		SELECT u.id, u.primary_email, COALESCE(up.bio_experience_text, ''), COALESCE(up.master_cv_text, ''), COALESCE(up.target_roles, '[]'),
		       COALESCE(up.target_locations, '[]'), COALESCE(up.work_models->>0, ''), 0
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.ai_matching_enabled = true;
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
			item.ExperienceYears = calculateTotalExperienceYears(item.MasterCVText)
			profiles = append(profiles, item)
		}
	}
	return profiles, nil
}

func (s *GeminiBatchMatchService) fetchSingleUserProfile(ctx context.Context, targetUserID string) (*UserProfileData, error) {
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
	item.ExperienceYears = calculateTotalExperienceYears(item.MasterCVText)
	return &item, nil
}

func (s *GeminiBatchMatchService) fetchUnevaluatedJobs(ctx context.Context) ([]JobSnippetData, error) {
	sqlQuery := `
		SELECT j.id, j.title, COALESCE(c.name, ''), COALESCE(j.location, ''), COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE j.ai_evaluated = false
		  AND j.scraped_at >= NOW() - INTERVAL '14 days'
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

func (s *GeminiBatchMatchService) fetchJobsUnmatchedForUser(ctx context.Context, targetUserID string) ([]JobSnippetData, error) {
	sqlQuery := `
		SELECT j.id, j.title, COALESCE(c.name, ''), COALESCE(j.location, ''), COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE j.ai_evaluated = true
		  AND j.scraped_at >= NOW() - INTERVAL '14 days'
		  AND NOT EXISTS (
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

func (s *GeminiBatchMatchService) upsertUserMatch(ctx context.Context, userID, jobID string, score int, reasoning string, isMatch bool, modelName string) error {
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
	_, errExec := s.DB.Exec(ctx, sqlQuery, userID, jobID, score, reasoning, isMatch, modelName)
	return errExec
}

func (s *GeminiBatchMatchService) markJobsEvaluated(ctx context.Context, jobIDsMap map[string]bool) error {
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

// GetNextModelNameForTest exposes getNextModelName for unit test verification.
func (s *GeminiBatchMatchService) GetNextModelNameForTest() (string, error) {
	return s.getNextModelName()
}

// BuildMultiJobTokenBatchesForTest exposes buildMultiJobTokenBatches for unit test verification.
func (s *GeminiBatchMatchService) BuildMultiJobTokenBatchesForTest(ctx context.Context, allJobs []JobSnippetData, targetTokenBudget int) [][]JobSnippetData {
	return s.buildMultiJobTokenBatches(ctx, allJobs, targetTokenBudget)
}

// GenerateBatchContentForTest exposes generateBatchContent for unit test verification.
func (s *GeminiBatchMatchService) GenerateBatchContentForTest(ctx context.Context, modelName string, prompt string) (string, error) {
	return s.generateBatchContent(ctx, modelName, prompt)
}

// ParseAndValidateBatchJSONForTest unmarshals and validates JSON for unit test verification.
func (s *GeminiBatchMatchService) ParseAndValidateBatchJSONForTest(rawJSON string) (GeminiBatchResponse, bool) {
	cleanJSON := sanitizeJSONResponse(rawJSON)
	var batchResult GeminiBatchResponse
	if errJSON := json.Unmarshal([]byte(cleanJSON), &batchResult); errJSON != nil {
		return GeminiBatchResponse{}, false
	}
	return batchResult, true
}

// ApplyExperienceAndLocationCapsForTest exposes applyExperienceAndLocationCaps for unit test verification.
func (s *GeminiBatchMatchService) ApplyExperienceAndLocationCapsForTest(result *GeminiMatchResult, profile *UserProfileData) {
	s.applyExperienceAndLocationCaps(result, profile)
}
