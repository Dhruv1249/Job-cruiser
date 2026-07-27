package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/Dhruv1249/Job-cruiser/backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IngestHandler struct {
	DB           *pgxpool.Pool
	MatchService services.BatchMatchEvaluator
}

type StartRunResponse struct {
	RunID string `json:"run_id"`
}

type IngestJobPayload struct {
	JobID           interface{} `json:"job_id"`
	Title           string      `json:"title"`
	Company         string      `json:"company"`
	Source          string      `json:"source"`
	UpdatedAt       string      `json:"updated_at"`
	AbsoluteURL     string      `json:"absolute_url"`
	Location        string      `json:"location"`
	Departments     []string    `json:"departments"`
	Offices         []string    `json:"offices"`
	DescriptionText string      `json:"description_text"`
	Seniority       string      `json:"seniority"`
	Summary         string      `json:"summary"`
	TechStack       []string    `json:"tech_stack"`
	SalaryMin       int         `json:"salary_min"`
	SalaryMax       int         `json:"salary_max"`
	Currency        string      `json:"currency"`
}

type IngestRawRequest struct {
	RunID string             `json:"run_id" binding:"required"`
	Jobs  []IngestJobPayload `json:"jobs" binding:"required"`
}


type IngestRequest struct {
	RunID   string             `json:"run_id" binding:"required"`
	Company string             `json:"company" binding:"required"`
	Jobs    []IngestJobPayload `json:"jobs" binding:"required"`
}

type FinishRequest struct {
	RunID        string `json:"run_id" binding:"required"`
	Status       string `json:"status" binding:"required"` // 'success' or 'failed'
	ErrorMessage string `json:"error_message"`
}

// StartRun registers a new scraper run in the telemetry tracking tables
func (h *IngestHandler) StartRun(c *gin.Context) {
	var runID string
	query := `
		INSERT INTO scraper_runs (started_at, status, jobs_added, sources_hit)
		VALUES (CURRENT_TIMESTAMP, 'running', 0, '[]'::jsonb)
		RETURNING id;
	`
	err := h.DB.QueryRow(context.Background(), query).Scan(&runID)
	if err != nil {
		log.Printf("Failed to start scraper run: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start scraper run recording"})
		return
	}

	c.JSON(http.StatusOK, StartRunResponse{RunID: runID})
}

