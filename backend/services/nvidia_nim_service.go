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
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultNvidiaNimEndpoint        = "https://integrate.api.nvidia.com/v1/chat/completions"
	defaultNvidiaNimModel           = "deepseek-ai/deepseek-v4-flash-0731"
	backgroundWorkerBatchSize       = 8
	backgroundTickerIntervalSeconds = 600
	maxJobDescriptionLength         = 10000
	matchScoreMinThreshold          = 30
	maxTargetTokensPerPrompt        = 250000
	maxJobsPerBatch                 = 300
	tokenBucketRatePerMinute        = 30
	retryBackoffBaseSeconds         = 120
	retryBackoffMaxSeconds          = 300
	retryBackoffJitterFraction      = 0.55
	maxOutputTokensPerBatch         = 90000
	httpTimeoutDuration             = 10 * time.Minute
	probeRetryDelaySeconds          = 60
	maxProbeAttempts                = 3
)

// NvidiaNimService manages single-key AI job matching, CV parsing, and queue preemption for NVIDIA NIM API.
type NvidiaNimService struct {
	DB                           *pgxpool.Pool
	APIKey                       string
	Endpoint                     string
	ModelName                    string
	HTTPClient                   *http.Client
	queueMutex                   sync.RWMutex
	isQueuePaused                bool
	isEvaluationInProgress       bool
	isPipelinePermanentlyStopped bool
	tokenBucket                  chan struct{}
	consecutiveFailures          atomic.Int64
	ProbeRetryDelayDuration      time.Duration
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

// BatchMatchResultItem encapsulates an evaluated job match output for a specific candidate.
type BatchMatchResultItem struct {
	JobID               string `json:"job_id"`
	UserID              string `json:"user_id"`
	MatchScore          int    `json:"match_score"`
	MatchReasoning      string `json:"match_reasoning"`
	InferredRequiredYoE int    `json:"inferred_required_yoe"`
	IsMatched           bool   `json:"is_matched"`
}

// BatchMatchResponse defines the structured JSON array envelope emitted by AI matching engines.
type BatchMatchResponse struct {
	Results []BatchMatchResultItem `json:"results"`
}

type nvidiaMatchResult = BatchMatchResultItem
type nvidiaBatchResponse = BatchMatchResponse

type nvidiaRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type nvidiaResponseFormat struct {
	Type string `json:"type"`
}

// NvidiaNvExt defines the nvext parameters for NVIDIA NIM structured output generation (guided_json, guided_regex, guided_choice, guided_grammar).
type NvidiaNvExt struct {
	GuidedJSON    any      `json:"guided_json,omitempty"`
	GuidedRegex   string   `json:"guided_regex,omitempty"`
	GuidedChoice  []string `json:"guided_choice,omitempty"`
	GuidedGrammar string   `json:"guided_grammar,omitempty"`
}

// buildBatchJobMatchSchema returns a JSON schema for the batch job match evaluation output.
func buildBatchJobMatchSchema(expectedCount int) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"results": map[string]any{
				"type":     "array",
				"minItems": expectedCount,
				"maxItems": expectedCount,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"job_id":                map[string]any{"type": "string"},
						"user_id":               map[string]any{"type": "string"},
						"match_score":           map[string]any{"type": "integer"},
						"match_reasoning":       map[string]any{"type": "string"},
						"inferred_required_yoe": map[string]any{"type": "integer"},
						"is_matched":            map[string]any{"type": "boolean"},
					},
					"required": []string{"job_id", "user_id", "match_score", "match_reasoning", "inferred_required_yoe", "is_matched"},
				},
			},
		},
		"required": []string{"results"},
	}
}

// CVParsingJSONSchema defines the JSON schema enforcing structured resume parsing output format from NVIDIA NIM.
var CVParsingJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"bio_summary": map[string]any{"type": "string"},
		"location":    map[string]any{"type": "string"},
		"skills": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
		"education": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"institution": map[string]any{"type": "string"},
					"degree":      map[string]any{"type": "string"},
					"year":        map[string]any{"type": "string"},
					"grade":       map[string]any{"type": "string"},
				},
				"required": []string{"institution", "degree"},
			},
		},
		"experience": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"company":    map[string]any{"type": "string"},
					"role":       map[string]any{"type": "string"},
					"duration":   map[string]any{"type": "string"},
					"highlights": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required": []string{"company", "role"},
			},
		},
		"projects": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"tech_stack":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"description": map[string]any{"type": "string"},
					"link":        map[string]any{"type": "string"},
				},
				"required": []string{"title"},
			},
		},
		"achievements": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":   map[string]any{"type": "string"},
					"details": map[string]any{"type": "string"},
				},
				"required": []string{"title"},
			},
		},
		"certifications": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":   map[string]any{"type": "string"},
					"issuer": map[string]any{"type": "string"},
				},
				"required": []string{"name"},
			},
		},
		"discovered_keywords": map[string]any{
			"type":  "array",
			"items": map[string]any{"type": "string"},
		},
	},
	"required": []string{"bio_summary", "location", "skills", "education", "experience", "projects", "achievements", "certifications", "discovered_keywords"},
}

type nvidiaChatTemplateKwargs struct {
	EnableThinking       bool   `json:"enable_thinking,omitempty"`
	ThinkingMode         string `json:"thinking_mode,omitempty"`
	ForceNonemptyContent bool   `json:"force_nonempty_content,omitempty"`
}

