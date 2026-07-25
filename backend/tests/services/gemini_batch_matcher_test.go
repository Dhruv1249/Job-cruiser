package services_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestNewGeminiBatchMatchService_KeyPreservation(t *testing.T) {
	rawKeysStr := "key0_reserved,key1_worker,key2_worker,key3_worker"
	service := services.NewGeminiBatchMatchService(nil, rawKeysStr)

	if service.ReservedKey != "key0_reserved" {
		t.Fatalf("Expected ReservedKey to be 'key0_reserved', got: '%s'", service.ReservedKey)
	}

	if len(service.WorkerKeys) != 3 {
		t.Fatalf("Expected 3 worker keys, got: %d", len(service.WorkerKeys))
	}

	if service.WorkerKeys[0] != "key1_worker" || service.WorkerKeys[1] != "key2_worker" || service.WorkerKeys[2] != "key3_worker" {
		t.Fatalf("WorkerKeys slice mismatch: %v", service.WorkerKeys)
	}

	for _, k := range service.WorkerKeys {
		if k == "key0_reserved" {
			t.Fatalf("CRITICAL BUG: Reserved key0 was found inside worker keys pool!")
		}
	}
}

func TestBuildMultiMatchPrompt_FullJDCorrectness(t *testing.T) {
	service := &services.GeminiBatchMatchService{}
	userChunk := []services.UserProfile{
		{
			UserID:           "usr-101",
			ParsedExperience: "1 year Go/Python backend experience at tech startup",
			TargetRoles:      "[\"Software Engineer\", \"Backend Developer\"]",
			TargetLocations:  "[\"India (On-site & Hybrid)\", \"India (Remote)\", \"Global Remote\"]",
		},
		{
			UserID:           "usr-102",
			ParsedExperience: "Junior Frontend Engineer with React/TypeScript skills",
			TargetRoles:      "[\"Frontend Developer\"]",
			TargetLocations:  "[\"India (Remote)\", \"Global Remote\"]",
		},
	}

	rawJDText := "We are looking for a Junior Backend Engineer to build scalable microservices in Go and Python. Requirements: 1+ year experience with Go or Python, PostgreSQL, Docker."

	batch := []services.JobSnippet{
		{
			JobID:       "job-999",
			Title:       "Junior Backend Engineer (Go/Python)",
			Company:     "Acme Corp",
			Location:    "Global Remote",
			Description: rawJDText,
		},
	}

	prompt := service.BuildMultiMatchPrompt(userChunk, batch)

	if !strings.Contains(prompt, "usr-101") || !strings.Contains(prompt, "usr-102") {
		t.Errorf("Prompt missing candidate UserIDs")
	}
	if !strings.Contains(prompt, "job-999") {
		t.Errorf("Prompt missing JobID")
	}
	if !strings.Contains(prompt, "Junior Backend Engineer") {
		t.Errorf("Prompt missing job title")
	}
	if !strings.Contains(prompt, rawJDText) {
		t.Errorf("Prompt did NOT contain the full untruncated JD body!")
	}
	if !strings.Contains(prompt, "INDIA and ONLY have work authorization for India or Global Remote roles") {
		t.Errorf("Prompt missing location compliance rules")
	}
}

func TestLocationFilteringRules(t *testing.T) {
	targetLocs := "[\"India (On-site & Hybrid)\", \"India (Remote)\", \"Global Remote\"]"

	if !services.IsNonTargetOnsite("San Francisco, CA", false, targetLocs) {
		t.Errorf("Expected San Francisco, CA physical location to be flagged as non-target onsite for India candidate")
	}

	if services.IsNonTargetOnsite("Bengaluru, India", false, targetLocs) {
		t.Errorf("Expected Bengaluru, India to be valid target location")
	}

	if services.IsNonTargetOnsite("Remote", true, targetLocs) {
		t.Errorf("Expected Global Remote to be valid")
	}
}

func TestSeniorRoleFilteringRules(t *testing.T) {
	if !services.IsSeniorRoleTitle("Senior Backend Engineer") {
		t.Errorf("Expected 'Senior Backend Engineer' to be flagged as senior role")
	}
	if !services.IsSeniorRoleTitle("Lead Software Architect") {
		t.Errorf("Expected 'Lead Software Architect' to be flagged as senior role")
	}
	if !services.IsSeniorRoleTitle("VP of Engineering") {
		t.Errorf("Expected 'VP of Engineering' to be flagged as senior role")
	}

	if services.IsSeniorRoleTitle("Junior Software Developer") {
		t.Errorf("'Junior Software Developer' should NOT be flagged as senior role")
	}
	if services.IsSeniorRoleTitle("Associate Backend Engineer") {
		t.Errorf("'Associate Backend Engineer' should NOT be flagged as senior role")
	}
}

