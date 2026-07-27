package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Dhruv1249/Job-cruiser/backend/models"
)

type JobHandler struct {
	DB *pgxpool.Pool
}

// GetJobs fetches the latest scraped jobs from the database
func (h *JobHandler) GetJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	// Updated query: Replaced salary and score with salary_min, salary_max, currency
	query := `
		SELECT j.id, j.company_id, COALESCE(comp.name, ''), j.title, j.location, j.salary_min, j.salary_max, j.currency, 
		       j.experience_required, j.job_type, j.is_easy_apply, j.is_remote, j.source, 
		       j.url, j.posted_date, j.tags, COALESCE(j.summary, ''), COALESCE(j.raw_desc, ''), j.scraped_at 
		FROM jobs j
		LEFT JOIN companies comp ON j.company_id = comp.id
		ORDER BY j.scraped_at DESC 
		LIMIT $1 OFFSET $2;
	`

	rows, err := h.DB.Query(context.Background(), query, limit, offset)
	if err != nil {
		log.Printf("DATABASE ERROR IN GETJOBS: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var j models.Job
		err := rows.Scan(
			&j.ID, &j.CompanyID, &j.Company, &j.Title, &j.Location, &j.SalaryMin, &j.SalaryMax, &j.Currency,
			&j.ExperienceRequired, &j.JobType, &j.IsEasyApply, &j.IsRemote, &j.Source,
			&j.URL, &j.PostedDate, &j.Tags, &j.Summary, &j.RawDescription, &j.ScrapedAt,
		)
		if err != nil {
			log.Printf("Row scan error: %v", err) // Helpful for debugging struct mismatches
			continue
		}
		jobs = append(jobs, j)
	}

	if jobs == nil {
		jobs = []models.Job{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  jobs,
		"page":  page,
		"limit": limit,
	})
}

/*
GetMasterKeywords returns all approved master keywords for scraping and search matching.
*/
func (h *JobHandler) GetMasterKeywords(c *gin.Context) {
	query := `SELECT keyword FROM master_keywords ORDER BY keyword ASC;`
	rows, err := h.DB.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch master keywords"})
		return
	}
	defer rows.Close()

	var keywords []string
	for rows.Next() {
		var kw string
		if err := rows.Scan(&kw); err == nil {
			keywords = append(keywords, kw)
		}
	}
	if keywords == nil {
		keywords = []string{}
	}

	c.JSON(http.StatusOK, gin.H{"data": keywords})
}

/*
MarkJobViewed records that the authenticated user viewed a specific job detail page.
*/
func (h *JobHandler) MarkJobViewed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID required"})
		return
	}

	query := `
		INSERT INTO user_job_views (user_id, job_id, viewed_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, job_id) DO NOTHING;
	`
	_, err := h.DB.Exec(context.Background(), query, userID, jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record job view"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job marked as viewed", "job_id": jobID})
}

