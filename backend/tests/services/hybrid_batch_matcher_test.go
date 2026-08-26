// Package services_test contains unit tests for backend services.
package services_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestHybridBatchMatchServiceFallbackToGemini(t *testing.T) {
	t.Setenv("PRIMARY_AI_PROVIDER", "")

	geminiService := services.NewGeminiBatchMatchService(nil, "test-gemini-key")
	hybridService := services.NewHybridBatchMatchService(nil, geminiService)

	if hybridService.GeminiBatchService == nil {
		t.Fatalf("expected GeminiBatchService to be initialized in HybridBatchMatchService")
	}

	hybridService.EvaluatePendingForAllUsers(context.Background())
	hybridService.EvaluateForSingleUser(context.Background(), "user-123")
}

func TestHybridBatchMatchServiceAutomaticPilotProbingFallback(t *testing.T) {
	t.Setenv("PRIMARY_AI_PROVIDER", "nvidia_nim")

	nvidiaService := services.NewNvidiaNimService(nil, "test-nvidia-key")
	geminiService := services.NewGeminiBatchMatchService(nil, "test-gemini-key")
	hybridService := services.NewHybridBatchMatchService(nvidiaService, geminiService)

	hybridService.EvaluatePendingForAllUsers(context.Background())
	hybridService.EvaluateForSingleUser(context.Background(), "user-123")
}

func TestHybridBatchMatchServicePrimaryProviderGemini(t *testing.T) {
	t.Setenv("PRIMARY_AI_PROVIDER", "gemini")

	nvidiaService := services.NewNvidiaNimService(nil, "test-nvidia-key")
	geminiService := services.NewGeminiBatchMatchService(nil, "test-gemini-key")
	hybridService := services.NewHybridBatchMatchService(nvidiaService, geminiService)

	hybridService.EvaluatePendingForAllUsers(context.Background())
	hybridService.EvaluateForSingleUser(context.Background(), "user-123")
}

func TestNvidiaNimServiceSharedCircuitBreakerFourErrors(t *testing.T) {
	var requestCount atomic.Int64

	mockServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		count := requestCount.Add(1)
		writer.Header().Set("Content-Type", "text/event-stream")
		if count == 1 {
			_, _ = writer.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"results\\\":[{\\\"job_id\\\":\\\"00000000-0000-0000-0000-000000000001\\\",\\\"user_id\\\":\\\"11111111-1111-1111-1111-111111111111\\\",\\\"match_score\\\":85,\\\"match_reasoning\\\":\\\"Good fit\\\",\\\"inferred_required_yoe\\\":2,\\\"is_matched\\\":true}]}\"}}]}\n\ndata: [DONE]\n\n"))
			return
		}
		_, _ = writer.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"invalid non-json text\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer mockServer.Close()

	nvidiaService := services.NewNvidiaNimService(nil, "test-api-key")
	nvidiaService.Endpoint = mockServer.URL

	userProfiles := []services.UserProfileData{
		{UserID: "11111111-1111-1111-1111-111111111111", Email: "test@example.com"},
	}

	firstBatch := []services.JobSnippetData{
		{JobID: "00000000-0000-0000-0000-000000000001", Title: "Dev", Company: "Co", Location: "Remote", Description: "Go"},
	}
	nvidiaService.FillTokenBucketForTest()
	success1 := nvidiaService.EvaluatePilotBatch(context.Background(), userProfiles, firstBatch)
	if !success1 {
		t.Fatalf("expected first pilot batch to succeed")
	}

	failBatch := []services.JobSnippetData{
		{JobID: "00000000-0000-0000-0000-000000000002", Title: "Dev2", Company: "Co2", Location: "Remote", Description: "Go"},
	}

	for i := 0; i < 4; i++ {
		nvidiaService.FillTokenBucketForTest()
		nvidiaService.EvaluatePilotBatch(context.Background(), userProfiles, failBatch)
	}

	singlePassResult := nvidiaService.EvaluateForSingleUserWithResult(context.Background(), "11111111-1111-1111-1111-111111111111")
	if singlePassResult != true {
		t.Fatalf("expected result when DB is nil to be true")
	}
}

func TestHybridBatchMatchServicePipelineShutdownDisablesNim(t *testing.T) {
	t.Setenv("GEMINI_BATCH_MODELS", "test-model-1")

	nvidiaService := services.NewNvidiaNimService(nil, "test-nvidia-key")
	geminiService := services.NewGeminiBatchMatchService(nil, "test-gemini-key")
	hybridService := services.NewHybridBatchMatchService(nvidiaService, geminiService)

	// Simulate 5 errors on gemini model to trigger pipeline shutdown
	for i := 0; i < 5; i++ {
		geminiService.RecordModelFailureForTest(context.Background(), "test-model-1", "mock quota error")
	}

	if !geminiService.IsPipelinePermanentlyStopped() {
		t.Fatalf("expected gemini pipeline to be permanently stopped")
	}

	if !nvidiaService.IsPipelinePermanentlyStopped() {
		t.Fatalf("expected nvidia NIM engine to also be permanently stopped via shutdown hook")
	}

	// Invocations to hybrid matcher should safely return without running
	hybridService.EvaluatePendingForAllUsers(context.Background())
	hybridService.EvaluateForSingleUser(context.Background(), "user-123")
}