type nvidiaRequest struct {
	Model              string                    `json:"model"`
	Messages           []nvidiaRequestMessage    `json:"messages"`
	Temperature        float64                   `json:"temperature"`
	TopP               float64                   `json:"top_p"`
	MaxTokens          int                       `json:"max_tokens,omitempty"`
	Seed               int                       `json:"seed,omitempty"`
	Stream             bool                      `json:"stream,omitempty"`
	ChatTemplateKwargs *nvidiaChatTemplateKwargs `json:"chat_template_kwargs,omitempty"`
	ResponseFormat     *nvidiaResponseFormat     `json:"response_format,omitempty"`
	NvExt              *NvidiaNvExt              `json:"nvext,omitempty"`
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
			Reasoning        string `json:"reasoning"`
			Content          string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// NewNvidiaNimService initializes the single-key NVIDIA NIM service using provided environment values
// and starts a background token-bucket goroutine that emits one token every (60/tokenBucketRatePerMinute)
// seconds, enforcing the global outbound request rate across all concurrent workers.
func NewNvidiaNimService(databasePool *pgxpool.Pool, rawApiKey string) *NvidiaNimService {
	apiKey := strings.TrimSpace(rawApiKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("NVIDIA_API_KEY"))
	}
	modelName := strings.TrimSpace(os.Getenv("NVIDIA_MODEL"))
	if modelName == "" {
		modelName = defaultNvidiaNimModel
	}

	tokenIntervalSeconds := 60.0 / float64(tokenBucketRatePerMinute)
	tokenInterval := time.Duration(tokenIntervalSeconds * float64(time.Second))

	service := &NvidiaNimService{
		DB:          databasePool,
		APIKey:      apiKey,
		Endpoint:    defaultNvidiaNimEndpoint,
		ModelName:   modelName,
		HTTPClient:  &http.Client{Timeout: 0},
		tokenBucket: make(chan struct{}, 1),
	}

	go func() {
		ticker := time.NewTicker(tokenInterval)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case service.tokenBucket <- struct{}{}:
			default:
			}
		}
	}()

	log.Printf("[NvidiaNimService] Initialized with single NVIDIA API Key (Model: %s, Rate: %d RPM / 1 token per %.1fs)",
		modelName, tokenBucketRatePerMinute, tokenIntervalSeconds)
	return service
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

// IsPipelinePermanentlyStopped returns true if the matching pipeline has been permanently shut down.
func (s *NvidiaNimService) IsPipelinePermanentlyStopped() bool {
	s.queueMutex.RLock()
	defer s.queueMutex.RUnlock()
	return s.isPipelinePermanentlyStopped
}

// SetPipelinePermanentlyStopped marks the NVIDIA NIM engine as permanently shut down.
func (s *NvidiaNimService) SetPipelinePermanentlyStopped(stopped bool) {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.isPipelinePermanentlyStopped = stopped
}

// FillTokenBucketForTest refills the rate limiter token bucket continuously for non-blocking unit tests.
func (s *NvidiaNimService) FillTokenBucketForTest() {
	go func() {
		for iteration := 0; iteration < 50; iteration++ {
			select {
			case s.tokenBucket <- struct{}{}:
			default:
			}
			time.Sleep(time.Millisecond)
		}
	}()
}

// StartBackgroundScheduler starts a 10-minute periodic background ticker executing pending job evaluations.
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
				if s.IsPipelinePermanentlyStopped() {
					log.Println("[NvidiaNimService] Pipeline is permanently shut down, skipping background batch dispatch.")
					continue
				}
				if s.IsQueuePaused() {
					log.Println("[NvidiaNimService] Queue is paused, skipping background batch dispatch.")
					continue
				}
				s.EvaluatePendingForAllUsers(ctx)
			}
		}
	}()
}

// EvaluatePendingForAllUsers fetches unevaluated jobs, packs them into multi-job token batches, and dispatches a worker pool.
func (s *NvidiaNimService) EvaluatePendingForAllUsers(ctx context.Context) {
	_ = s.EvaluatePendingForAllUsersWithResult(ctx)
}

// EvaluatePendingForAllUsersWithResult executes the worker pool and returns true if completed without circuit breaker abort.
func (s *NvidiaNimService) EvaluatePendingForAllUsersWithResult(ctx context.Context) bool {
	s.queueMutex.Lock()
	if s.isPipelinePermanentlyStopped {
		s.queueMutex.Unlock()
		log.Println("[NvidiaNimService] Pipeline is permanently shut down. Skipping evaluation.")
		return false
	}
	if s.isEvaluationInProgress {
		s.queueMutex.Unlock()
		return true
	}
	s.isEvaluationInProgress = true
	s.queueMutex.Unlock()

	defer func() {
		s.queueMutex.Lock()
		s.isEvaluationInProgress = false
		s.queueMutex.Unlock()
	}()

	userProfiles, errProfiles := fetchAllActiveUserProfiles(ctx, s.DB)
	if errProfiles != nil || len(userProfiles) == 0 {
		return true
	}

	pendingJobs, errJobs := fetchRecentUnevaluatedJobs(ctx, s.DB)
	if errJobs != nil || len(pendingJobs) == 0 {
		return true
	}

	jobBatches := buildMultiJobTokenBatches(ctx, pendingJobs, maxTargetTokensPerPrompt)
	if len(jobBatches) == 0 {
		return true
	}

	log.Printf("[NvidiaNimService] Starting evaluation pass for %d active users across %d pending jobs...", len(userProfiles), len(pendingJobs))
	log.Printf("[NvidiaNimService] Dispatching continuous pipeline: %d max concurrent workers across %d total batches...", backgroundWorkerBatchSize, len(jobBatches))

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	var hasCircuitBroken atomic.Bool
	evaluatedJobIDs := make(map[string]bool)
	var syncMutex sync.Mutex
	var workerWaitGroup sync.WaitGroup

	workerPool := make(chan int, backgroundWorkerBatchSize)
	for i := 1; i <= backgroundWorkerBatchSize; i++ {
		workerPool <- i
	}

	for _, singleBatch := range jobBatches {
		if workerCtx.Err() != nil || hasCircuitBroken.Load() {
			break
		}

		workerID := <-workerPool
		workerWaitGroup.Add(1)

		go func(batch []JobSnippetData, id int) {
			defer workerWaitGroup.Done()
			defer func() { workerPool <- id }()

			if workerCtx.Err() != nil || hasCircuitBroken.Load() {
				return
			}

			log.Printf("[NvidiaNimWorker-%d] Starting evaluation for batch of %d jobs...", id, len(batch))
			success := s.evaluateJobBatchWithBackoff(workerCtx, userProfiles, batch, evaluatedJobIDs, &syncMutex, id)
			if !success {
				failureCount := s.consecutiveFailures.Add(1)
				log.Printf("[NvidiaNimWorker-%d] Batch of %d jobs failed. Consecutive failure count across all workers: %d/4", id, len(batch), failureCount)
				if failureCount >= 4 {
					log.Println("[NvidiaNimService] 4 continuous worker errors reached across all workers. Triggering circuit breaker and stopping NVIDIA NIM worker pool.")
					hasCircuitBroken.Store(true)
					cancelWorkers()
				}
			} else {
				s.consecutiveFailures.Store(0)
				log.Printf("[NvidiaNimWorker-%d] Successfully evaluated batch of %d jobs. Reset shared consecutive failure count to 0.", id, len(batch))
				batchJobIDs := make(map[string]bool)
				for _, item := range batch {
					batchJobIDs[item.JobID] = true
				}
				_ = markJobsAsEvaluatedInDatabase(ctx, s.DB, batchJobIDs)
			}
		}(singleBatch, workerID)
	}

	workerWaitGroup.Wait()

	if len(evaluatedJobIDs) > 0 {
		markErr := markJobsAsEvaluatedInDatabase(ctx, s.DB, evaluatedJobIDs)
		if markErr != nil {
			log.Printf("[NvidiaNimService] Failed marking jobs as evaluated: %v", markErr)
		} else {
			log.Printf("[NvidiaNimService] Pass complete: Marked %d jobs as evaluated in database.", len(evaluatedJobIDs))
		}
	}

	return !hasCircuitBroken.Load()
}

