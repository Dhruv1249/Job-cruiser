package services_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

func init() {
	_ = godotenv.Load("../../.env")
}

func TestLiveGenAI_ListModelsAndCallAPI(t *testing.T) {
	keysStr := os.Getenv("GEMINI_API_KEYS")
	if keysStr == "" {
		keysStr = os.Getenv("GEMINI_API_KEY")
	}
	if keysStr == "" {
		t.Skip("Skipping live API test: GEMINI_API_KEYS / GEMINI_API_KEY not set")
	}

	keys := strings.Split(keysStr, ",")
	workerKey := keys[0]
	if len(keys) > 1 {
		workerKey = keys[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  workerKey,
	})
	if err != nil {
		t.Fatalf("Failed to create GenAI client: %v", err)
	}

	activeModel := "gemini-3.5-flash-lite"
	t.Logf("Calling Live API Model: %s", activeModel)

	userProfiles := []services.UserProfile{
		{
			UserID:           "usr-test-1",
			ParsedExperience: "1 year experience building Go backend microservices & REST APIs with PostgreSQL",
			TargetRoles:      "[\"Backend Engineer\", \"Software Engineer\"]",
			TargetLocations:  "[\"India (Remote)\", \"Global Remote\"]",
		},
	}

	jobSnippets := []services.JobSnippet{
		{
			JobID:       "job-test-101",
			Title:       "Junior Go Developer",
			Company:     "Acme Innovations",
			Location:    "Global Remote",
			Description: "We are seeking a Junior Go Engineer with 1+ years experience in Golang, PostgreSQL, and Docker. 100% Global Remote role.",
		},
	}

	service := &services.GeminiBatchMatchService{}
	prompt := service.BuildMultiMatchPrompt(userProfiles, jobSnippets)

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		Temperature:      genai.Ptr[float32](0.0),
	}

	res, err := client.Models.GenerateContent(ctx, activeModel, genai.Text(prompt), config)
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	rawJSON := res.Text()
	t.Logf("LIVE API RESPONSE RECEIVED:\n%s", rawJSON)

	if rawJSON == "" {
		t.Fatalf("Received empty response from live API")
	}
}