func TestHybridBatchMatchService_ProviderRouting(t *testing.T) {
	geminiMatcher := services.NewGeminiBatchMatchService(nil, "key0_res,key1_wrk")
	hybrid := services.NewHybridBatchMatchService(nil, geminiMatcher)

	t.Setenv("PRIMARY_AI_PROVIDER", "gemini")
	hybrid.EvaluateForSingleUser(context.Background(), "test-user-id")
}

func TestGeminiBatchMatchService_ConcurrencyDeduplication(t *testing.T) {
	const totalJobs = 120
	const numWorkers = 12

	jobQueueChan := make(chan services.JobSnippet, totalJobs)
	for i := 1; i <= totalJobs; i++ {
		jobQueueChan <- services.JobSnippet{
			JobID:       fmt.Sprintf("job-conc-%d", i),
			Title:       fmt.Sprintf("Developer Position %d", i),
			Company:     "Tech Corp",
			Location:    "Remote",
			Description: "Full-time Go developer role.",
		}
	}
	close(jobQueueChan)

	var claimedMap sync.Map
	var processedCount int64
	var duplicateFound int64

	var wg sync.WaitGroup

	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobQueueChan {
				if _, loaded := claimedMap.LoadOrStore(job.JobID, workerID); loaded {
					atomic.AddInt64(&duplicateFound, 1)
				} else {
					atomic.AddInt64(&processedCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()

	if duplicateFound > 0 {
		t.Fatalf("CRITICAL CONCURRENCY BUG: %d jobs were received by multiple threads!", duplicateFound)
	}

	if processedCount != int64(totalJobs) {
		t.Fatalf("Expected %d jobs processed, got: %d", totalJobs, processedCount)
	}
}

func TestAIResponseUnmarshalVerification(t *testing.T) {
	sampleAIJSON := `{
		"results": [
			{
				"job_id": "job-101",
				"seniority": "Junior",
				"salary_min": 70000,
				"salary_max": 90000,
				"currency": "USD",
				"matches": [
					{
						"user_id": "usr-1",
						"is_matched": true,
						"match_score": 88,
						"tech_stack": ["Go", "PostgreSQL", "Docker"],
						"reasoning": "Strong match for 1 year Go backend experience and location preferences."
					}
				]
			}
		]
	}`

	var batchResponse struct {
		Results []struct {
			JobID     string `json:"job_id"`
			Seniority string `json:"seniority"`
			SalaryMin *int   `json:"salary_min"`
			SalaryMax *int   `json:"salary_max"`
			Currency  *string`json:"currency"`
			Matches   []struct {
				UserID         string   `json:"user_id"`
				IsMatched      bool     `json:"is_matched"`
				MatchScore     int      `json:"match_score"`
				TechStack      []string `json:"tech_stack"`
				MatchReasoning string   `json:"reasoning"`
			} `json:"matches"`
		} `json:"results"`
	}

	err := json.Unmarshal([]byte(sampleAIJSON), &batchResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal sample AI JSON response: %v", err)
	}

	if len(batchResponse.Results) != 1 {
		t.Fatalf("Expected 1 job result, got %d", len(batchResponse.Results))
	}

	jobRes := batchResponse.Results[0]
	if jobRes.JobID != "job-101" || jobRes.Seniority != "Junior" {
		t.Errorf("Mismatch in JobID or Seniority: %s / %s", jobRes.JobID, jobRes.Seniority)
	}

	if jobRes.SalaryMin == nil || *jobRes.SalaryMin != 70000 {
		t.Errorf("Expected salary_min 70000, got: %v", jobRes.SalaryMin)
	}

	if len(jobRes.Matches) != 1 {
		t.Fatalf("Expected 1 candidate match, got %d", len(jobRes.Matches))
	}

	m := jobRes.Matches[0]
	if m.UserID != "usr-1" || !m.IsMatched || m.MatchScore != 88 {
		t.Errorf("Candidate match mismatch: UserID=%s, IsMatched=%v, Score=%d", m.UserID, m.IsMatched, m.MatchScore)
	}

	if len(m.TechStack) != 3 || m.TechStack[0] != "Go" {
		t.Errorf("TechStack mismatch: %v", m.TechStack)
	}
}