// EvaluateForSingleUser evaluates unmatched jobs for a specific user upon initial onboarding or bio update.
func (s *NvidiaNimService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	_ = s.EvaluateForSingleUserWithResult(ctx, targetUserID)
}

// EvaluateForSingleUserWithResult evaluates single user unmatched jobs and returns true if completed without circuit breaker abort.
func (s *NvidiaNimService) EvaluateForSingleUserWithResult(ctx context.Context, targetUserID string) bool {
	s.queueMutex.RLock()
	if s.isPipelinePermanentlyStopped {
		s.queueMutex.RUnlock()
		log.Printf("[NvidiaNimService] Pipeline is permanently shut down. Skipping evaluation for user %s.", targetUserID)
		return false
	}
	s.queueMutex.RUnlock()

	profile, errProfile := fetchSingleUserProfileByID(ctx, s.DB, targetUserID)
	if errProfile != nil || profile == nil {
		return true
	}

	unmatchedJobs, errJobs := fetchUnmatchedJobsForSingleUser(ctx, s.DB, targetUserID)
	if errJobs != nil || len(unmatchedJobs) == 0 {
		return true
	}

	jobBatches := buildMultiJobTokenBatches(ctx, unmatchedJobs, maxTargetTokensPerPrompt)
	evaluatedJobIDs := make(map[string]bool)
	var syncMutex sync.Mutex
	var workerWaitGroup sync.WaitGroup
	var hasCircuitBroken atomic.Bool

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	for index := 0; index < len(jobBatches); index++ {
		if workerCtx.Err() != nil || hasCircuitBroken.Load() {
			break
		}

		if index > 0 && index%backgroundWorkerBatchSize == 0 {
			workerWaitGroup.Wait()
			time.Sleep(2 * time.Second)
		}

		workerWaitGroup.Add(1)
		singleBatch := jobBatches[index]

		go func(batch []JobSnippetData, id int) {
			defer workerWaitGroup.Done()
			if workerCtx.Err() != nil || hasCircuitBroken.Load() {
				return
			}
			success := s.evaluateJobBatchWithBackoff(workerCtx, []UserProfileData{*profile}, batch, evaluatedJobIDs, &syncMutex, id)
			if !success {
				failureCount := s.consecutiveFailures.Add(1)
				if failureCount >= 4 {
					log.Println("[NvidiaNimService] 4 continuous worker errors reached for single user evaluation. Circuit breaker triggered.")
					hasCircuitBroken.Store(true)
					cancelWorkers()
				}
			} else {
				s.consecutiveFailures.Store(0)
			}
		}(singleBatch, index+1)
	}

	workerWaitGroup.Wait()
	return !hasCircuitBroken.Load()
}

