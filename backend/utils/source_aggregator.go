package utils

import (
	"encoding/json"
	"math"
	"strings"
)

type ScraperPlatformMetric struct {
	JobsFound       int     `json:"jobs_found"`
	JobsAdded       int     `json:"jobs_added"`
	DurationSeconds float64 `json:"duration_seconds"`
	QueryCount      int     `json:"query_count"`
	Error           *string `json:"error,omitempty"`
}

func AggregateSourceStatistics(rawSourcesJSON []byte) ([]byte, error) {
	if len(rawSourcesJSON) == 0 {
		return []byte("{}"), nil
	}

	trimmedInput := strings.TrimSpace(string(rawSourcesJSON))
	if trimmedInput == "" || trimmedInput == "null" {
		return []byte("{}"), nil
	}

	var mapPayload map[string]interface{}
	if unmarshalError := json.Unmarshal([]byte(trimmedInput), &mapPayload); unmarshalError == nil {
		aggregatedMetrics := make(map[string]*ScraperPlatformMetric)

		for rawKey, rawValue := range mapPayload {
			trimmedKey := strings.TrimSpace(rawKey)
			if trimmedKey == "" {
				continue
			}

			keyTokens := strings.Split(trimmedKey, ":")
			platformIdentifier := strings.ToLower(strings.TrimSpace(keyTokens[0]))
			if platformIdentifier == "" {
				continue
			}

			platformMetric, metricExists := aggregatedMetrics[platformIdentifier]
			if !metricExists {
				platformMetric = &ScraperPlatformMetric{}
				aggregatedMetrics[platformIdentifier] = platformMetric
			}
			platformMetric.QueryCount++

			if numericCount, isNumber := rawValue.(float64); isNumber {
				platformMetric.JobsFound += int(numericCount)
			} else if detailMap, isMap := rawValue.(map[string]interface{}); isMap {
				if countValue, hasCount := detailMap["jobs_found"]; hasCount && countValue != nil {
					if numericCount, isNumber := countValue.(float64); isNumber {
						platformMetric.JobsFound += int(numericCount)
					}
				} else if countValue, hasCount := detailMap["count"]; hasCount && countValue != nil {
					if numericCount, isNumber := countValue.(float64); isNumber {
						platformMetric.JobsFound += int(numericCount)
					}
				}

				if addedValue, hasAdded := detailMap["jobs_added"]; hasAdded && addedValue != nil {
					if numericAdded, isNumber := addedValue.(float64); isNumber {
						platformMetric.JobsAdded += int(numericAdded)
					}
				}

				if durationValue, hasDuration := detailMap["duration_seconds"]; hasDuration && durationValue != nil {
					if numericDuration, isNumber := durationValue.(float64); isNumber {
						platformMetric.DurationSeconds += numericDuration
					}
				}

				if errorValue, hasError := detailMap["error"]; hasError && errorValue != nil {
					if errorString, isString := errorValue.(string); isString && strings.TrimSpace(errorString) != "" {
						trimmedError := strings.TrimSpace(errorString)
						platformMetric.Error = &trimmedError
					}
				}
			}
		}

		for _, metric := range aggregatedMetrics {
			metric.DurationSeconds = math.Round(metric.DurationSeconds*100) / 100
		}

		return json.Marshal(aggregatedMetrics)
	}

	var arrayPayload []interface{}
	if unmarshalArrayError := json.Unmarshal([]byte(trimmedInput), &arrayPayload); unmarshalArrayError == nil {
		aggregatedMetrics := make(map[string]*ScraperPlatformMetric)

		for _, rawItem := range arrayPayload {
			if stringItem, isString := rawItem.(string); isString {
				platformIdentifier := strings.ToLower(strings.TrimSpace(stringItem))
				if platformIdentifier == "" {
					continue
				}

				platformMetric, metricExists := aggregatedMetrics[platformIdentifier]
				if !metricExists {
					platformMetric = &ScraperPlatformMetric{}
					aggregatedMetrics[platformIdentifier] = platformMetric
				}
				platformMetric.QueryCount++
			}
		}

		return json.Marshal(aggregatedMetrics)
	}

	return []byte("{}"), nil
}
