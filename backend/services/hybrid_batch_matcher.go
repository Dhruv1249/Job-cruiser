// Package services provides orchestration for AI batch matchers.
package services

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
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

// EvaluatePendingForAllUsers probes NVIDIA NIM with a fast health check; if NIM fails, it automatically falls back to Gemini.
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
		log.Println("[HybridMatcher] Probing NVIDIA NIM with fast health probe...")
		probeSuccess := s.NvidiaNimService.ProbeHealth(ctx)
		if probeSuccess {
			log.Println("[HybridMatcher] NVIDIA NIM health probe succeeded. Dispatching full evaluation worker pool...")
			fullPassSuccess := s.NvidiaNimService.EvaluatePendingForAllUsersWithResult(ctx)
			if fullPassSuccess {
				return
			}
			log.Println("[HybridMatcher] NVIDIA NIM encountered 4 continuous errors across workers. Automatically falling back to Gemini Batch Service...")
		} else {
			log.Println("[HybridMatcher] NVIDIA NIM health probe failed after retries. Automatically falling back to Gemini Batch Service...")
		}
	}

	if s.GeminiBatchService != nil {
		log.Println("[HybridMatcher] Delegating job evaluation pass to Gemini Batch Service fallback...")
		s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
	}
}

// EvaluateForSingleUser probes NVIDIA NIM with a fast health check; if NIM fails, it automatically falls back to Gemini.
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
		log.Printf("[HybridMatcher] Probing NVIDIA NIM before evaluating single user %s...", targetUserID)
		probeSuccess := s.NvidiaNimService.ProbeHealth(ctx)
		if probeSuccess {
			log.Printf("[HybridMatcher] NVIDIA NIM health probe succeeded. Evaluating single user %s with NVIDIA NIM engine...", targetUserID)
			userPassSuccess := s.NvidiaNimService.EvaluateForSingleUserWithResult(ctx, targetUserID)
			if userPassSuccess {
				return
			}
			log.Printf("[HybridMatcher] NVIDIA NIM encountered 4 continuous errors during evaluation for user %s. Automatically falling back to Gemini Batch Service...", targetUserID)
		} else {
			log.Printf("[HybridMatcher] NVIDIA NIM health probe failed. Automatically falling back to Gemini Batch Service for user %s...", targetUserID)
		}
	}

	if s.GeminiBatchService != nil {
		log.Printf("[HybridMatcher] Delegating single-user evaluation for user %s to Gemini Batch Service...", targetUserID)
		s.GeminiBatchService.EvaluateForSingleUser(ctx, targetUserID)
	}
}

// StartBackgroundScheduler starts a 10-minute periodic background scheduler evaluating pending matches through the hybrid pipeline.
func (s *HybridBatchMatchService) StartBackgroundScheduler(ctx context.Context) {
	tickerDuration := 10 * time.Minute
	ticker := time.NewTicker(tickerDuration)
	log.Printf("[HybridMatcher] Started %v background ticker scheduler.", tickerDuration)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				s.EvaluatePendingForAllUsers(ctx)
			}
		}
	}()
}
