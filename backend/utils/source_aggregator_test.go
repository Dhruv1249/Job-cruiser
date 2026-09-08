package utils_test

import (
	"encoding/json"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

func TestAggregateSourceStatisticsEmptyInput(testingContext *testing.T) {
	emptyResult, emptyError := utils.AggregateSourceStatistics(nil)
	if emptyError != nil {
		testingContext.Fatalf("unexpected error on nil input: %v", emptyError)
	}
	if string(emptyResult) != "{}" {
		testingContext.Errorf("expected empty json object, got %s", string(emptyResult))
	}

	nullResult, nullError := utils.AggregateSourceStatistics([]byte("null"))
	if nullError != nil {
		testingContext.Fatalf("unexpected error on null input: %v", nullError)
	}
	if string(nullResult) != "{}" {
		testingContext.Errorf("expected empty json object, got %s", string(nullResult))
	}
}

func TestAggregateSourceStatisticsGranularCompositeKeys(testingContext *testing.T) {
	inputPayload := map[string]interface{}{
		"linkedin:sde i:india": map[string]interface{}{
			"jobs_found":       200,
			"duration_seconds": 15.5,
			"error":            nil,
		},
		"linkedin:c developer:india": map[string]interface{}{
			"jobs_found":       150,
			"duration_seconds": 12.0,
			"error":            nil,
		},
		"indeed:build engineer:remote": map[string]interface{}{
			"jobs_found":       50,
			"duration_seconds": 5.2,
			"error":            nil,
		},
		"themuse:software engineer:remote": map[string]interface{}{
			"jobs_found":       20,
			"duration_seconds": 3.1,
			"error":            "timeout warning",
		},
	}

	inputBytes, marshalError := json.Marshal(inputPayload)
	if marshalError != nil {
		testingContext.Fatalf("failed to marshal input payload: %v", marshalError)
	}

	aggregatedBytes, aggregationError := utils.AggregateSourceStatistics(inputBytes)
	if aggregationError != nil {
		testingContext.Fatalf("unexpected aggregation error: %v", aggregationError)
	}

	var parsedOutput map[string]utils.ScraperPlatformMetric
	if unmarshalError := json.Unmarshal(aggregatedBytes, &parsedOutput); unmarshalError != nil {
		testingContext.Fatalf("failed to unmarshal aggregated output: %v", unmarshalError)
	}

	linkedinMetric, linkedinExists := parsedOutput["linkedin"]
	if !linkedinExists {
		testingContext.Fatalf("expected linkedin platform in output")
	}
	if linkedinMetric.JobsFound != 350 {
		testingContext.Errorf("expected 350 jobs found for linkedin, got %d", linkedinMetric.JobsFound)
	}
	if linkedinMetric.QueryCount != 2 {
		testingContext.Errorf("expected query count 2 for linkedin, got %d", linkedinMetric.QueryCount)
	}
	if linkedinMetric.DurationSeconds != 27.5 {
		testingContext.Errorf("expected duration 27.5 for linkedin, got %f", linkedinMetric.DurationSeconds)
	}

	indeedMetric, indeedExists := parsedOutput["indeed"]
	if !indeedExists {
		testingContext.Fatalf("expected indeed platform in output")
	}
	if indeedMetric.JobsFound != 50 {
		testingContext.Errorf("expected 50 jobs found for indeed, got %d", indeedMetric.JobsFound)
	}
	if indeedMetric.QueryCount != 1 {
		testingContext.Errorf("expected query count 1 for indeed, got %d", indeedMetric.QueryCount)
	}

	themuseMetric, themuseExists := parsedOutput["themuse"]
	if !themuseExists {
		testingContext.Fatalf("expected themuse platform in output")
	}
	if themuseMetric.Error == nil || *themuseMetric.Error != "timeout warning" {
		testingContext.Errorf("expected error 'timeout warning' for themuse")
	}
}

func TestAggregateSourceStatisticsFlatCountsMap(testingContext *testing.T) {
	inputPayload := map[string]interface{}{
		"greenhouse": 45,
		"ashby":      12,
		"lever":      30,
	}

	inputBytes, marshalError := json.Marshal(inputPayload)
	if marshalError != nil {
		testingContext.Fatalf("failed to marshal flat input: %v", marshalError)
	}

	aggregatedBytes, aggregationError := utils.AggregateSourceStatistics(inputBytes)
	if aggregationError != nil {
		testingContext.Fatalf("unexpected aggregation error: %v", aggregationError)
	}

	var parsedOutput map[string]utils.ScraperPlatformMetric
	if unmarshalError := json.Unmarshal(aggregatedBytes, &parsedOutput); unmarshalError != nil {
		testingContext.Fatalf("failed to unmarshal aggregated output: %v", unmarshalError)
	}

	if parsedOutput["greenhouse"].JobsFound != 45 {
		testingContext.Errorf("expected 45 jobs for greenhouse, got %d", parsedOutput["greenhouse"].JobsFound)
	}
	if parsedOutput["greenhouse"].QueryCount != 1 {
		testingContext.Errorf("expected query count 1 for greenhouse, got %d", parsedOutput["greenhouse"].QueryCount)
	}
	if parsedOutput["ashby"].JobsFound != 12 {
		testingContext.Errorf("expected 12 jobs for ashby, got %d", parsedOutput["ashby"].JobsFound)
	}
}

func TestAggregateSourceStatisticsStringArray(testingContext *testing.T) {
	inputBytes := []byte(`["greenhouse", "lever", "ashby"]`)

	aggregatedBytes, aggregationError := utils.AggregateSourceStatistics(inputBytes)
	if aggregationError != nil {
		testingContext.Fatalf("unexpected aggregation error on array: %v", aggregationError)
	}

	var parsedOutput map[string]utils.ScraperPlatformMetric
	if unmarshalError := json.Unmarshal(aggregatedBytes, &parsedOutput); unmarshalError != nil {
		testingContext.Fatalf("failed to unmarshal array output: %v", unmarshalError)
	}

	if len(parsedOutput) != 3 {
		testingContext.Errorf("expected 3 platform entries, got %d", len(parsedOutput))
	}
	if parsedOutput["greenhouse"].QueryCount != 1 {
		testingContext.Errorf("expected query count 1 for greenhouse, got %d", parsedOutput["greenhouse"].QueryCount)
	}
}