// IngestRaw stores the complete batch of scraped jobs from a scraper run without
// any AI filtering. Each job is upserted by URL and marked ai_evaluated=false so
// the Mistral batch matcher picks it up after the run finishes.
func (h *IngestHandler) IngestRaw(c *gin.Context) {
	var req IngestRawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input payload: " + err.Error()})
		return
	}

	ctx := context.Background()

	var currentStatus string
	checkRunQuery := `SELECT status FROM scraper_runs WHERE id = $1`
	err := h.DB.QueryRow(ctx, checkRunQuery, req.RunID).Scan(&currentStatus)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scraper run not found"})
		return
	}
	if currentStatus != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scraper run is not active"})
		return
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback(ctx)

	insertedCount := 0
	companyCache := make(map[string]string)

	for _, job := range req.Jobs {
		if job.AbsoluteURL == "" {
			continue
		}

		companyName := utils.ExtractCompanyName(job.Company, job.AbsoluteURL, job.Title)

		var companyID string
		cacheKey := strings.ToLower(companyName)
		if cachedID, exists := companyCache[cacheKey]; exists {
			companyID = cachedID
		} else {
			compLookup := `SELECT id FROM companies WHERE LOWER(name) = LOWER($1)`
			compErr := tx.QueryRow(ctx, compLookup, companyName).Scan(&companyID)
			if compErr != nil {
				insertCompQuery := `INSERT INTO companies (name) VALUES ($1) RETURNING id`
				if scanErr := tx.QueryRow(ctx, insertCompQuery, companyName).Scan(&companyID); scanErr != nil {
					continue
				}
			}
			companyCache[cacheKey] = companyID
		}

		loc := job.Location
		isRemote := strings.Contains(strings.ToLower(loc), "remote") ||
			strings.Contains(strings.ToLower(loc), "anywhere") ||
			strings.Contains(strings.ToLower(loc), "wfh")

		source := job.Source
		if source == "" {
			source = "unknown"
		}

		var tags []string
		for _, dep := range job.Departments {
			if dep != "" {
				tags = append(tags, strings.ToLower(dep))
			}
		}
		for _, ts := range job.TechStack {
			if ts != "" {
				tags = append(tags, strings.ToLower(ts))
			}
		}
		tagsJSON, _ := json.Marshal(tags)

		jobType := "Full-time"
		titleLower := strings.ToLower(job.Title)
		if strings.Contains(titleLower, "intern") || strings.Contains(titleLower, "co-op") {
			jobType = "Internship"
		} else if strings.Contains(titleLower, "contract") {
			jobType = "Contract"
		}

		var salMinParam, salMaxParam *int
		if job.SalaryMin > 0 {
			salMinParam = &job.SalaryMin
		}
		if job.SalaryMax > 0 {
			salMaxParam = &job.SalaryMax
		}
		curr := job.Currency
		if curr == "" {
			curr = "USD"
		}

		upsertQuery := `
			INSERT INTO jobs (company_id, title, location, is_remote, source, url, tags, raw_desc, job_type,
			                  salary_min, salary_max, currency, scraped_at, ai_evaluated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP, false)
			ON CONFLICT (url) DO NOTHING
			RETURNING id;
		`
		var insertedID string
		execErr := tx.QueryRow(ctx, upsertQuery,
			companyID, job.Title, loc, isRemote, source, job.AbsoluteURL,
			tagsJSON, job.DescriptionText, jobType, salMinParam, salMaxParam, curr,
		).Scan(&insertedID)
		if execErr != nil {
			continue
		}
		if insertedID != "" {
			insertedCount++
		}
	}

	updateRunQuery := `
		UPDATE scraper_runs SET jobs_added = jobs_added + $1 WHERE id = $2;
	`
	if _, execErr := tx.Exec(ctx, updateRunQuery, insertedCount, req.RunID); execErr != nil {
		log.Printf("IngestRaw: failed to update run telemetry: %v", execErr)
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit raw job ingestion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Raw jobs ingested successfully",
		"jobs_added": insertedCount,
	})
}

