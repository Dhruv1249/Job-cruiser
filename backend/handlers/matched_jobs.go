package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func unmarshalStringJSON(raw string, target interface{}) error {
	return json.Unmarshal([]byte(raw), target)
}

// MatchedJobsHandler serves the matched jobs endpoint, reading from user_job_matches
// joined with the jobs table, ordered by match_score descending.
type MatchedJobsHandler struct {
	DB *pgxpool.Pool
}

// MatchedJobResponse is the response shape for a single matched job.
type MatchedJobResponse struct {
	JobID          string   `json:"job_id"`
	Title          string   `json:"title"`
	Company        string   `json:"company"`
	Location       string   `json:"location"`
	IsRemote       bool     `json:"is_remote"`
	Source         string   `json:"source"`
	URL            string   `json:"url"`
	PostedDate     string   `json:"posted_date"`
	Seniority      string   `json:"seniority"`
	Summary        string   `json:"summary"`
	MatchScore     int      `json:"match_score"`
	MatchReasoning string   `json:"match_reasoning"`
	TechStack      []string `json:"tech_stack"`
	IsMatched      bool     `json:"is_matched"`
	SalaryMin      *int     `json:"salary_min"`
	SalaryMax      *int     `json:"salary_max"`
	Currency       string   `json:"currency"`
}

// GetMatchedJobs returns jobs evaluated for the authenticated user, sorted by
// match_score descending. Accepts optional query param min_score (default 0)
// to filter to only jobs above a confidence threshold.
func (h *MatchedJobsHandler) GetMatchedJobs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	minScoreParam := c.DefaultQuery("min_score", "0")
	minScore, err := strconv.Atoi(minScoreParam)
	if err != nil || minScore < 0 || minScore > 100 {
		minScore = 0
	}

	limitParam := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}

	offsetParam := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetParam)
	if err != nil || offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			j.id,
			j.title,
			COALESCE(comp.name, ''),
			COALESCE(j.location, ''),
			j.is_remote,
			COALESCE(j.source, ''),
			j.url,
			COALESCE(j.posted_date, ''),
			COALESCE(j.seniority, ''),
			COALESCE(j.summary, ''),
			ujm.match_score,
			COALESCE(ujm.match_reasoning, ''),
			COALESCE(ujm.tech_stack::text, '[]'),
			ujm.is_ai_matched,
			j.salary_min,
			j.salary_max,
			COALESCE(j.currency, 'USD')
		FROM user_job_matches ujm
		JOIN jobs j ON ujm.job_id = j.id
		LEFT JOIN companies comp ON j.company_id = comp.id
		WHERE ujm.user_id = $1
		  AND ujm.match_score >= $2
		ORDER BY ujm.match_score DESC
		LIMIT $3 OFFSET $4;
	`

	rows, err := h.DB.Query(context.Background(), query, userID, minScore, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query matched jobs"})
		return
	}
	defer rows.Close()

	var matchedJobs []MatchedJobResponse
	for rows.Next() {
		var job MatchedJobResponse
		var techStackRaw string
		if err := rows.Scan(
			&job.JobID,
			&job.Title,
			&job.Company,
			&job.Location,
			&job.IsRemote,
			&job.Source,
			&job.URL,
			&job.PostedDate,
			&job.Seniority,
			&job.Summary,
			&job.MatchScore,
			&job.MatchReasoning,
			&techStackRaw,
			&job.IsMatched,
			&job.SalaryMin,
			&job.SalaryMax,
			&job.Currency,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan matched job row"})
			return
		}

		if err := unmarshalStringJSON(techStackRaw, &job.TechStack); err != nil {
			job.TechStack = []string{}
		}

		matchedJobs = append(matchedJobs, job)
	}

	if matchedJobs == nil {
		matchedJobs = []MatchedJobResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  matchedJobs,
		"count": len(matchedJobs),
	})
}