// ProbeHealth verifies that NVIDIA NIM is reachable and generating valid output with a lightweight test prompt.
func (s *NvidiaNimService) ProbeHealth(ctx context.Context) bool {
	if s.IsPipelinePermanentlyStopped() || s.APIKey == "" {
		return false
	}

	probePayload := nvidiaRequest{
		Model: s.ModelName,
		Messages: []nvidiaRequestMessage{
			{
				Role:    "system",
				Content: "You are a JSON-only ping responder. Return ONLY valid JSON: {\"status\":\"ok\"}",
			},
			{
				Role:    "user",
				Content: "Respond with {\"status\":\"ok\"}",
			},
		},
		Temperature: 0.1,
		TopP:        0.95,
		MaxTokens:   50,
		Stream:      false,
		ChatTemplateKwargs: &nvidiaChatTemplateKwargs{
			ThinkingMode: "disabled",
		},
	}

	for attempt := 1; attempt <= maxProbeAttempts; attempt++ {
		select {
		case <-s.tokenBucket:
		case <-ctx.Done():
			return false
		}

		log.Printf("[NvidiaNimProbe] Probing NVIDIA NIM health (Model: %s, Attempt %d/%d)...",
			s.ModelName, attempt, maxProbeAttempts)
		requestStart := time.Now()

		jsonBytes, errMarshal := json.Marshal(probePayload)
		if errMarshal != nil {
			log.Printf("[NvidiaNimProbe] Failed marshaling probe payload: %v", errMarshal)
			return false
		}

		probeCtx, cancelProbe := context.WithTimeout(ctx, httpTimeoutDuration)
		httpRequest, errReq := http.NewRequestWithContext(probeCtx, http.MethodPost, s.Endpoint, bytes.NewBuffer(jsonBytes))
		if errReq != nil {
			cancelProbe()
			log.Printf("[NvidiaNimProbe] Failed creating HTTP probe request: %v", errReq)
			return false
		}

		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("Accept", "application/json")
		httpRequest.Header.Set("Authorization", "Bearer "+s.APIKey)

		httpResponse, errDo := s.HTTPClient.Do(httpRequest)
		if errDo != nil {
			cancelProbe()
			log.Printf("[NvidiaNimProbe] Probe network error after %v (Attempt %d/%d): %v",
				time.Since(requestStart).Truncate(time.Millisecond), attempt, maxProbeAttempts, errDo)
			if attempt < maxProbeAttempts {
				log.Printf("[NvidiaNimProbe] Retrying probe in %d seconds (fixed 1m retry)...", probeRetryDelaySeconds)
				select {
				case <-time.After(time.Duration(probeRetryDelaySeconds) * time.Second):
				case <-ctx.Done():
					return false
				}
			}
			continue
		}

		bodyBytes, errRead := io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()
		cancelProbe()

		if httpResponse.StatusCode == http.StatusUnauthorized || httpResponse.StatusCode == http.StatusForbidden {
			log.Printf("[NvidiaNimProbe] Probe returned unrecoverable HTTP %d after %v (Attempt %d/%d): %s. Failing probe.",
				httpResponse.StatusCode, time.Since(requestStart).Truncate(time.Millisecond), attempt, maxProbeAttempts, string(bodyBytes))
			return false
		}

		if errRead != nil || httpResponse.StatusCode != http.StatusOK {
			log.Printf("[NvidiaNimProbe] Probe returned HTTP %d after %v (Attempt %d/%d): %s",
				httpResponse.StatusCode, time.Since(requestStart).Truncate(time.Millisecond), attempt, maxProbeAttempts, string(bodyBytes))
			if attempt < maxProbeAttempts {
				delay := s.ProbeRetryDelayDuration
				if delay <= 0 {
					delay = time.Duration(probeRetryDelaySeconds) * time.Second
				}
				log.Printf("[NvidiaNimProbe] Retrying probe in %v (fixed 1m retry)...", delay)
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return false
				}
			}
			continue
		}

		var parsedResponse nvidiaAPIResponse
		if errUnmarshal := json.Unmarshal(bodyBytes, &parsedResponse); errUnmarshal == nil && len(parsedResponse.Choices) > 0 {
			cleanText := strings.ToLower(sanitizeJSONResponse(parsedResponse.Choices[0].Message.Content))
			if strings.Contains(cleanText, "ok") || strings.Contains(cleanText, "status") {
				log.Printf("[NvidiaNimProbe] NVIDIA NIM health probe succeeded in %v.", time.Since(requestStart).Truncate(time.Millisecond))
				return true
			}
		}

		bodyText := strings.ToLower(sanitizeJSONResponse(string(bodyBytes)))
		if strings.Contains(bodyText, "ok") || strings.Contains(bodyText, "status") || strings.Contains(bodyText, "results") {
			log.Printf("[NvidiaNimProbe] NVIDIA NIM health probe succeeded in %v.", time.Since(requestStart).Truncate(time.Millisecond))
			return true
		}

		log.Printf("[NvidiaNimProbe] Unexpected probe response content (Attempt %d/%d): %s",
			attempt, maxProbeAttempts, string(bodyBytes))
		if attempt < maxProbeAttempts {
			delay := s.ProbeRetryDelayDuration
			if delay <= 0 {
				delay = time.Duration(probeRetryDelaySeconds) * time.Second
			}
			log.Printf("[NvidiaNimProbe] Retrying probe in %v (fixed 1m retry)...", delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return false
			}
		}
	}

	log.Printf("[NvidiaNimProbe] Health probe failed after %d attempts.", maxProbeAttempts)
	return false
}

// EvaluatePilotBatch tests NVIDIA NIM availability using the fast health probe.
func (s *NvidiaNimService) EvaluatePilotBatch(ctx context.Context, userProfiles []UserProfileData, pilotBatch []JobSnippetData) bool {
	return s.ProbeHealth(ctx)
}

// GenerateCompletion executes a synchronous API request for CV parsing or Overleaf interactive tasks.
func (s *NvidiaNimService) GenerateCompletion(ctx context.Context, promptContent string, systemInstruction string) (string, error) {
	return s.GenerateCompletionWithSchema(ctx, promptContent, systemInstruction, nil)
}

// GenerateCompletionWithSchema executes a synchronous API request with an optional JSON schema constraint in nvext.guided_json.
func (s *NvidiaNimService) GenerateCompletionWithSchema(ctx context.Context, promptContent string, systemInstruction string, jsonSchema any) (string, error) {
	if jsonSchema == nil {
		return s.GenerateCompletionWithNvExt(ctx, promptContent, systemInstruction, nil)
	}
	return s.GenerateCompletionWithNvExt(ctx, promptContent, systemInstruction, &NvidiaNvExt{
		GuidedJSON: jsonSchema,
	})
}

// GenerateCompletionWithNvExt executes a synchronous API request with explicit nvext constraints (guided_json, guided_regex, guided_choice, guided_grammar).
func (s *NvidiaNimService) GenerateCompletionWithNvExt(ctx context.Context, promptContent string, systemInstruction string, nvExt *NvidiaNvExt) (string, error) {
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
		Temperature: 1.0,
		TopP:        0.95,
		Seed:        42,
		ChatTemplateKwargs: &nvidiaChatTemplateKwargs{
			EnableThinking:       false,
			ForceNonemptyContent: true,
		},
		NvExt: nvExt,
	}

	requestStart := time.Now()
	rawResponseBody, errCall := s.executeNvidiaAPIWithRetry(ctx, payload, 0)
	if errCall != nil {
		utils.LogRawAIResponse("GenerateCompletion", s.ModelName, promptContent, "ERROR: "+errCall.Error(), time.Since(requestStart), true)
		return "", errCall
	}

	var parsedResponse nvidiaAPIResponse
	if errUnmarshal := json.Unmarshal(rawResponseBody, &parsedResponse); errUnmarshal != nil {
		utils.LogRawAIResponse("GenerateCompletion", s.ModelName, promptContent, string(rawResponseBody), time.Since(requestStart), true)
		return "", fmt.Errorf("failed unmarshaling NVIDIA response: %w", errUnmarshal)
	}

	if len(parsedResponse.Choices) == 0 {
		utils.LogRawAIResponse("GenerateCompletion", s.ModelName, promptContent, string(rawResponseBody), time.Since(requestStart), true)
		return "", fmt.Errorf("empty choice array returned from NVIDIA API")
	}

	generatedContent := parsedResponse.Choices[0].Message.Content
	utils.LogRawAIResponse("GenerateCompletion", s.ModelName, promptContent, generatedContent, time.Since(requestStart), false)
	return generatedContent, nil
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
		return calculateExactSubwordTokens(text), nil
	}

	httpRequest, errReq := http.NewRequestWithContext(ctx, http.MethodPost, tokenizeEndpoint, bytes.NewBuffer(jsonBytes))
	if errReq != nil {
		return calculateExactSubwordTokens(text), nil
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	if s.APIKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+s.APIKey)
	}

	httpResponse, errDo := s.HTTPClient.Do(httpRequest)
	if errDo != nil {
		return calculateExactSubwordTokens(text), nil
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return calculateExactSubwordTokens(text), nil
	}

	bodyBytes, errRead := io.ReadAll(httpResponse.Body)
	if errRead != nil {
		return calculateExactSubwordTokens(text), nil
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

	return calculateExactSubwordTokens(text), nil
}

