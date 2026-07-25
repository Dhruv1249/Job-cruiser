package services

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// BatchMatchEvaluator is the common interface for batch AI job matching engines.
type BatchMatchEvaluator interface {
	EvaluatePendingForAllUsers(ctx context.Context)
	EvaluateForSingleUser(ctx context.Context, userID string)
	StartBackgroundScheduler(ctx context.Context)
}

// HybridBatchMatchService delegates matching between Mistral and Gemini engines
// based on PRIMARY_AI_PROVIDER environment settings and quota availability.
type HybridBatchMatchService struct {
	MistralService *MistralBatchMatchService
	GeminiService  *GeminiBatchMatchService
}

func NewHybridBatchMatchService(mistral *MistralBatchMatchService, gemini *GeminiBatchMatchService) *HybridBatchMatchService {
	return &HybridBatchMatchService{
		MistralService: mistral,
		GeminiService:  gemini,
	}
}

func (s *HybridBatchMatchService) EvaluatePendingForAllUsers(ctx context.Context) {
	provider := strings.ToLower(os.Getenv("PRIMARY_AI_PROVIDER"))

	if provider == "gemini" || s.MistralService == nil || len(s.MistralService.WorkerKeys) == 0 {
		log.Println("[HybridMatcher] Delegating batch evaluation pass to Gemini Engine...")
		s.GeminiService.EvaluatePendingForAllUsers(ctx)
		return
	}

	if provider == "mistral" || s.GeminiService == nil || len(s.GeminiService.WorkerKeys) == 0 {
		log.Println("[HybridMatcher] Delegating batch evaluation pass to Mistral Engine...")
		s.MistralService.EvaluatePendingForAllUsers(ctx)
		return
	}

	log.Println("[HybridMatcher] Launching Parallel Dual-Engine Matching: Running Gemini Engine and Mistral Engine concurrently!")
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.GeminiService.EvaluatePendingForAllUsers(ctx)
	}()

	go func() {
		defer wg.Done()
		s.MistralService.EvaluatePendingForAllUsers(ctx)
	}()

	wg.Wait()
}

func (s *HybridBatchMatchService) EvaluateForSingleUser(ctx context.Context, userID string) {
	provider := strings.ToLower(os.Getenv("PRIMARY_AI_PROVIDER"))

	if provider == "gemini" || s.MistralService == nil || len(s.MistralService.WorkerKeys) == 0 {
		log.Println("[HybridMatcher] Delegating single-user evaluation pass to Gemini Engine...")
		s.GeminiService.EvaluateForSingleUser(ctx, userID)
		return
	}

	if provider == "mistral" || s.GeminiService == nil || len(s.GeminiService.WorkerKeys) == 0 {
		log.Println("[HybridMatcher] Delegating single-user evaluation pass to Mistral Engine...")
		s.MistralService.EvaluateForSingleUser(ctx, userID)
		return
	}

	log.Printf("[HybridMatcher] Launching Parallel Dual-Engine Matching for user %s: Running Gemini and Mistral concurrently!", userID)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		s.GeminiService.EvaluateForSingleUser(ctx, userID)
	}()

	go func() {
		defer wg.Done()
		s.MistralService.EvaluateForSingleUser(ctx, userID)
	}()

	wg.Wait()
}

func (s *HybridBatchMatchService) StartBackgroundScheduler(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	log.Println("[HybridMatcher] Starting 5-minute background AI matcher scheduler...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				log.Println("[HybridMatcher] 5-minute ticker tick: Triggering background AI evaluation check...")
				s.EvaluatePendingForAllUsers(ctx)
			}
		}
	}()
}
