package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BasicMatcherService struct {
	DB *pgxpool.Pool
}

// ComputeBasicMatch calculates a 0-100 score based on raw string/array intersections
func (s *BasicMatcherService) ComputeBasicMatch(userID string, jobID string) (*MatchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Fetch Candidate constraints
	var targetRolesRaw, workModelsRaw string
	var minSalary int
	userQuery := `
		SELECT COALESCE(p.target_roles::text, '[]'), 
		       COALESCE(p.work_models::text, '[]'), 
		       COALESCE(p.min_salary, 0)
		FROM user_preferences p
		WHERE p.user_id = $1;
	`
	err := s.DB.QueryRow(ctx, userQuery, userID).Scan(&targetRolesRaw, &workModelsRaw, &minSalary)
	if err != nil {
		return nil, fmt.Errorf("failed fetching basic user preferences: %v", err)
	}

	// 2. Fetch Job requirements
	var title, rawDesc, tagsRaw, jobType string
	var salaryMax int
	jobQuery := `
		SELECT COALESCE(title, ''), 
		       COALESCE(raw_desc, ''), 
		       COALESCE(tags::text, '[]'), 
		       COALESCE(job_type, ''), 
		       COALESCE(salary_max, 0)
		FROM jobs
		WHERE id = $1;
	`
	err = s.DB.QueryRow(ctx, jobQuery, jobID).Scan(&title, &rawDesc, &tagsRaw, &jobType, &salaryMax)
	if err != nil {
		return nil, fmt.Errorf("failed fetching basic job metrics: %v", err)
	}

	// 3. Parse JSON arrays
	var targetRoles, workModels, jobTags []string
	json.Unmarshal([]byte(targetRolesRaw), &targetRoles)
	json.Unmarshal([]byte(workModelsRaw), &workModels)
	json.Unmarshal([]byte(tagsRaw), &jobTags)

	score := 0
	var reasons []string

	// Logic 1: Title match matching target roles (40 Points Max)
	titleLower := strings.ToLower(title)
	roleMatched := false
	for _, role := range targetRoles {
		if strings.Contains(titleLower, strings.ToLower(role)) {
			score += 40
			reasons = append(reasons, fmt.Sprintf("Job title matches your target role: %s", role))
			roleMatched = true
			break
		}
	}
	if !roleMatched {
		reasons = append(reasons, "Job title does not directly match your specified target roles")
	}

	// Logic 2: Salary verification (30 Points Max)
	if salaryMax >= minSalary && minSalary > 0 {
		score += 30
		reasons = append(reasons, "Job maximum compensation range meets your salary floor")
	} else if minSalary > 0 && salaryMax < minSalary && salaryMax > 0 {
		reasons = append(reasons, "Job listed maximum compensation falls below your salary floor")
	}

	// Logic 3: Tech stack overlap (30 Points Max)
	descLower := strings.ToLower(rawDesc)
	matchedTagsCount := 0
	for _, tag := range jobTags {
		if strings.Contains(descLower, strings.ToLower(tag)) {
			matchedTagsCount++
		}
	}
	if len(jobTags) > 0 && matchedTagsCount > 0 {
		tagScore := (matchedTagsCount * 30) / len(jobTags)
		if tagScore > 30 {
			tagScore = 30
		}
		score += tagScore
		reasons = append(reasons, fmt.Sprintf("Matched %d keywords within your preferred technical stack", matchedTagsCount))
	}

	// Determine generic suggested action based on scores
	suggestedAction := "review"
	if score >= 70 {
		suggestedAction = "apply"
	} else if score < 40 {
		suggestedAction = "skip"
	}

	return &MatchResponse{
		MatchScore:      score,
		MatchReasons:    reasons,
		SuggestedAction: suggestedAction,
	}, nil
}