func calculateExactSubwordTokens(text string) int {
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

func buildMultiJobTokenBatches(ctx context.Context, allJobs []JobSnippetData, targetTokenBudget int) [][]JobSnippetData {
	var resultBatches [][]JobSnippetData
	var currentBatch []JobSnippetData
	currentTokens := 0

	for _, job := range allJobs {
		snippetText := fmt.Sprintf("ID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription: %s\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))

		itemTokens := calculateExactSubwordTokens(snippetText)

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

func (s *NvidiaNimService) evaluateJobBatchWithBackoff(
	ctx context.Context,
	userProfiles []UserProfileData,
	batch []JobSnippetData,
	evaluatedJobIDs map[string]bool,
	syncMutex *sync.Mutex,
	workerID int,
) bool {
	expectedResultCount := len(batch) * len(userProfiles)
	batchResult, rawText, promptText, ok := s.callBatchEvaluationAPI(ctx, userProfiles, batch, expectedResultCount, workerID)
	if !ok {
		utils.LogRawAIResponse(fmt.Sprintf("Worker-%d-ERROR", workerID), s.ModelName, promptText, rawText, 0, true)
		return false
	}

	if len(batchResult.Results) == 0 {
		log.Printf("[Worker-%d] Model returned empty results for batch of %d jobs — skipping.", workerID, len(batch))
		utils.LogRawAIResponse(fmt.Sprintf("Worker-%d-EMPTY", workerID), s.ModelName, promptText, rawText, 0, true)
		return false
	}

	utils.LogRawAIResponse(fmt.Sprintf("Worker-%d", workerID), s.ModelName, promptText, rawText, 0, false)

	for _, res := range batchResult.Results {
		if res.JobID == "" || res.UserID == "" || !isValidUUIDString(res.JobID) || !isValidUUIDString(res.UserID) {
			continue
		}

		var matchedProfile *UserProfileData
		for profileIndex := range userProfiles {
			if userProfiles[profileIndex].UserID == res.UserID {
				matchedProfile = &userProfiles[profileIndex]
				break
			}
		}

		if matchedProfile != nil {
			applyExperienceAndLocationCaps(&res, matchedProfile)
		}

		isMatch := res.MatchScore >= matchScoreMinThreshold
		upsertErr := upsertUserJobMatchRecord(ctx, s.DB, res.UserID, res.JobID, res.MatchScore, res.MatchReasoning, isMatch, s.ModelName)
		if upsertErr != nil {
			log.Printf("[Worker-%d] Upsert error for user %s job %s: %v", workerID, res.UserID, res.JobID, upsertErr)
		}
		syncMutex.Lock()
		evaluatedJobIDs[res.JobID] = true
		syncMutex.Unlock()
	}

	return true
}

// callBatchEvaluationAPI issues a single NIM API call for the given batch and returns the parsed
// result, the raw response text, the prompt used, and whether the call succeeded structurally.
func (s *NvidiaNimService) callBatchEvaluationAPI(
	ctx context.Context,
	userProfiles []UserProfileData,
	batch []JobSnippetData,
	expectedResultCount int,
	workerID int,
) (nvidiaBatchResponse, string, string, bool) {
	promptText := buildMultiJobPrompt(userProfiles, batch, expectedResultCount)
	messages := []nvidiaRequestMessage{
		{
			Role: "system",
			Content: `You are a JSON-only API. Your entire response must be one single valid JSON object — no prose, no markdown, no code fences, no explanation, no preamble, no postamble. Not a single word outside the JSON.

You MUST output exactly this structure:
{
  "results": [
    {
      "job_id": "<uuid string>",
      "user_id": "<uuid string>",
      "match_score": <integer 0-100>,
      "match_reasoning": "<detailed 2-3 line description detailing location, tech stack, and experience comparison>",
      "inferred_required_yoe": <integer: minimum required years of experience inferred from JD, title, or requirements>,
      "is_matched": <true|false>
    }
  ]
}

Violating this format — outputting any text before or after the JSON, using a different key name, returning an empty array, or omitting any entry — is a critical failure.`,
		},
		{
			Role:    "user",
			Content: promptText,
		},
	}
	payload := nvidiaRequest{
		Model:       s.ModelName,
		Messages:    messages,
		Temperature: 1.0,
		TopP:        0.95,
		MaxTokens:   maxOutputTokensPerBatch,
		Stream:      true,
		ChatTemplateKwargs: &nvidiaChatTemplateKwargs{
			ThinkingMode: "disabled",
		},
	}

	responseBytes, errCall := s.executeNvidiaAPIWithRetry(ctx, payload, workerID)
	if errCall != nil {
		log.Printf("[Worker-%d] API call failed for batch of %d jobs: %v", workerID, len(batch), errCall)
		return nvidiaBatchResponse{}, "", promptText, false
	}

	var parsedResponse nvidiaAPIResponse
	if errUnmarshal := json.Unmarshal(responseBytes, &parsedResponse); errUnmarshal != nil {
		log.Printf("[Worker-%d] Unmarshal error for batch of %d jobs: %v", workerID, len(batch), errUnmarshal)
		return nvidiaBatchResponse{}, "", promptText, false
	}

	if len(parsedResponse.Choices) == 0 {
		log.Printf("[Worker-%d] Empty choices array for batch of %d jobs", workerID, len(batch))
		return nvidiaBatchResponse{}, "", promptText, false
	}

	rawText := parsedResponse.Choices[0].Message.Content
	cleanJSON := sanitizeJSONResponse(rawText)

	var batchResult nvidiaBatchResponse
	if errJSON := json.Unmarshal([]byte(cleanJSON), &batchResult); errJSON != nil {
		log.Printf("[Worker-%d] JSON parse error for batch of %d jobs: %v", workerID, len(batch), errJSON)
		utils.LogRawAIResponse(fmt.Sprintf("Worker-%d-ERROR", workerID), s.ModelName, promptText, "RAW_OUTPUT:\n"+rawText+"\n\nCLEANED_JSON:\n"+cleanJSON, 0, true)
		return nvidiaBatchResponse{}, rawText, promptText, false
	}

	return batchResult, rawText, promptText, true
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
	backoffDuration := time.Duration(retryBackoffBaseSeconds) * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-s.tokenBucket:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		log.Printf("[Worker-%d] Posting request to %s (Model: %s, Payload: %d bytes, Attempt: %d/%d)...",
			workerID, s.Endpoint, payload.Model, len(jsonBytes), attempt, maxAttempts)
		requestStart := time.Now()

		httpRequest, errReq := http.NewRequestWithContext(ctx, http.MethodPost, s.Endpoint, bytes.NewBuffer(jsonBytes))
		if errReq != nil {
			return nil, fmt.Errorf("failed creating HTTP request: %w", errReq)
		}

		httpRequest.Header.Set("Content-Type", "application/json")
		if payload.Stream {
			httpRequest.Header.Set("Accept", "text/event-stream")
		} else {
			httpRequest.Header.Set("Accept", "application/json")
		}
		httpRequest.Header.Set("Authorization", "Bearer "+s.APIKey)

		httpResponse, errDo := s.HTTPClient.Do(httpRequest)
		if errDo != nil {
			log.Printf("[Worker-%d] Request error after %v: %v", workerID, time.Since(requestStart), errDo)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("HTTP request failed after %d attempts: %w", maxAttempts, errDo)
			}
			sleepWithJitter(backoffDuration)
			backoffDuration = capBackoff(backoffDuration * 2)
			continue
		}

		if httpResponse.StatusCode == http.StatusTooManyRequests {
			bodyBytes, _ := io.ReadAll(httpResponse.Body)
			retryAfterHeader := httpResponse.Header.Get("Retry-After")
			httpResponse.Body.Close()
			log.Printf("[Worker-%d] API returned HTTP 429 (Attempt %d/%d): %s", workerID, attempt, maxAttempts, string(bodyBytes))
			if attempt == maxAttempts {
				return nil, fmt.Errorf("NVIDIA API error HTTP 429 after %d attempts: %s", maxAttempts, string(bodyBytes))
			}
			sleepDuration := backoffDuration
			if retryAfterSeconds, parseErr := strconv.Atoi(retryAfterHeader); parseErr == nil && retryAfterSeconds > 0 {
				retryAfterDuration := time.Duration(retryAfterSeconds) * time.Second
				if retryAfterDuration > sleepDuration {
					sleepDuration = retryAfterDuration
				}
			}
			log.Printf("[Worker-%d] Backing off for %v before retry...", workerID, sleepDuration.Truncate(time.Millisecond))
			sleepWithJitter(sleepDuration)
			backoffDuration = capBackoff(backoffDuration * 2)
			continue
		}

		if httpResponse.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(httpResponse.Body)
			httpResponse.Body.Close()
			log.Printf("[Worker-%d] API returned error HTTP %d (Attempt %d/%d): %s", workerID, httpResponse.StatusCode, attempt, maxAttempts, string(bodyBytes))
			if attempt == maxAttempts {
				return nil, fmt.Errorf("NVIDIA API error HTTP %d: %s", httpResponse.StatusCode, string(bodyBytes))
			}
			sleepWithJitter(backoffDuration)
			backoffDuration = capBackoff(backoffDuration * 2)
			continue
		}

		if payload.Stream {
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
							reasoningText := delta.ReasoningContent
							if reasoningText == "" {
								reasoningText = delta.Reasoning
							}

							if reasoningText != "" || delta.Content != "" {
								lastChunkTime = time.Now()
							}
							if reasoningText != "" {
								reasoningTokenCount += len(reasoningText)
								if !firstTokenReceived {
									firstTokenReceived = true
									log.Printf("[STREAM] [Worker-%d] First reasoning token received after %v", workerID, time.Since(requestStart))
								}
							}
							if delta.Content != "" {
								if !firstTokenReceived {
									firstTokenReceived = true
									log.Printf("[STREAM] [Worker-%d] First content token received after %v", workerID, time.Since(requestStart))
								}
								accumulatedBuilder.WriteString(delta.Content)
							}

							if time.Since(lastLogTime) >= 10*time.Second {
								log.Printf("[STREAM] [Worker-%d] Live Progress: Reasoning bytes: %d, Response bytes: %d (Elapsed: %v)",
									workerID, reasoningTokenCount, accumulatedBuilder.Len(), time.Since(requestStart).Truncate(time.Second))
								lastLogTime = time.Now()
							}
						}
					}
				}
			}
			httpResponse.Body.Close()

			accumulatedJSON := accumulatedBuilder.String()
			log.Printf("[STREAM] [Worker-%d] Streaming finished in %v (Reasoning bytes: %d, Response size: %d bytes)",
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
		return bodyBytes, nil
	}

	return nil, fmt.Errorf("NVIDIA API call exceeded max retries")
}

