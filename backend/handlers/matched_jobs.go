package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
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
	JobID             string   `json:"job_id"`
	Title             string   `json:"title"`
	Company           string   `json:"company"`
	Location          string   `json:"location"`
	IsRemote          bool     `json:"is_remote"`
	Source            string   `json:"source"`
	URL               string   `json:"url"`
	PostedDate        string   `json:"posted_date"`
	ScrapedAt         string   `json:"scraped_at"`
	Seniority         string   `json:"seniority"`
	Summary           string   `json:"summary"`
	RawDescription    string   `json:"raw_description"`
	MatchScore        int      `json:"match_score"`
	MatchReasoning    string   `json:"match_reasoning"`
	TechStack         []string `json:"tech_stack"`
	IsMatched         bool     `json:"is_matched"`
	SalaryMin         *int     `json:"salary_min"`
	SalaryMax         *int     `json:"salary_max"`
	Currency          string   `json:"currency"`
	IsViewed          bool     `json:"is_viewed"`
	ApplicationStatus string   `json:"application_status"`
	ViewedAt          *string  `json:"viewed_at"`
	IsNew             bool     `json:"is_new"`
}

/*
GetMatchedJobs returns jobs evaluated for the authenticated user within the 14-day retention window.
When viewed_only=true, orders strictly by ujv.viewed_at DESC (latest viewed first).
Otherwise, orders by match_score DESC, unviewed first, and scraped date DESC.
*/
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

	viewedOnly := c.DefaultQuery("viewed_only", "false") == "true"
	unviewedOnly := c.DefaultQuery("unviewed_only", "false") == "true"

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

	var conditions []string
	var args []interface{}
	args = append(args, userID)

	// Enforce 14-day retention policy window and exclude dismissed jobs
	conditions = append(conditions, "j.scraped_at >= NOW() - INTERVAL '14 days'")
	conditions = append(conditions, "COALESCE(ujm.is_dismissed, false) = false")

	if unviewedOnly {
		conditions = append(conditions, "ujv.viewed_at IS NULL")
	} else if viewedOnly {
		conditions = append(conditions, "ujv.viewed_at IS NOT NULL")
	}

	if minScore > 0 {
		args = append(args, minScore)
		conditions = append(conditions, fmt.Sprintf("COALESCE(ujm.match_score, 0) >= $%d", len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			whereClause += " AND " + conditions[i]
		}
	}

	args = append(args, limit)
	limitParamIndex := fmt.Sprintf("$%d", len(args))
	args = append(args, offset)
	offsetParamIndex := fmt.Sprintf("$%d", len(args))

	orderByClause := `
		ORDER BY
			COALESCE(ujm.match_score, 0) DESC,
			CASE WHEN ujv.viewed_at IS NULL THEN 0 ELSE 1 END ASC,
			j.scraped_at DESC`

	if viewedOnly {
		orderByClause = `
		ORDER BY
			ujv.viewed_at DESC`
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
			COALESCE(j.scraped_at::text, ''),
			COALESCE(ujm.seniority, ''),
			COALESCE(j.summary, ''),
			COALESCE(j.raw_desc, ''),
			COALESCE(ujm.match_score, 0) AS match_score,
			COALESCE(ujm.match_reasoning, ''),
			COALESCE(ujm.tech_stack::text, '[]'),
			COALESCE(ujm.is_ai_matched, false) AS is_matched,
			j.salary_min,
			j.salary_max,
			COALESCE(j.currency, 'USD'),
			(ujv.viewed_at IS NOT NULL) AS is_viewed,
			COALESCE(app.status, 'unapplied') AS application_status,
			ujv.viewed_at::text AS viewed_at,
			(j.scraped_at >= NOW() - INTERVAL '24 hours' AND ujv.viewed_at IS NULL) AS is_new
		FROM jobs j
		LEFT JOIN user_job_matches ujm ON ujm.job_id = j.id AND ujm.user_id = $1
		LEFT JOIN companies comp ON j.company_id = comp.id
		LEFT JOIN user_job_views ujv ON ujv.user_id = $1 AND ujv.job_id = j.id
		LEFT JOIN applications app ON app.user_id = $1 AND app.job_id = j.id
		` + whereClause + orderByClause + `
		LIMIT ` + limitParamIndex + ` OFFSET ` + offsetParamIndex + `;
	`

	rows, err := h.DB.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query matched jobs: " + err.Error()})
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
			&job.ScrapedAt,
			&job.Seniority,
			&job.Summary,
			&job.RawDescription,
			&job.MatchScore,
			&job.MatchReasoning,
			&techStackRaw,
			&job.IsMatched,
			&job.SalaryMin,
			&job.SalaryMax,
			&job.Currency,
			&job.IsViewed,
			&job.ApplicationStatus,
			&job.ViewedAt,
			&job.IsNew,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan matched job row: " + err.Error()})
			return
		}

		if err := unmarshalStringJSON(techStackRaw, &job.TechStack); err != nil {
			job.TechStack = []string{}
		}

		if job.SalaryMin == nil && job.SalaryMax == nil {
			salMin, salMax := utils.ExtractSalaryFromText(job.RawDescription)
			if salMin != nil && salMax != nil {
				job.SalaryMin = salMin
				job.SalaryMax = salMax
			}
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

/*
GetMatchStatus checks whether AI matching evaluations are pending or active for recent jobs.
*/
func (h *MatchedJobsHandler) GetMatchStatus(c *gin.Context) {
	var count int
	query := `
		SELECT COUNT(*) 
		FROM jobs 
		WHERE ai_evaluated = false 
		  AND scraped_at >= NOW() - INTERVAL '14 days';
	`
	err := h.DB.QueryRow(c.Request.Context(), query).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"is_evaluating": false,
			"pending_count": 0,
			"error":         "Failed to query match engine status",
		})
		return
	}

	isEvaluating := count > 0
	c.JSON(http.StatusOK, gin.H{
		"is_evaluating": isEvaluating,
		"pending_count": count,
	})
}

/*
GetMatchedJobByID retrieves full details and match metadata for a single specific job.
*/
func (h *MatchedJobsHandler) GetMatchedJobByID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	jobID := c.Param("id")

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
			COALESCE(j.scraped_at::text, ''),
			COALESCE(ujm.seniority, ''),
			COALESCE(j.summary, ''),
			COALESCE(j.raw_desc, ''),
			COALESCE(ujm.match_score, 0) AS match_score,
			COALESCE(ujm.match_reasoning, ''),
			COALESCE(ujm.tech_stack::text, '[]'),
			COALESCE(ujm.is_ai_matched, false) AS is_matched,
			j.salary_min,
			j.salary_max,
			COALESCE(j.currency, 'USD'),
			(ujv.viewed_at IS NOT NULL) AS is_viewed,
			COALESCE(app.status, 'unapplied') AS application_status,
			ujv.viewed_at::text AS viewed_at,
			(j.scraped_at >= NOW() - INTERVAL '24 hours' AND ujv.viewed_at IS NULL) AS is_new
		FROM jobs j
		LEFT JOIN user_job_matches ujm ON ujm.job_id = j.id AND ujm.user_id = $1
		LEFT JOIN companies comp ON j.company_id = comp.id
		LEFT JOIN user_job_views ujv ON ujv.user_id = $1 AND ujv.job_id = j.id
		LEFT JOIN applications app ON app.user_id = $1 AND app.job_id = j.id
		WHERE j.id = $2
		LIMIT 1;
	`

	var job MatchedJobResponse
	var techStackRaw string
	err := h.DB.QueryRow(c.Request.Context(), query, userID, jobID).Scan(
		&job.JobID,
		&job.Title,
		&job.Company,
		&job.Location,
		&job.IsRemote,
		&job.Source,
		&job.URL,
		&job.PostedDate,
		&job.ScrapedAt,
		&job.Seniority,
		&job.Summary,
		&job.RawDescription,
		&job.MatchScore,
		&job.MatchReasoning,
		&techStackRaw,
		&job.IsMatched,
		&job.SalaryMin,
		&job.SalaryMax,
		&job.Currency,
		&job.IsViewed,
		&job.ApplicationStatus,
		&job.ViewedAt,
		&job.IsNew,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	if techStackRaw != "" {
		_ = unmarshalStringJSON(techStackRaw, &job.TechStack)
	}
	if job.TechStack == nil {
		job.TechStack = []string{}
	}

	c.JSON(http.StatusOK, gin.H{"data": job})
}
