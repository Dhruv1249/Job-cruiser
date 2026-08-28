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
type GeminiMatchResult = BatchMatchResultItem

// GeminiBatchResponse defines the structured JSON array envelope emitted by Gemini models.
type GeminiBatchResponse = BatchMatchResponse

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
          "standardized_location": {"type": "string"},
          "work_model": {"type": "string"},
          "is_matched": {"type": "boolean"}
        },
        "required": ["job_id", "user_id", "match_score", "match_reasoning", "inferred_required_yoe", "standardized_location", "work_model", "is_matched"],
        "propertyOrdering": ["job_id", "user_id", "match_score", "match_reasoning", "inferred_required_yoe", "standardized_location", "work_model", "is_matched"]
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
	queueMutex             sync.RWMutex
	isQueuePaused          bool
	isEvaluationInProgress bool
	isPipelinePermanentlyStopped bool
	OnPipelineShutdown     func()
	modelErrorMutex        sync.RWMutex
	modelConsecutiveErrors map[string]int
	disabledModels         map[string]bool
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

	userProfiles, errProfiles := fetchAllActiveUserProfiles(ctx, s.DB)
	if errProfiles != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch user profiles: %v", errProfiles)
		return
	}
	if len(userProfiles) == 0 {
		log.Println("[GeminiBatchMatchService] No active user profiles found.")
		return
	}

	pendingJobs, errJobs := fetchRecentUnevaluatedJobs(ctx, s.DB)
	if errJobs != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch unevaluated jobs: %v", errJobs)
		return
	}
	if len(pendingJobs) == 0 {
		log.Println("[GeminiBatchMatchService] Zero unevaluated jobs found.")
		return
	}

	jobBatches := buildMultiJobTokenBatches(ctx, pendingJobs, s.TargetTokenBudget)
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
				markErr := markJobsAsEvaluatedInDatabase(ctx, s.DB, currentBatchEvaluatedIDs)
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

	profile, errProfile := fetchSingleUserProfileByID(ctx, s.DB, targetUserID)
	if errProfile != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch profile for user %s: %v", targetUserID, errProfile)
		return
	}

	unmatchedJobs, errJobs := fetchUnmatchedJobsForSingleUser(ctx, s.DB, targetUserID)
	if errJobs != nil {
		log.Printf("[GeminiBatchMatchService] Failed to fetch unmatched jobs for user %s: %v", targetUserID, errJobs)
		return
	}
	if len(unmatchedJobs) == 0 {
		return
	}

	jobBatches := buildMultiJobTokenBatches(ctx, unmatchedJobs, s.TargetTokenBudget)
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
		return calculateExactSubwordTokens(text), nil
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
		return calculateExactSubwordTokens(text), nil
	}

	return int(response.TotalTokens), nil
}

func (s *GeminiBatchMatchService) evaluateJobBatch(
	ctx context.Context,
	userProfiles []UserProfileData,
	batch []JobSnippetData,
	evaluatedJobIDs map[string]bool,
	modelName string,
) bool {
	expectedResultCount := len(batch) * len(userProfiles)
	promptText := buildMultiJobPrompt(userProfiles, batch, expectedResultCount)

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
			applyExperienceAndLocationCaps(&resultItem, matchedProfile)
		}

		isMatch := resultItem.MatchScore >= matchScoreMinThreshold
		upsertErr := upsertUserJobMatchRecord(ctx, s.DB, resultItem.UserID, resultItem.JobID, resultItem.MatchScore, resultItem.MatchReasoning, isMatch, modelName)
		if upsertErr != nil {
			log.Printf("[GeminiBatchMatchService] Upsert error for user %s job %s: %v", resultItem.UserID, resultItem.JobID, upsertErr)
		}
		_ = updateJobStandardizedLocationAndWorkModel(ctx, s.DB, resultItem.JobID, resultItem.StandardizedLocation, resultItem.WorkModel)

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

// GetNextModelNameForTest exposes getNextModelName for unit test verification.
func (s *GeminiBatchMatchService) GetNextModelNameForTest() (string, error) {
	return s.getNextModelName()
}

// BuildMultiJobTokenBatchesForTest exposes buildMultiJobTokenBatches for unit test verification.
func (s *GeminiBatchMatchService) BuildMultiJobTokenBatchesForTest(ctx context.Context, allJobs []JobSnippetData, targetTokenBudget int) [][]JobSnippetData {
	return buildMultiJobTokenBatches(ctx, allJobs, targetTokenBudget)
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
	applyExperienceAndLocationCaps(result, profile)
}
