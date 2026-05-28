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
	// Get pagination parameters from the URL, with safe defaults
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Prevent users from requesting 10,000 jobs at once and crashing your DB
	if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	// Calculate the offset (e.g., Page 2 with limit 20 means skip the first 20)
	offset := (page - 1) * limit

	// Inject the variables safely using $1 and $2
	query := `
		SELECT id, company_id, title, location, salary, experience_required, 
		       job_type, is_easy_apply, is_remote, source, url, posted_date, 
		       tags, score, scraped_at 
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
		err := rows.Scan(
			&j.ID, &j.CompanyID, &j.Title, &j.Location, &j.Salary, &j.ExperienceRequired,
			&j.JobType, &j.IsEasyApply, &j.IsRemote, &j.Source, &j.URL, &j.PostedDate,
			&j.Tags, &j.Score, &j.ScrapedAt,
		)
		if err != nil {
			continue
		}
		jobs = append(jobs, j)
	}

	if jobs == nil {
		jobs = []models.Job{}
	}

	// Send metadata alongside the data so Flutter knows what page it is on
	c.JSON(http.StatusOK, gin.H{
		"data":  jobs,
		"page":  page,
		"limit": limit,
	})
}
