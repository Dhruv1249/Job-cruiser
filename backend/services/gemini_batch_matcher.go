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
	DB                           *pgxpool.Pool
	APIKey                       string
	BaseURL                      string
	HTTPClient                   *http.Client
	Models                       []string
	currentModelIndex            atomic.Int64
	TargetTokenBudget            int
	RateLimitInterval            time.Duration
	lastRequestTime              time.Time
	rateLimitMutex               sync.Mutex
	queueMutex                   sync.RWMutex
	isQueuePaused                bool
	isEvaluationInProgress       bool
	isPipelinePermanentlyStopped bool
	OnPipelineShutdown           func()
	modelErrorMutex              sync.RWMutex
	modelRunErrors               map[string]int
	disabledModels               map[string]bool
	runDisabledModels            map[string]bool
	ModelTokenBudgets            map[string]int
}

const (
	maxConsecutiveModelErrors = 6
	defaultGlobalTokenBudget  = 100000
)

// GetTargetTokenBudgetForModel returns the per-model token budget based on GEMINI_MODEL_TOKEN_BUDGETS or global fallback.
func (s *GeminiBatchMatchService) GetTargetTokenBudgetForModel(modelName string) int {
	normalizedName := strings.ToLower(strings.TrimSpace(modelName))

	if s.ModelTokenBudgets != nil {
		if budget, exists := s.ModelTokenBudgets[normalizedName]; exists && budget > 0 {
			return budget
		}
	}

	if s.TargetTokenBudget > 0 {
		return s.TargetTokenBudget
	}

	return defaultGlobalTokenBudget
}

func parseConfiguredModelTokenBudgets() map[string]int {
	budgets := make(map[string]int)
	rawBudgets := strings.TrimSpace(os.Getenv("GEMINI_MODEL_TOKEN_BUDGETS"))
	if rawBudgets != "" {
		pairs := strings.Split(rawBudgets, ",")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				modelKey := strings.ToLower(strings.TrimSpace(parts[0]))
				valStr := strings.TrimSpace(parts[1])
				if parsedVal, err := strconv.Atoi(valStr); err == nil && parsedVal > 0 {
					budgets[modelKey] = parsedVal
				}
			}
		}
	}
	return budgets
}