// IngestJobs processes a batch of jobs for a company and registers them in CockroachDB
func (h *IngestHandler) IngestJobs(c *gin.Context) {
	var req IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input payload: " + err.Error()})
		return
	}

	ctx := context.Background()

	// 1. Verify that the run exists and is running
	var currentStatus string
	var currentSourcesJSON []byte
	checkRunQuery := `SELECT status, sources_hit FROM scraper_runs WHERE id = $1`
	err := h.DB.QueryRow(ctx, checkRunQuery, req.RunID).Scan(&currentStatus, &currentSourcesJSON)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scraper run not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error validating scraper run"})
		return
	}

	if currentStatus != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scraper run is not active"})
		return
	}

	// 2. Lookup or create company by name
	var companyID string
	cleanCompanyName := strings.TrimSpace(req.Company)
	compQuery := `SELECT id FROM companies WHERE LOWER(name) = LOWER($1)`
	err = h.DB.QueryRow(ctx, compQuery, cleanCompanyName).Scan(&companyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Create new company
			insertCompQuery := `
				INSERT INTO companies (name, domain)
				VALUES ($1, $2)
				RETURNING id;
			`
			domain := strings.ToLower(cleanCompanyName) + ".com"
			err = h.DB.QueryRow(ctx, insertCompQuery, cleanCompanyName, domain).Scan(&companyID)
			if err != nil {
				log.Printf("Failed to insert company %s: %v", cleanCompanyName, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register company"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database lookup error"})
			return
		}
	}

	// 3. Insert or Update Jobs in a transaction
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback(ctx)

	insertedCount := 0

	for _, job := range req.Jobs {
		if job.AbsoluteURL == "" {
			continue
		}

		// Detect if the job is remote based on location string
		loc := job.Location
		isRemote := false
		locLower := strings.ToLower(loc)
		if strings.Contains(locLower, "remote") || strings.Contains(locLower, "anywhere") || strings.Contains(locLower, "wfh") {
			isRemote = true
		}

		// Extract tech keywords and add departments, excluding incorrect location offices
		var tags []string
		for _, dep := range job.Departments {
			if dep != "" {
				tags = append(tags, strings.ToLower(dep))
			}
		}
		
		if len(job.TechStack) > 0 {
			for _, ts := range job.TechStack {
				if ts != "" {
					tags = append(tags, strings.ToLower(ts))
				}
			}
		} else {
			techTags := ExtractTechTags(job.Title, job.DescriptionText)
			tags = append(tags, techTags...)
		}
		
		tagsJSON, _ := json.Marshal(tags)

		// Determine Job Type (rough heuristic)
		jobType := "Full-time"
		titleLower := strings.ToLower(job.Title)
		if strings.Contains(titleLower, "intern") || strings.Contains(titleLower, "co-op") {
			jobType = "Internship"
		} else if strings.Contains(titleLower, "contract") || strings.Contains(titleLower, "temp") {
			jobType = "Contract"
		} else if strings.Contains(titleLower, "part-time") || strings.Contains(titleLower, "parttime") {
			jobType = "Part-time"
		}

		expRequired := ExtractExperience(job.Title, job.DescriptionText)
		var expParam *string
		if expRequired != "" {
			expParam = &expRequired
		}

		var salMinParam, salMaxParam *int
		if job.SalaryMin > 0 {
			salMinParam = &job.SalaryMin
		}
		if job.SalaryMax > 0 {
			salMaxParam = &job.SalaryMax
		}

		curr := job.Currency
		if curr == "" {
			curr = "USD"
		}

		var seniorityParam, summaryParam *string
		if job.Seniority != "" {
			seniorityParam = &job.Seniority
		}
		if job.Summary != "" {
			summaryParam = &job.Summary
		}

		jobQuery := `
			INSERT INTO jobs (company_id, title, location, is_remote, source, url, tags, raw_desc, job_type, experience_required, salary_min, salary_max, currency, seniority, summary, scraped_at, ai_evaluated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, CURRENT_TIMESTAMP, false)
			ON CONFLICT (url) DO NOTHING
			RETURNING id;
		`
		source := job.Source
		if source == "" {
			source = "unknown"
		}
		var insertedID string
		err = tx.QueryRow(ctx, jobQuery, companyID, job.Title, loc, isRemote, source, job.AbsoluteURL, tagsJSON, job.DescriptionText, jobType, expParam, salMinParam, salMaxParam, curr, seniorityParam, summaryParam).Scan(&insertedID)
		if err != nil {
			continue
		}
		if insertedID != "" {
			insertedCount++
		}
	}

	// 4. Update the scraper run telemetry details
	var sources []string
	_ = json.Unmarshal(currentSourcesJSON, &sources)
	
	// Add company name if not already listed
	alreadyExists := false
	for _, src := range sources {
		if strings.EqualFold(src, cleanCompanyName) {
			alreadyExists = true
			break
		}
	}
	if !alreadyExists {
		sources = append(sources, cleanCompanyName)
	}
	updatedSourcesJSON, _ := json.Marshal(sources)

	updateRunQuery := `
		UPDATE scraper_runs
		SET jobs_added = jobs_added + $1,
		    sources_hit = $2
		WHERE id = $3;
	`
	_, err = tx.Exec(ctx, updateRunQuery, insertedCount, updatedSourcesJSON, req.RunID)
	if err != nil {
		log.Printf("Failed to update scraper run telemetry: %v", err)
		// We can still proceed if the jobs were inserted, but rolling back to maintain consistency
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update run telemetry"})
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit job ingestion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Jobs ingested successfully",
		"company":    cleanCompanyName,
		"jobs_added": insertedCount,
	})
}

