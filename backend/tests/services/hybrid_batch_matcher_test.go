// Package services_test contains unit tests for backend services.
package services_test

import (
	"context"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

func TestHybridBatchMatchServiceFallbackToGemini(t *testing.T) {
	geminiService := services.NewGeminiBatchMatchService(nil, "test-gemini-key")
	hybridService := services.NewHybridBatchMatchService(nil, geminiService)

	if hybridService.GeminiBatchService == nil {
		t.Fatalf("expected GeminiBatchService to be initialized in HybridBatchMatchService")
	}

	hybridService.EvaluatePendingForAllUsers(context.Background())
	hybridService.EvaluateForSingleUser(context.Background(), "user-123")
}