// sleepWithJitter sleeps for the given duration scaled by a uniform random jitter in
// [1 - retryBackoffJitterFraction, 1 + retryBackoffJitterFraction], so simultaneous
// workers desynchronise their retries and avoid thundering-herd re-collisions.
func sleepWithJitter(duration time.Duration) {
	jitterMultiplier := 1.0 + (rand.Float64()*2-1)*retryBackoffJitterFraction
	time.Sleep(time.Duration(float64(duration) * jitterMultiplier))
}

// capBackoff returns the given duration clamped to retryBackoffMaxSeconds.
func capBackoff(duration time.Duration) time.Duration {
	maximum := time.Duration(retryBackoffMaxSeconds) * time.Second
	if duration > maximum {
		return maximum
	}
	return duration
}

// buildMultiJobPrompt constructs the evaluation prompt for a batch of jobs against a set of user profiles.
func buildMultiJobPrompt(userProfiles []UserProfileData, jobsBatch []JobSnippetData, expectedResultCount int) string {
	var builder strings.Builder

	currentTimeText := time.Now().Format("January 2006")

	builder.WriteString("Return ONLY a raw JSON object formatted as follows:\n")
	builder.WriteString("{\n  \"results\": [\n    {\n      \"job_id\": \"<job_id string copied verbatim from JOB LISTINGS below>\",\n      \"user_id\": \"<user_id string copied verbatim from CANDIDATE PROFILES below>\",\n      \"match_score\": 85,\n      \"match_reasoning\": \"Highly detailed 2-3 line reasoning specifying exact tech stack overlap, candidate base location vs job requirement, and YoE comparison details.\",\n      \"inferred_required_yoe\": 4,\n      \"is_matched\": true\n    }\n  ]\n}\n\n")
	fmt.Fprintf(&builder, "IMPORTANT: You MUST return exactly %d entries in the \"results\" array — one for every (job × candidate) pair listed below. An empty array or partial array is invalid.\n\n", expectedResultCount)

	builder.WriteString("SCORING RULES — apply in strict priority order; a higher-priority penalty overrides skill match entirely:\n")
	builder.WriteString("1. LOCATION MISMATCH (HARD CAP): Candidate is India-based. Any job that is US Onsite, US Hybrid, or US-only Remote (explicitly excludes non-US applicants) → cap score at 0–15, regardless of any other signal.\n")
	builder.WriteString("2. EXPERIENCE GAP (HARD CAP — highest priority penalty):\n")
	fmt.Fprintf(&builder, "   - Use the provided Candidate Years of Experience (YoE) calculated from their resume, or infer it from the profile/resume relative to the current evaluation date: %s. Infer job's minimum required YoE from the JD (look for phrases like '4+ years', '5-7 years experience', etc.).\n", currentTimeText)
	builder.WriteString("   - If no explicit years of experience are mentioned, infer the required YoE based on the role level norms: Intern/Co-op/Apprentice = 0 YoE; Junior/Associate = 0-2 YoE; Mid-Level/SWE = 2-4 YoE; Senior/Lead/Manager = 4-6 YoE; Staff/Architect = 6-8 YoE; Principal/Director/VP = 8+ YoE.\n")
	builder.WriteString("   - If the job's minimum required YoE > (candidate YoE + 3): cap score at 0–25. Skill match is IRRELEVANT — a candidate cannot overcome a 3+ year experience deficit.\n")
	builder.WriteString("   - If the job's minimum required YoE is (candidate YoE + 1) to (candidate YoE + 2): cap score at 45–65. This is a stretch role the candidate cannot realistically get.\n")
	builder.WriteString("   - If candidate YoE is perfectly aligned with the required range (candidate YoE >= minimum required YoE), award a strong bonus (+15 to +20 points) to the match score if the tech stack matches.\n")
	builder.WriteString("3. HIGH MATCH (75–100): ONLY for roles where candidate YoE meets or exceeds the minimum, role location matches, the candidate has a strong tech stack overlap, and they receive the experience range alignment bonus.\n")
	builder.WriteString("4. DETAILED REASONING REQUIREMENT: The 'match_reasoning' must be a minimum of 4-5 lines of text explaining location verification, tech stack match, and YoE ranges.\n\n")

	builder.WriteString("### CANDIDATE PROFILES\n")
	for _, profile := range userProfiles {
		combinedProfileText := profile.ParsedBio
		if profile.MasterCVText != "" {
			combinedProfileText += "\n\nMaster CV / Full Experience Context:\n" + profile.MasterCVText
		}
		fmt.Fprintf(&builder, "User ID: %s\nCandidate Years of Experience (YoE): %d\nProfile & Resume Context:\n%s\nPreferred Roles: %s\nPreferred Locations: %s\nWork Model: %s\n\n",
			profile.UserID, profile.ExperienceYears, combinedProfileText, strings.Join(profile.PreferredRoles, ", "), strings.Join(profile.PreferredLocations, ", "), profile.WorkModel)
	}

	builder.WriteString("### JOB LISTINGS TO EVALUATE\n")
	for _, job := range jobsBatch {
		fmt.Fprintf(&builder, "---\nID: %s\nTitle: %s\nCompany: %s\nLocation: %s\nDescription:\n%s\n\n",
			job.JobID, job.Title, job.Company, job.Location, truncateTextString(job.Description, maxJobDescriptionLength))
	}

	fmt.Fprintf(&builder, "NOW OUTPUT THE JSON. Exactly %d entries. No other text. Start your response with '{' and end it with '}'.\n", expectedResultCount)

	return builder.String()
}

