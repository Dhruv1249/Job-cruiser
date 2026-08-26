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
	return &HybridBatchMatchService{
		NvidiaNimService:   nvidiaNim,
		GeminiBatchService: geminiBatch,
	}
}

// EvaluatePendingForAllUsers delegates batch evaluation pass to active provider with automatic fallback.
func (s *HybridBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	primaryProvider := strings.ToLower(strings.TrimSpace(os.Getenv("PRIMARY_AI_PROVIDER")))

	if primaryProvider == "gemini" {
		if s.GeminiBatchService != nil {
			log.Println("[HybridMatcher] PRIMARY_AI_PROVIDER is set to gemini. Delegating to Gemini Batch Service...")
			s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
			return
		}
	}

	if s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" {
		log.Println("[HybridMatcher] Delegating batch evaluation pass to NVIDIA NIM Engine...")
		s.NvidiaNimService.EvaluatePendingForAllUsers(ctx)
		return
	}

	if s.GeminiBatchService != nil {
		log.Println("[HybridMatcher] NVIDIA NIM unconfigured or unavailable. Falling back to Gemini Batch Service...")
		s.GeminiBatchService.EvaluatePendingForAllUsers(ctx)
	}
}

// EvaluateForSingleUser delegates single-user evaluation pass with automatic fallback.
func (s *HybridBatchMatchService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	primaryProvider := strings.ToLower(strings.TrimSpace(os.Getenv("PRIMARY_AI_PROVIDER")))

	if primaryProvider == "gemini" {
		if s.GeminiBatchService != nil {
			log.Printf("[HybridMatcher] PRIMARY_AI_PROVIDER is set to gemini. Evaluating single user %s via Gemini...", targetUserID)
			s.GeminiBatchService.EvaluateForSingleUser(ctx, targetUserID)
			return
		}
	}

	if s.NvidiaNimService != nil && s.NvidiaNimService.APIKey != "" {
		log.Printf("[HybridMatcher] Delegating single-user evaluation pass for user %s to NVIDIA NIM Engine...", targetUserID)
		s.NvidiaNimService.EvaluateForSingleUser(ctx, targetUserID)
		return
	}

	if s.GeminiBatchService != nil {
		log.Printf("[HybridMatcher] NVIDIA NIM unavailable. Falling back to Gemini single-user evaluation for user %s...", targetUserID)
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
