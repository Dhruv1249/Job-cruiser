// Package services provides orchestration for AI batch matchers.
package services

import (
	"context"
	"log"
	"os"
	"strings"
)

// BatchMatchEvaluator defines the common contract for AI job matching engines.
type BatchMatchEvaluator interface {
	EvaluatePendingForAllUsers(ctx context.Context)
	EvaluateForSingleUser(ctx context.Context, userID string)
	StartBackgroundScheduler(ctx context.Context)
}

// HybridBatchMatchService delegates matching requests to NVIDIA NIM or Gemini batch services.
type HybridBatchMatchService struct {
	NvidiaNimService   *NvidiaNimService
	GeminiBatchService *GeminiBatchMatchService
}

// NewHybridBatchMatchService initializes a hybrid batch matcher service with NVIDIA NIM and Gemini fallback engines.
func NewHybridBatchMatchService(nvidiaNim *NvidiaNimService, geminiBatch *GeminiBatchMatchService) *HybridBatchMatchService {
	service := &HybridBatchMatchService{
		NvidiaNimService:   nvidiaNim,
		GeminiBatchService: geminiBatch,
	}

	if geminiBatch != nil && nvidiaNim != nil {
		geminiBatch.OnPipelineShutdown = func() {
			nvidiaNim.SetPipelinePermanentlyStopped(true)
		}
	}

	return service
}

// EvaluatePendingForAllUsers probes NVIDIA NIM with the first 10 jobs; if NIM fails, it automatically falls back to Gemini.
func (s *HybridBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	if s.GeminiBatchService != nil && s.GeminiBatchService.IsPipelinePermanentlyStopped() {
		log.Println("[HybridMatcher] AI evaluation pipeline is permanently shut down. Skipping evaluation pass.")
		return
	}

	primaryProvider := strings.ToLower(strings.TrimSpace(os.Getenv("PRIMARY_AI_PROVIDER")))

	if primaryProvider == "gemini" {
		if s.GeminiBatchService != nil {
			log.Println("[HybridMatcher] PRIMARY_AI_PROVIDER is set to gemini. Delegating to Gemini Batch Service...")
			s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
			return
		}
	}

	if s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" {
		log.Println("[HybridMatcher] Probing NVIDIA NIM with pilot batch of first 10 jobs...")
		userProfiles, errProfiles := s.NvidiaNimService.FetchAllUserProfiles(ctx)
		if errProfiles == nil && len(userProfiles) > 0 {
			pendingJobs, errJobs := s.NvidiaNimService.FetchUnevaluatedJobs(ctx)
			if errJobs == nil && len(pendingJobs) > 0 {
				pilotBatchSize := 10
				if len(pendingJobs) < pilotBatchSize {
					pilotBatchSize = len(pendingJobs)
				}
				pilotBatch := pendingJobs[:pilotBatchSize]
				pilotSuccess := s.NvidiaNimService.EvaluatePilotBatch(ctx, userProfiles, pilotBatch)
				if pilotSuccess {
					log.Println("[HybridMatcher] NVIDIA NIM pilot batch succeeded. Continuing full evaluation pass with NVIDIA NIM engine...")
					fullPassSuccess := s.NvidiaNimService.EvaluatePendingForAllUsersWithResult(ctx)
					if fullPassSuccess {
						return
					}
					log.Println("[HybridMatcher] NVIDIA NIM encountered 4 continuous errors across workers. Automatically falling back to Gemini Batch Service...")
				} else {
					log.Println("[HybridMatcher] NVIDIA NIM pilot batch of 10 jobs failed. Automatically falling back to Gemini Batch Service...")
				}
			}
		}
	}

	if s.GeminiBatchService != nil {
		log.Println("[HybridMatcher] Delegating job evaluation pass to Gemini Batch Service fallback...")
		s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
	}
}

// EvaluateForSingleUser probes NVIDIA NIM with the user's first 10 jobs; if NIM fails, it automatically falls back to Gemini.
func (s *HybridBatchMatchService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	if s.GeminiBatchService != nil && s.GeminiBatchService.IsPipelinePermanentlyStopped() {
		log.Printf("[HybridMatcher] AI evaluation pipeline is permanently shut down. Skipping evaluation for user %s.", targetUserID)
		return
	}

	primaryProvider := strings.ToLower(strings.TrimSpace(os.Getenv("PRIMARY_AI_PROVIDER")))

	if primaryProvider == "gemini" {
		if s.GeminiBatchService != nil {
			log.Printf("[HybridMatcher] PRIMARY_AI_PROVIDER is set to gemini. Evaluating single user %s via Gemini...", targetUserID)
			s.GeminiBatchService.EvaluateForSingleUser(ctx, targetUserID)
			return
		}
	}

	if s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" {
		profile, errProfile := s.NvidiaNimService.FetchSingleUserProfile(ctx, targetUserID)
		if errProfile == nil && profile != nil {
			unmatchedJobs, errJobs := s.NvidiaNimService.FetchJobsUnmatchedForUser(ctx, targetUserID)
			if errJobs == nil && len(unmatchedJobs) > 0 {
				pilotBatchSize := 10
				if len(unmatchedJobs) < pilotBatchSize {
					pilotBatchSize = len(unmatchedJobs)
				}
				pilotBatch := unmatchedJobs[:pilotBatchSize]
				pilotSuccess := s.NvidiaNimService.EvaluatePilotBatch(ctx, []UserProfileData{*profile}, pilotBatch)
				if pilotSuccess {
					log.Printf("[HybridMatcher] NVIDIA NIM pilot batch succeeded for user %s. Continuing with NVIDIA NIM engine...", targetUserID)
					userPassSuccess := s.NvidiaNimService.EvaluateForSingleUserWithResult(ctx, targetUserID)
					if userPassSuccess {
						return
					}
					log.Printf("[HybridMatcher] NVIDIA NIM encountered 4 continuous errors during evaluation for user %s. Automatically falling back to Gemini Batch Service...", targetUserID)
				} else {
					log.Printf("[HybridMatcher] NVIDIA NIM pilot batch failed for user %s. Automatically falling back to Gemini Batch Service...", targetUserID)
				}
			}
		}
	}

	if s.GeminiBatchService != nil {
		log.Printf("[HybridMatcher] Delegating single-user evaluation for user %s to Gemini Batch Service...", targetUserID)
		s.GeminiBatchService.EvaluateForSingleUser(ctx, targetUserID)
	}
}

// StartBackgroundScheduler starts background scheduler for the active engine.
func (s *HybridBatchMatchService) StartBackgroundScheduler(ctx context.Context) {
	primaryProvider := strings.ToLower(strings.TrimSpace(os.Getenv("PRIMARY_AI_PROVIDER")))

	if primaryProvider == "gemini" {
		if s.GeminiBatchService != nil {
			log.Println("[HybridMatcher] Starting Gemini Batch background scheduler...")
			s.GeminiBatchService.StartBackgroundScheduler(ctx)
			return
		}
	}

	if s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" {
		log.Println("[HybridMatcher] Starting NVIDIA NIM background scheduler...")
		s.NvidiaNimService.StartBackgroundScheduler(ctx)
		return
	}

	if s.GeminiBatchService != nil {
		log.Println("[HybridMatcher] Starting Gemini Batch fallback background scheduler...")
		s.GeminiBatchService.StartBackgroundScheduler(ctx)
	}
}