// FinishRun marks a scraper run as completed and triggers AI evaluation in the background.
func (h *IngestHandler) FinishRun(c *gin.Context) {
	var req FinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input payload"})
		return
	}

	statusClean := strings.ToLower(req.Status)
	if statusClean != "success" && statusClean != "failed" {
		statusClean = "finished"
	}

	query := `
		UPDATE scraper_runs
		SET status = $1,
		    finished_at = CURRENT_TIMESTAMP,
		    error_message = $2
		WHERE id = $3;
	`
	_, err := h.DB.Exec(context.Background(), query, statusClean, req.ErrorMessage, req.RunID)
	if err != nil {
		log.Printf("Failed to finish scraper run: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scraper run closure status"})
		return
	}

	if statusClean == "success" && h.MatchService != nil {
		go func() {
			log.Printf("[IngestHandler] Triggering AI evaluation pass.")
			h.MatchService.EvaluatePendingForAllUsers(context.Background())
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Scraper run recorded as completed"})
}

// ==========================================================
// TECH STACK KEYWORD EXTRACTION HELPERS
// ==========================================================

var knownTechKeywords = []string{
	"go", "golang", "python", "java", "javascript", "typescript", "react", "vue", "angular",
	"node", "nodejs", "rust", "c++", "c#", ".net", "ruby", "rails", "php", "aws", "gcp", "azure",
	"docker", "kubernetes", "postgres", "postgresql", "mysql", "redis", "mongodb", "sqlite",
	"kafka", "graphql", "rest", "grpc", "microservices", "swift", "kotlin", "flutter", "dart",
	"terraform", "pytorch", "tensorflow", "ci/cd", "html", "css", "sql", "nosql", "django",
	"flask", "spring", "spark", "hadoop",
}

func ExtractTechTags(title, description string) []string {
	text := strings.ToLower(title + " " + description)
	var tags []string
	
	for _, kw := range knownTechKeywords {
		if ContainsWord(text, kw) {
			tags = append(tags, kw)
		}
	}
	return tags
}

func ContainsWord(text, word string) bool {
	index := 0
	for {
		i := strings.Index(text[index:], word)
		if i == -1 {
			return false
		}
		start := index + i
		end := start + len(word)
		
		startOk := start == 0 || !IsAlphanumeric(text[start-1])
		endOk := end == len(text) || !IsAlphanumeric(text[end])
		
		if startOk && endOk {
			return true
		}
		index = end
		if index >= len(text) {
			break
		}
	}
	return false
}

func IsAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '#' || c == '/' || c == '-'
}

// ==========================================================
// EXPERIENCE EXTRACTION HELPERS
// ==========================================================

var rangeRegex = regexp.MustCompile(`\b(\d+)\s*(?:-|to)\s*(\d+)\s*(?:years?|yrs?)\b`)
var plusRegex = regexp.MustCompile(`\b(\d+)\s*\+\s*(?:years?|yrs?)\b`)
var minRegex = regexp.MustCompile(`(?i)\b(?:at\s+least|minimum\s+of|requires?|with)\s+(\d+)\s*(?:years?|yrs?)\b`)
var simpleRegex = regexp.MustCompile(`(?i)\b(\d+)\s*(?:years?|yrs?)(?:\s+of)?\s+experience\b`)

func ExtractExperience(title, description string) string {
	text := strings.ToLower(title + " " + description)

	// 1. Try ranges first (e.g. "3-5 years")
	if loc := rangeRegex.FindStringSubmatch(text); len(loc) == 3 {
		return fmt.Sprintf("%s-%s years", loc[1], loc[2])
	}

	// 2. Try plus format (e.g. "5+ years")
	if loc := plusRegex.FindStringSubmatch(text); len(loc) == 2 {
		return fmt.Sprintf("%s+ years", loc[1])
	}

	// 3. Try minimum prefix matches (e.g. "at least 3 years")
	if loc := minRegex.FindStringSubmatch(text); len(loc) == 2 {
		return fmt.Sprintf("%s+ years", loc[1])
	}

	// 4. Try simple suffix matches (e.g. "5 years of experience")
	if loc := simpleRegex.FindStringSubmatch(text); len(loc) == 2 {
		return fmt.Sprintf("%s+ years", loc[1])
	}

	return ""
}