func TestLiveFull12WorkerConcurrentEngine(t *testing.T) {
	keysStr := os.Getenv("GEMINI_API_KEYS")
	if keysStr == "" {
		keysStr = os.Getenv("GEMINI_API_KEY")
	}
	if keysStr == "" {
		t.Skip("Skipping live 12-worker test: GEMINI_API_KEYS not set")
	}

	rawKeys := strings.Split(keysStr, ",")
	if len(rawKeys) < 2 {
		t.Skip("Skipping live 12-worker test: requires at least 2 keys (1 reserved + worker keys)")
	}

	reservedKey := rawKeys[0]
	workerKeys := rawKeys[1:]

	t.Logf("================================================================================")
	t.Logf("[START] LAUNCHING FULL 12-WORKER LIVE CONCURRENT ENGINE TEST")
	t.Logf("[RESERVED KEY 0]: %s... (100%% UNTOUCHED)", reservedKey[:8])
	t.Logf("[WORKER KEYS]: %d keys assigned to engine", len(workerKeys))
	t.Logf("================================================================================")

	const totalJobs = 12
	jobQueueChan := make(chan services.JobSnippet, totalJobs)

	distinctJobs := []services.JobSnippet{
		{JobID: "live-job-1", Title: "Junior Go Backend Developer", Company: "Acme Cloud", Location: "Global Remote", Description: "Go, PostgreSQL, Docker, REST APIs."},
		{JobID: "live-job-2", Title: "Junior React Engineer", Company: "Beta Tech", Location: "Remote", Description: "React, TypeScript, Next.js, Tailwind CSS."},
		{JobID: "live-job-3", Title: "Python Microservices Developer", Company: "Gamma Data", Location: "Global Remote", Description: "Python, FastAPI, Redis, Kafka."},
		{JobID: "live-job-4", Title: "Junior DevOps Engineer", Company: "Delta Ops", Location: "Global Remote", Description: "Terraform, AWS, Docker, CI/CD pipelines."},
		{JobID: "live-job-5", Title: "Associate Java Backend Engineer", Company: "Epsilon FinTech", Location: "Global Remote", Description: "Java 21, Spring Boot, MySQL, Microservices."},
		{JobID: "live-job-6", Title: "Junior Fullstack Developer (Node/React)", Company: "Zeta Interactive", Location: "Global Remote", Description: "Node.js, Express, React, PostgreSQL."},
		{JobID: "live-job-7", Title: "Junior Mobile Developer (Flutter)", Company: "Eta Apps", Location: "Global Remote", Description: "Flutter, Dart, Firebase, iOS/Android."},
		{JobID: "live-job-8", Title: "Junior Rust Developer", Company: "Theta Systems", Location: "Global Remote", Description: "Rust, Tokio, Async, WebAssembly."},
		{JobID: "live-job-9", Title: "Entry Level QA Automation Engineer", Company: "Iota Testing", Location: "Global Remote", Description: "Playwright, Cypress, JavaScript, Selenium."},
		{JobID: "live-job-10", Title: "Junior Data Engineer", Company: "Kappa Analytics", Location: "Global Remote", Description: "Python, SQL, Apache Spark, Snowflake."},
		{JobID: "live-job-11", Title: "Junior Cloud Infrastructure Engineer", Company: "Lambda Cloud", Location: "Global Remote", Description: "AWS, Kubernetes, Helm, Go."},
		{JobID: "live-job-12", Title: "Junior Machine Learning Engineer", Company: "Mu AI", Location: "Global Remote", Description: "Python, PyTorch, Transformers, HuggingFace."},
	}

	for _, j := range distinctJobs {
		jobQueueChan <- j
	}
	close(jobQueueChan)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	userProfiles := []services.UserProfile{
		{
			UserID:           "usr-candidate-1",
			ParsedExperience: "1 year experience in Go, Python, PostgreSQL, REST APIs, and Docker.",
			TargetRoles:      "[\"Backend Developer\", \"Software Engineer\"]",
			TargetLocations:  "[\"India (Remote)\", \"Global Remote\"]",
		},
	}

	var claimedMap sync.Map
	var processedCount int64
	var duplicateFound int64
	var validAIResponses int64

	testModels := []string{"gemini-3.5-flash-lite", "gemini-3.1-flash-lite"}
	var wg sync.WaitGroup

	workerID := 0
	for keyIdx, apiKey := range workerKeys {
		for modelIdx, modelName := range testModels {
			workerID++
			wg.Add(1)

			go func(id int, kIdx int, mIdx int, key string, mName string) {
				defer wg.Done()
				workerLabel := fmt.Sprintf("Worker-%d [Key-%d / Model: %s]", id, kIdx+1, mName)

				client, err := genai.NewClient(ctx, &genai.ClientConfig{
					Backend: genai.BackendGeminiAPI,
					APIKey:  key,
				})
				if err != nil {
					t.Errorf("[%s] SDK Init Failed: %v", workerLabel, err)
					return
				}

				for job := range jobQueueChan {
					previousWorker, loaded := claimedMap.LoadOrStore(job.JobID, workerLabel)
					if loaded {
						atomic.AddInt64(&duplicateFound, 1)
						t.Errorf("[CRITICAL DUP ERROR] Job %s was claimed by BOTH '%s' AND '%s'!",
							job.JobID, previousWorker, workerLabel)
						continue
					}

					atomic.AddInt64(&processedCount, 1)

					service := &services.GeminiBatchMatchService{}
					prompt := service.BuildMultiMatchPrompt(userProfiles, []services.JobSnippet{job})

					genConfig := &genai.GenerateContentConfig{
						ResponseMIMEType: "application/json",
						Temperature:      genai.Ptr[float32](0.0),
					}

					res, err := client.Models.GenerateContent(ctx, mName, genai.Text(prompt), genConfig)
					if err != nil {
						t.Logf("[%s] Live Call Failed for %s: %v", workerLabel, job.JobID, err)
						continue
					}

					rawText := res.Text()
					if rawText != "" {
						atomic.AddInt64(&validAIResponses, 1)
						t.Logf("[%s LIVE SUCCESS] Job: %s -> Response: %s", workerLabel, job.JobID, rawText)
					}
				}
			}(workerID, keyIdx, modelIdx, apiKey, modelName)
		}
	}

	wg.Wait()

	t.Logf("================================================================================")
	t.Logf("[CONCURRENCY SUMMARY RESULTS]")
	t.Logf("Total Jobs Enqueued:               %d", totalJobs)
	t.Logf("Total Jobs Claimed & Processed:    %d", processedCount)
	t.Logf("DUPLICATE CLAIMS DETECTED:         %d", duplicateFound)
	t.Logf("VALID LIVE AI RESPONSES PARSED:    %d", validAIResponses)
	t.Logf("================================================================================")

	if duplicateFound > 0 {
		t.Fatalf("FAIL: %d jobs were received by multiple concurrent threads!", duplicateFound)
	}

	if processedCount != int64(totalJobs) {
		t.Fatalf("FAIL: Expected %d jobs processed, got %d", totalJobs, processedCount)
	}
}
