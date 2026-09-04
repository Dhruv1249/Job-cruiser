// Package services provides orchestration for AI batch matchers.
package services

import (
	"context"
	"log"
	"sync"
	"time"
)

// BatchMatchEvaluator defines the common contract for AI job matching engines.
type BatchMatchEvaluator interface {
	EvaluatePendingForAllUsers(ctx context.Context)
	EvaluateForSingleUser(ctx context.Context, userID string)
	StartBackgroundScheduler(ctx context.Context)
	IsPipelinePermanentlyStopped() bool
	ResetPipeline()
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

// EvaluatePendingForAllUsers runs NVIDIA NIM and Gemini batch matchers concurrently in parallel.
func (s *HybridBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	if s.GeminiBatchService != nil && s.GeminiBatchService.IsPipelinePermanentlyStopped() {
		log.Println("[HybridMatcher] AI evaluation pipeline is permanently shut down. Skipping evaluation pass.")
		return
	}

	nimActive := s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" && !s.NvidiaNimService.IsPipelinePermanentlyStopped()
	geminiActive := s.GeminiBatchService != nil && s.GeminiBatchService.APIKey != "" && !s.GeminiBatchService.IsPipelinePermanentlyStopped()

	if nimActive && geminiActive {
		log.Println("[HybridMatcher] Running NVIDIA NIM and Gemini Batch Service concurrently in parallel...")
		var waitGroup sync.WaitGroup
		waitGroup.Add(2)

		go func() {
			defer waitGroup.Done()
			s.NvidiaNimService.EvaluatePendingForAllUsers(ctx)
		}()

		go func() {
			defer waitGroup.Done()
			s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
		}()

		waitGroup.Wait()
		return
	}

	if nimActive {
		log.Println("[HybridMatcher] Dispatching evaluation pass with NVIDIA NIM...")
		s.NvidiaNimService.EvaluatePendingForAllUsers(ctx)
		return
	}

	if geminiActive {
		log.Println("[HybridMatcher] Dispatching evaluation pass with Gemini Batch Service...")
		s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
	}
}

// EvaluateForSingleUser runs NVIDIA NIM and Gemini batch matchers concurrently in parallel for a single user.
func (s *HybridBatchMatchService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	if s.GeminiBatchService != nil && s.GeminiBatchService.IsPipelinePermanentlyStopped() {
		log.Printf("[HybridMatcher] AI evaluation pipeline is permanently shut down. Skipping evaluation for user %s.", targetUserID)
		return
	}

	nimActive := s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" && !s.NvidiaNimService.IsPipelinePermanentlyStopped()
	geminiActive := s.GeminiBatchService != nil && s.GeminiBatchService.APIKey != "" && !s.GeminiBatchService.IsPipelinePermanentlyStopped()

	if nimActive && geminiActive {
		log.Printf("[HybridMatcher] Running NVIDIA NIM and Gemini concurrently in parallel for single user %s...", targetUserID)
		var waitGroup sync.WaitGroup
		waitGroup.Add(2)

		go func() {
			defer waitGroup.Done()
			s.NvidiaNimService.EvaluateForSingleUser(ctx, targetUserID)
		}()

		go func() {
			defer waitGroup.Done()
			s.GeminiBatchService.EvaluateForSingleUser(ctx, targetUserID)
		}()

		waitGroup.Wait()
		return
	}

	if nimActive {
		log.Printf("[HybridMatcher] Evaluating single user %s with NVIDIA NIM...", targetUserID)
		s.NvidiaNimService.EvaluateForSingleUser(ctx, targetUserID)
		return
	}

	if geminiActive {
		log.Printf("[HybridMatcher] Evaluating single user %s with Gemini Batch Service...", targetUserID)
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

/*
IsPipelinePermanentlyStopped returns true if the Gemini matching pipeline has been permanently shut down.
*/
func (s *HybridBatchMatchService) IsPipelinePermanentlyStopped() bool {
	if s.GeminiBatchService != nil && s.GeminiBatchService.IsPipelinePermanentlyStopped() {
		return true
	}
	return false
}

/*
ResetPipeline resets circuit breaker counters and re-enables all matching engines across the hybrid pipeline.
*/
func (s *HybridBatchMatchService) ResetPipeline() {
	if s.GeminiBatchService != nil {
		s.GeminiBatchService.ResetPipeline()
	}
	if s.NvidiaNimService != nil {
		s.NvidiaNimService.ResetPipeline()
	}
	log.Println("[HybridMatcher] AI evaluation pipeline has been fully reset and resumed.")
}
