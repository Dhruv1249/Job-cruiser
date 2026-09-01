package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func UnmarshalStringJSON(raw string, target interface{}) error {
	return json.Unmarshal([]byte(raw), target)
}

func unmarshalStringJSON(raw string, target interface{}) error {
	return UnmarshalStringJSON(raw, target)
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
GetMatchedJobs returns jobs evaluated for the authenticated user within the retention window.
Supports dynamic filtering by min_score, max_score, days, match_scope, remote_only, viewed status, and sort_by.
*/
func (h *MatchedJobsHandler) GetMatchedJobs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	minScoreParam := c.DefaultQuery("min_score", "0")
	minScore, minScoreErr := strconv.Atoi(minScoreParam)
	if minScoreErr != nil || minScore < 0 || minScore > 100 {
		minScore = 0
	}

	maxScoreParam := c.DefaultQuery("max_score", "100")
	maxScore, maxScoreErr := strconv.Atoi(maxScoreParam)
	if maxScoreErr != nil || maxScore < 0 || maxScore > 100 {
		maxScore = 100
	}

	daysParam := c.DefaultQuery("days", "0")
	daysFilter, daysErr := strconv.Atoi(daysParam)
	if daysErr != nil || daysFilter < 0 {
		daysFilter = 0
	}

	matchScope := c.DefaultQuery("match_scope", "all")
	remoteOnly := c.DefaultQuery("remote_only", "false") == "true"
	viewedOnly := c.DefaultQuery("viewed_only", "false") == "true"
	unviewedOnly := c.DefaultQuery("unviewed_only", "false") == "true"
	sortBy := c.DefaultQuery("sort_by", "score_desc")
	applicationStatus := c.DefaultQuery("application_status", "all")
	searchQuery := strings.TrimSpace(c.DefaultQuery("search", ""))

	limitParam := c.DefaultQuery("limit", "50")
	limit, limitErr := strconv.Atoi(limitParam)
	if limitErr != nil || limit <= 0 || limit > 200 {
		limit = 50
	}

	offsetParam := c.DefaultQuery("offset", "0")
	offset, offsetErr := strconv.Atoi(offsetParam)
	if offsetErr != nil || offset < 0 {
		offset = 0
	}

	var conditions []string
	var args []interface{}
	args = append(args, userID)

	if daysFilter > 0 {
		conditions = append(conditions, fmt.Sprintf("j.scraped_at >= NOW() - INTERVAL '%d days'", daysFilter))
	}
	conditions = append(conditions, "COALESCE(ujm.is_dismissed, false) = false")

	if unviewedOnly {
		conditions = append(conditions, "ujv.viewed_at IS NULL")
	} else if viewedOnly {
		conditions = append(conditions, "ujv.viewed_at IS NOT NULL")
	}

	if matchScope == "matched_only" {
		conditions = append(conditions, "(COALESCE(ujm.is_ai_matched, false) = true OR COALESCE(ujm.match_score, 0) > 0)")
	} else if matchScope == "unmatched_only" {
		conditions = append(conditions, "(ujm.match_score IS NULL OR ujm.match_score = 0)")
	}

	if minScore > 0 {
		args = append(args, minScore)
		conditions = append(conditions, fmt.Sprintf("COALESCE(ujm.match_score, 0) >= $%d", len(args)))
	}

	if maxScore < 100 {
		args = append(args, maxScore)
		conditions = append(conditions, fmt.Sprintf("COALESCE(ujm.match_score, 0) <= $%d", len(args)))
	}

	if remoteOnly {
		conditions = append(conditions, "j.is_remote = true")
	}

	if applicationStatus == "unapplied" {
		conditions = append(conditions, "(app.status IS NULL OR app.status = 'unapplied')")
	} else if applicationStatus != "all" {
		args = append(args, applicationStatus)
		conditions = append(conditions, fmt.Sprintf("app.status = $%d", len(args)))
	}

	if searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		searchParamIndex := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"(j.title ILIKE $%d OR COALESCE(comp.name, '') ILIKE $%d OR COALESCE(j.location, '') ILIKE $%d OR COALESCE(j.summary, '') ILIKE $%d)",
			searchParamIndex, searchParamIndex, searchParamIndex, searchParamIndex,
		))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + conditions[0]
		for index := 1; index < len(conditions); index++ {
			whereClause += " AND " + conditions[index]
		}
	}

	args = append(args, limit)
	limitParamIndex := fmt.Sprintf("$%d", len(args))
	args = append(args, offset)
	offsetParamIndex := fmt.Sprintf("$%d", len(args))

	var orderByClause string
	if viewedOnly {
		orderByClause = "\n\t\tORDER BY ujv.viewed_at DESC"
	} else {
		switch sortBy {
		case "date_desc":
			orderByClause = "\n\t\tORDER BY j.scraped_at DESC"
		case "date_asc":
			orderByClause = "\n\t\tORDER BY j.scraped_at ASC"
		case "salary_desc":
			orderByClause = "\n\t\tORDER BY COALESCE(j.salary_max, j.salary_min, 0) DESC, j.scraped_at DESC"
		default:
			orderByClause = `
		ORDER BY
			COALESCE(ujm.match_score, 0) DESC,
			CASE WHEN ujv.viewed_at IS NULL THEN 0 ELSE 1 END ASC,
			j.scraped_at DESC`
		}
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