// NewGeminiBatchMatchService constructs a new GeminiBatchMatchService configured via environment variables.
func NewGeminiBatchMatchService(databasePool *pgxpool.Pool, apiKey string) *GeminiBatchMatchService {
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}

	configuredModels := parseConfiguredGeminiModels()
	tokenBudget := parseConfiguredTokenBudget()
	rateLimitDuration := parseConfiguredRateLimitDuration()
	modelBudgets := parseConfiguredModelTokenBudgets()

	return &GeminiBatchMatchService{
		DB:                databasePool,
		APIKey:            apiKey,
		HTTPClient:        &http.Client{Timeout: 15 * time.Minute},
		Models:            configuredModels,
		TargetTokenBudget: tokenBudget,
		RateLimitInterval: rateLimitDuration,
		modelRunErrors:    make(map[string]int),
		disabledModels:    make(map[string]bool),
		runDisabledModels: make(map[string]bool),
		ModelTokenBudgets: modelBudgets,
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

	s.resetRunErrorsIfHealthy()

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
		log.Println("[GeminiBatchMatchService] Zero unevaluated jobs found. Evaluating unmatched backlog for active users.")
		for _, profile := range userProfiles {
			unmatchedJobs, errUnmatched := fetchUnmatchedJobsForSingleUser(ctx, s.DB, profile.UserID)
			if errUnmatched == nil && len(unmatchedJobs) > 0 {
				s.EvaluateForSingleUser(ctx, profile.UserID)
			}
		}
		return
	}

	batchIteration := 0
	for len(pendingJobs) > 0 {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.IsPipelinePermanentlyStopped() {
			log.Println("[GeminiBatchMatchService] AI matching pipeline is permanently stopped. Aborting remaining jobs.")
			return
		}

		selectedModel, errModel := s.getNextModelName()
		if errModel != nil {
			log.Printf("[GeminiBatchMatchService] Configuration/Shutdown error: %v", errModel)
			return
		}

		modelBudget := s.GetTargetTokenBudgetForModel(selectedModel)
		singleBatch, remainingJobs := extractSingleJobBatch(pendingJobs, modelBudget)
		pendingJobs = remainingJobs
		batchIteration++

		log.Printf("[GeminiBatchMatchService] Processing batch %d (%d jobs, %d remaining in queue) using model: %s (Token Budget: %d)...",
			batchIteration, len(singleBatch), len(pendingJobs), selectedModel, modelBudget)

		s.enforceRateLimitPacing()

		currentBatchEvaluatedIDs := make(map[string]bool)
		success := s.evaluateJobBatch(ctx, userProfiles, singleBatch, currentBatchEvaluatedIDs, selectedModel)
		if !success {
			log.Printf("[GeminiBatchMatchService] Batch of %d jobs failed with model %s — re-queuing jobs at the end of queue.", len(singleBatch), selectedModel)
			if !s.IsPipelinePermanentlyStopped() {
				pendingJobs = append(pendingJobs, singleBatch...)
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

	s.resetRunErrorsIfHealthy()

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

	userProfiles := []UserProfileData{*profile}
	batchIteration := 0

	for len(unmatchedJobs) > 0 {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.IsPipelinePermanentlyStopped() {
			log.Printf("[GeminiBatchMatchService] Pipeline stopped. Aborting single-user batches for user %s.", targetUserID)
			return
		}

		selectedModel, errModel := s.getNextModelName()
		if errModel != nil {
			log.Printf("[GeminiBatchMatchService] Configuration error for user %s: %v", targetUserID, errModel)
			return
		}

		modelBudget := s.GetTargetTokenBudgetForModel(selectedModel)
		singleBatch, remainingJobs := extractSingleJobBatch(unmatchedJobs, modelBudget)
		unmatchedJobs = remainingJobs
		batchIteration++

		log.Printf("[GeminiBatchMatchService] Processing single-user batch %d (%d jobs, %d remaining) for user %s using model: %s (Token Budget: %d)...",
			batchIteration, len(singleBatch), len(unmatchedJobs), targetUserID, selectedModel, modelBudget)

		s.enforceRateLimitPacing()
		currentBatchEvaluatedIDs := make(map[string]bool)
		success := s.evaluateJobBatch(ctx, userProfiles, singleBatch, currentBatchEvaluatedIDs, selectedModel)
		if !success {
			log.Printf("[GeminiBatchMatchService] Single-user batch of %d jobs failed with model %s — re-queuing jobs at the end of queue.", len(singleBatch), selectedModel)
			if !s.IsPipelinePermanentlyStopped() {
				unmatchedJobs = append(unmatchedJobs, singleBatch...)
			}
		} else {
			log.Printf("[GeminiBatchMatchService] Evaluated single-user batch %d (%d jobs) for user %s.", batchIteration, len(singleBatch), targetUserID)
		}
	}
}

func extractSingleJobBatch(jobs []JobSnippetData, targetTokenBudget int) ([]JobSnippetData, []JobSnippetData) {
	if len(jobs) == 0 {
		return nil, nil
	}
	if targetTokenBudget <= 0 {
		targetTokenBudget = defaultGlobalTokenBudget
	}

	var batch []JobSnippetData
	currentTokens := 0
	cutIndex := 0

	for index, job := range jobs {
		snippetText := fmt.Sprintf("ID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription: %s\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))

		itemTokens := calculateExactSubwordTokens(snippetText)
		tokenBudgetExceeded := len(batch) > 0 && currentTokens+itemTokens > targetTokenBudget
		jobCountExceeded := len(batch) >= maxJobsPerBatch

		if tokenBudgetExceeded || jobCountExceeded {
			cutIndex = index
			break
		}

		batch = append(batch, job)
		currentTokens += itemTokens
		cutIndex = index + 1
	}

	return batch, jobs[cutIndex:]
}

func (s *GeminiBatchMatchService) resetRunErrorsIfHealthy() {
	s.modelErrorMutex.Lock()
	defer s.modelErrorMutex.Unlock()

	if !s.isPipelinePermanentlyStopped {
		s.runDisabledModels = make(map[string]bool)
		s.modelRunErrors = make(map[string]int)
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
		if !s.disabledModels[model] && !s.runDisabledModels[model] {
			activeModels = append(activeModels, model)
		}
	}
	s.modelErrorMutex.RUnlock()

	if len(activeModels) == 0 {
		return "", fmt.Errorf("no active Gemini models available for this run")
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

	s.modelRunErrors[modelName] = 0
}

func (s *GeminiBatchMatchService) recordModelFailure(ctx context.Context, modelName string, failureReason string) {
	s.modelErrorMutex.Lock()
	defer s.modelErrorMutex.Unlock()

	s.modelRunErrors[modelName]++
	consecutiveErrors := s.modelRunErrors[modelName]

	log.Printf("[GeminiBatchMatchService] Model '%s' error count for current run: %d/%d (Reason: %s)",
		modelName, consecutiveErrors, maxConsecutiveModelErrors, failureReason)

	if consecutiveErrors >= maxConsecutiveModelErrors && !s.runDisabledModels[modelName] {
		s.runDisabledModels[modelName] = true
		log.Printf("[GeminiBatchMatchService] Model '%s' reached %d errors in current run. Disabled for this turn.",
			modelName, maxConsecutiveModelErrors)

		allDisabledInRun := true
		for _, model := range s.Models {
			if !s.runDisabledModels[model] && !s.disabledModels[model] {
				allDisabledInRun = false
				break
			}
		}

		if allDisabledInRun && !s.isPipelinePermanentlyStopped {
			s.isPipelinePermanentlyStopped = true
			for _, model := range s.Models {
				s.disabledModels[model] = true
			}
			log.Println("[GeminiBatchMatchService] All configured Gemini models reached error threshold. Permanently shutting down AI evaluation pipeline.")
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
	systemInstruction := buildBatchMatchSystemInstruction(expectedResultCount)
	userContent := buildBatchMatchUserContent(userProfiles, batch, expectedResultCount)

	requestStart := time.Now()
	rawOutput, errGenerate := s.generateBatchContentWithSystem(ctx, modelName, systemInstruction, userContent)
	if errGenerate != nil {
		log.Printf("[GeminiBatchMatchService] API error with model %s: %v", modelName, errGenerate)
		utils.LogRawAIResponse("GeminiBatch-ERROR", modelName, userContent, errGenerate.Error(), time.Since(requestStart), true)
		s.recordModelFailure(ctx, modelName, errGenerate.Error())
		return false
	}

	cleanJSON := sanitizeJSONResponse(rawOutput)
	var batchResult GeminiBatchResponse
	if errJSON := json.Unmarshal([]byte(cleanJSON), &batchResult); errJSON != nil {
		log.Printf("[GeminiBatchMatchService] JSON parse warning from model %s: %v. Raw output will not be counted as model error.", modelName, errJSON)
		utils.LogRawAIResponse("GeminiBatch-PARSE-ERROR", modelName, userContent, rawOutput, time.Since(requestStart), true)
		return false
	}

	utils.LogRawAIResponse("GeminiBatch", modelName, userContent, rawOutput, time.Since(requestStart), false)
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
		if matchedProfile != nil {
			notifyUserOnHighMatch(ctx, s.DB, matchedProfile, resultItem.JobID, resultItem.MatchScore, resultItem.MatchReasoning)
		}

		evaluatedJobIDs[resultItem.JobID] = true
	}

	return true
}

// IsModelDisabledForTest checks if a model is currently disabled.
func (s *GeminiBatchMatchService) IsModelDisabledForTest(modelName string) bool {
	s.modelErrorMutex.RLock()
	defer s.modelErrorMutex.RUnlock()
	return s.disabledModels[modelName] || s.runDisabledModels[modelName]
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
	return s.modelRunErrors[modelName]
}

// RecordModelFailureForTest records a model failure for testing.
func (s *GeminiBatchMatchService) RecordModelFailureForTest(ctx context.Context, modelName string, reason string) {
	s.recordModelFailure(ctx, modelName, reason)
}

// RecordModelSuccessForTest records a model success for testing.
func (s *GeminiBatchMatchService) RecordModelSuccessForTest(modelName string) {
	s.recordModelSuccess(modelName)
}

// ResetRunErrorsIfHealthyForTest resets run errors for test simulation of a new run.
func (s *GeminiBatchMatchService) ResetRunErrorsIfHealthyForTest() {
	s.resetRunErrorsIfHealthy()
}

func (s *GeminiBatchMatchService) generateBatchContent(ctx context.Context, modelName string, prompt string) (string, error) {
	return s.generateBatchContentWithSystem(ctx, modelName, "", prompt)
}

func (s *GeminiBatchMatchService) generateBatchContentWithSystem(ctx context.Context, modelName string, systemInstruction string, prompt string) (string, error) {
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

	if strings.TrimSpace(systemInstruction) != "" {
		config.SystemInstruction = &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: systemInstruction},
			},
		}
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
