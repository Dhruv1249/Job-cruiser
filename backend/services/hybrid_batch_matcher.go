// Package services provides orchestration for AI batch matchers.
package services

import (
	"context"
	"log"
)

// BatchMatchEvaluator defines the common contract for AI job matching engines.
type BatchMatchEvaluator interface {
	EvaluatePendingForAllUsers(ctx context.Context)
	EvaluateForSingleUser(ctx context.Context, userID string)
	StartBackgroundScheduler(ctx context.Context)
}

// HybridBatchMatchService delegates matching requests to the NVIDIA NIM engine.
type HybridBatchMatchService struct {
	NvidiaNimService *NvidiaNimService
}

// NewHybridBatchMatchService initializes a hybrid batch matcher service with the NVIDIA NIM engine.
func NewHybridBatchMatchService(nvidiaNim *NvidiaNimService) *HybridBatchMatchService {
	return &HybridBatchMatchService{
		NvidiaNimService: nvidiaNim,
	}
}

// EvaluatePendingForAllUsers delegates batch evaluation pass to the NVIDIA NIM Engine.
func (s *HybridBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	if s.NvidiaNimService != nil {
		log.Println("[HybridMatcher] Delegating batch evaluation pass to NVIDIA NIM Engine...")
		s.NvidiaNimService.EvaluatePendingForAllUsers(ctx)
	}
}

// EvaluateForSingleUser delegates single-user evaluation pass to the NVIDIA NIM Engine.
func (s *HybridBatchMatchService) EvaluateForSingleUser(ctx context.Context, targetUserID string) {
	if s.NvidiaNimService != nil {
		log.Printf("[HybridMatcher] Delegating single-user evaluation pass for user %s to NVIDIA NIM Engine...", targetUserID)
		s.NvidiaNimService.EvaluateForSingleUser(ctx, targetUserID)
	}
}

// StartBackgroundScheduler starts background scheduler for the NVIDIA NIM Engine.
func (s *HybridBatchMatchService) StartBackgroundScheduler(ctx context.Context) {
	if s.NvidiaNimService != nil {
		s.NvidiaNimService.StartBackgroundScheduler(ctx)
	}
}