// FetchAllUserProfiles retrieves all users with AI matching enabled.
func (s *NvidiaNimService) FetchAllUserProfiles(ctx context.Context) ([]UserProfileData, error) {
	return fetchAllActiveUserProfiles(ctx, s.DB)
}

// FetchSingleUserProfile retrieves candidate background data for a specific user ID.
func (s *NvidiaNimService) FetchSingleUserProfile(ctx context.Context, targetUserID string) (*UserProfileData, error) {
	return fetchSingleUserProfileByID(ctx, s.DB, targetUserID)
}

// FetchUnevaluatedJobs retrieves recent unevaluated job listings from the database.
func (s *NvidiaNimService) FetchUnevaluatedJobs(ctx context.Context) ([]JobSnippetData, error) {
	return fetchRecentUnevaluatedJobs(ctx, s.DB)
}

// FetchJobsUnmatchedForUser retrieves unevaluated jobs specific to a given user.
func (s *NvidiaNimService) FetchJobsUnmatchedForUser(ctx context.Context, targetUserID string) ([]JobSnippetData, error) {
	return fetchUnmatchedJobsForSingleUser(ctx, s.DB, targetUserID)
}

func fetchAllActiveUserProfiles(ctx context.Context, databasePool *pgxpool.Pool) ([]UserProfileData, error) {
	if databasePool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}
	sqlQuery := `
		SELECT u.id, u.primary_email, COALESCE(up.bio_experience_text, ''), COALESCE(up.master_cv_text, ''), COALESCE(up.target_roles, '[]'),
		       COALESCE(up.target_locations, '[]'), COALESCE(up.work_models->>0, ''), 0
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.ai_matching_enabled = true;
	`
	rows, errQuery := databasePool.Query(ctx, sqlQuery)
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

