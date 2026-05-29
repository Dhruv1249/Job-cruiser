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
		SELECT id, company_id, title, location, salary_min, salary_max, currency, 
		       experience_required, job_type, is_easy_apply, is_remote, source, 
		       url, posted_date, tags, scraped_at 
		FROM jobs 
		ORDER BY scraped_at DESC 
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
		// Scan updated to match the exact order of the SELECT statement
		err := rows.Scan(
			&j.ID, &j.CompanyID, &j.Title, &j.Location, &j.SalaryMin, &j.SalaryMax, &j.Currency,
			&j.ExperienceRequired, &j.JobType, &j.IsEasyApply, &j.IsRemote, &j.Source,
			&j.URL, &j.PostedDate, &j.Tags, &j.ScrapedAt,
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
