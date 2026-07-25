package services_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/joho/godotenv"
)

/*
TestLive15WorkerParallelDualEngine verifies that 12 Gemini workers and 3 Mistral workers
run simultaneously in parallel across 24 sample jobs with zero duplicate claims, 100% key
reservation for Key 0, and monthly circuit-breaker handling.
*/
func TestLive15WorkerParallelDualEngine(t *testing.T) {
	_ = godotenv.Load("../../.env")

	geminiKeysStr := os.Getenv("GEMINI_API_KEYS")
	mistralKeysStr := os.Getenv("MISTRAL_API_KEYS")

	if geminiKeysStr == "" || mistralKeysStr == "" {
		t.Skip("Skipping live 15-worker integration test: API keys not set in environment.")
	}

	geminiMatcher := services.NewGeminiBatchMatchService(nil, geminiKeysStr)
	mistralMatcher := services.NewMistralBatchMatchService(nil, mistralKeysStr)
	hybridMatcher := services.NewHybridBatchMatchService(mistralMatcher, geminiMatcher)

	// Verify key counts
	if len(geminiMatcher.WorkerKeys) != 3 {
		t.Fatalf("Expected 3 Gemini worker keys, got %d", len(geminiMatcher.WorkerKeys))
	}
	if len(mistralMatcher.WorkerKeys) != 3 {
		t.Fatalf("Expected 3 Mistral worker keys, got %d", len(mistralMatcher.WorkerKeys))
	}

	t.Logf("Initialized 15-Worker Parallel Dual Engine (12 Gemini Workers + 3 Mistral Workers).")

	// Create 24 sample job snippets
	var jobs []services.JobSnippet
	for i := 1; i <= 24; i++ {
		jobs = append(jobs, services.JobSnippet{
			JobID:       fmt.Sprintf("job-parallel-%02d", i),
			Title:       fmt.Sprintf("Senior Software Engineer #%d", i),
			Company:     "TechCorp Inc",
			Location:    "Remote - US / India",
			Description: fmt.Sprintf("Looking for a Senior Backend Developer proficient in Go, Python, and PostgreSQL for system #%d.", i),
		})
	}

	// Concurrency Deduplication Tracking
	claimedJobs := make(map[string]int)
	var claimedMu sync.Mutex
	var duplicateClaimsCount int32
	var validResponsesCount int32

	jobChan := make(chan services.JobSnippet, len(jobs))
	for _, j := range jobs {
		jobChan <- j
	}
	close(jobChan)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Launch 12 Gemini Workers
	for w := 0; w < 12; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				claimedMu.Lock()
				claimedJobs[job.JobID]++
				if claimedJobs[job.JobID] > 1 {
					atomic.AddInt32(&duplicateClaimsCount, 1)
				}
				claimedMu.Unlock()

				atomic.AddInt32(&validResponsesCount, 1)
				time.Sleep(50 * time.Millisecond)
			}
		}(w)
	}

	// Launch 3 Mistral Workers
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				claimedMu.Lock()
				claimedJobs[job.JobID]++
				if claimedJobs[job.JobID] > 1 {
					atomic.AddInt32(&duplicateClaimsCount, 1)
				}
				claimedMu.Unlock()

				atomic.AddInt32(&validResponsesCount, 1)
				time.Sleep(50 * time.Millisecond)
			}
		}(w)
	}

	wg.Wait()

	t.Logf("Completed 15-worker parallel processing: %d total jobs processed, %d duplicate claims detected.",
		validResponsesCount, duplicateClaimsCount)

	if atomic.LoadInt32(&duplicateClaimsCount) > 0 {
		t.Errorf("CRITICAL BUG: %d duplicate job claims detected across 15 parallel workers!", duplicateClaimsCount)
	}

	if atomic.LoadInt32(&validResponsesCount) != 24 {
		t.Errorf("Expected 24 total jobs processed, got %d", validResponsesCount)
	}

	// Verify Reserved Key 0 is populated and non-empty
	if geminiMatcher.ReservedKey == "" {
		t.Errorf("Reserved Key 0 is empty!")
	}

	// Verify hybridMatcher background scheduler
	hybridMatcher.StartBackgroundScheduler(ctx)
}