func fetchSingleUserProfileByID(ctx context.Context, databasePool *pgxpool.Pool, targetUserID string) (*UserProfileData, error) {
	if databasePool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}
	sqlQuery := `
		SELECT u.id, u.primary_email, COALESCE(up.bio_experience_text, ''), COALESCE(up.master_cv_text, ''), COALESCE(up.target_roles, '[]'),
		       COALESCE(up.target_locations, '[]'), COALESCE(up.work_models->>0, ''), 0
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.id = $1;
	`
	var item UserProfileData
	scanErr := databasePool.QueryRow(ctx, sqlQuery, targetUserID).Scan(&item.UserID, &item.Email, &item.ParsedBio, &item.MasterCVText, &item.PreferredRoles, &item.PreferredLocations, &item.WorkModel, &item.ExperienceYears)
	if scanErr != nil {
		return nil, scanErr
	}
	item.ExperienceYears = calculateTotalExperienceYears(item.MasterCVText)
	return &item, nil
}

func fetchRecentUnevaluatedJobs(ctx context.Context, databasePool *pgxpool.Pool) ([]JobSnippetData, error) {
	if databasePool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}
	sqlQuery := `
		SELECT j.id, j.title, COALESCE(c.name, ''), COALESCE(j.location, ''), COALESCE(j.raw_desc, '')
		FROM jobs j
		LEFT JOIN companies c ON j.company_id = c.id
		WHERE j.ai_evaluated = false
		  AND j.scraped_at >= NOW() - INTERVAL '14 days'
		ORDER BY j.scraped_at DESC;
	`
	rows, errQuery := databasePool.Query(ctx, sqlQuery)
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

func fetchUnmatchedJobsForSingleUser(ctx context.Context, databasePool *pgxpool.Pool, targetUserID string) ([]JobSnippetData, error) {
	if databasePool == nil {
		return nil, fmt.Errorf("database pool is not initialized")
	}
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
	rows, errQuery := databasePool.Query(ctx, sqlQuery, targetUserID)
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

func markJobsAsEvaluatedInDatabase(ctx context.Context, databasePool *pgxpool.Pool, jobIDsMap map[string]bool) error {
	if databasePool == nil || len(jobIDsMap) == 0 {
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
	_, errExec := databasePool.Exec(ctx, sqlQuery, keysList)
	return errExec
}

func upsertUserJobMatchRecord(ctx context.Context, databasePool *pgxpool.Pool, userID, jobID string, score int, reasoning string, isMatch bool, modelName string) error {
	if databasePool == nil {
		return nil
	}
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
	_, errExec := databasePool.Exec(ctx, sqlQuery, userID, jobID, score, reasoning, isMatch, modelName)
	return errExec
}

func sanitizeJSONResponse(textContent string) string {
	clean := strings.TrimSpace(textContent)
	if strings.HasPrefix(clean, "```json") {
		clean = strings.TrimPrefix(clean, "```json")
		clean = strings.TrimSuffix(clean, "```")
		clean = strings.TrimSpace(clean)
	} else if strings.HasPrefix(clean, "```") {
		clean = strings.TrimPrefix(clean, "```")
		clean = strings.TrimSuffix(clean, "```")
		clean = strings.TrimSpace(clean)
	}

	firstBrace := strings.Index(clean, "{")
	lastBrace := strings.LastIndex(clean, "}")
	if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
		return clean[firstBrace : lastBrace+1]
	}

	firstBracket := strings.Index(clean, "[")
	lastBracket := strings.LastIndex(clean, "]")
	if firstBracket != -1 && lastBracket != -1 && lastBracket > firstBracket {
		return clean[firstBracket : lastBracket+1]
	}

	return clean
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

func SanitizeJSONResponseForTest(s string) string {
	return sanitizeJSONResponse(s)
}

type cvExperience struct {
	Duration string `json:"duration"`
}

type cvSchema struct {
	Experiences []cvExperience `json:"experiences"`
	Experience  []cvExperience `json:"experience"`
}

func calculateTotalExperienceYears(cvText string) int {
	if cvText == "" {
		return 0
	}
	var schema cvSchema
	if err := json.Unmarshal([]byte(cvText), &schema); err != nil {
		return 0
	}
	var earliestStart time.Time
	var latestEnd time.Time
	hasExperience := false

	experiencesList := schema.Experiences
	if len(experiencesList) == 0 {
		experiencesList = schema.Experience
	}

	for _, exp := range experiencesList {
		start, end, parsed := parseDurationSpan(exp.Duration)
		if !parsed {
			continue
		}
		if !hasExperience {
			earliestStart = start
			latestEnd = end
			hasExperience = true
			continue
		}
		if start.Before(earliestStart) {
			earliestStart = start
		}
		if end.After(latestEnd) {
			latestEnd = end
		}
	}
	if !hasExperience {
		return 0
	}
	durationSpan := latestEnd.Sub(earliestStart)
	years := int(durationSpan.Hours() / 24 / 365)
	if years < 0 {
		return 0
	}
	return years
}

func parseDurationSpan(duration string) (time.Time, time.Time, bool) {
	duration = strings.ReplaceAll(duration, "–", "-")
	duration = strings.ReplaceAll(duration, "to", "-")
	parts := strings.Split(duration, "-")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, false
	}
	start := parseDateString(parts[0])
	end := parseDateString(parts[1])
	if start.IsZero() || end.IsZero() {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func parseDateString(dateStr string) time.Time {
	dateStr = strings.TrimSpace(strings.ToLower(dateStr))
	if dateStr == "present" || dateStr == "current" || dateStr == "" {
		return time.Now()
	}
	months := map[string]int{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
		"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
		"january": 1, "february": 2, "march": 3, "april": 4, "june": 6,
		"july": 7, "august": 8, "september": 9, "october": 10, "november": 11, "december": 12,
	}
	fields := strings.Fields(dateStr)
	var monthVal int = 1
	var yearVal int = 0
	for _, field := range fields {
		if val, exists := months[field]; exists {
			monthVal = val
			continue
		}
		var parsedYear int
		if _, err := fmt.Sscanf(field, "%d", &parsedYear); err == nil {
			if parsedYear < 100 {
				if parsedYear > 50 {
					yearVal = 1900 + parsedYear
				} else {
					yearVal = 2000 + parsedYear
				}
			} else {
				yearVal = parsedYear
			}
		}
	}
	if yearVal == 0 {
		return time.Time{}
	}
	return time.Date(yearVal, time.Month(monthVal), 1, 0, 0, 0, 0, time.UTC)
}

func applyExperienceAndLocationCaps(result *BatchMatchResultItem, profile *UserProfileData) {
	if profile == nil {
		return
	}
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

func CalculateTotalExperienceYearsForTest(cvText string) int {
	return calculateTotalExperienceYears(cvText)
}
