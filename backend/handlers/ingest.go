package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
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
	RunID        string          `json:"run_id" binding:"required"`
	Status       string          `json:"status" binding:"required"`
	ErrorMessage string          `json:"error_message"`
	SourcesHit   json.RawMessage `json:"sources_hit"`
}

// StartRun registers a new scraper run in the telemetry tracking tables.
// Any existing runs that have been in 'running' status for longer than 4 hours
// are automatically cancelled before the new run is created.
func (h *IngestHandler) StartRun(c *gin.Context) {
	ctx := context.Background()

	staleRunCancelQuery := `
		UPDATE scraper_runs
		SET status = 'cancelled',
		    finished_at = CURRENT_TIMESTAMP,
		    error_message = 'Auto-cancelled: run exceeded 4-hour timeout without receiving a finish signal'
		WHERE status = 'running'
		  AND started_at < CURRENT_TIMESTAMP - INTERVAL '4 hours';
	`
	if _, cancelErr := h.DB.Exec(ctx, staleRunCancelQuery); cancelErr != nil {
		log.Printf("StartRun: failed to auto-cancel stale runs: %v", cancelErr)
	}

	var runID string
	query := `
		INSERT INTO scraper_runs (started_at, status, jobs_added, sources_hit)
		VALUES (CURRENT_TIMESTAMP, 'running', 0, '[]'::jsonb)
		RETURNING id;
	`
	err := h.DB.QueryRow(ctx, query).Scan(&runID)
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
	batchSourceFound := make(map[string]int)
	batchSourceAdded := make(map[string]int)
	batchCompanyAdded := make(map[string]int)

	for jobIndex, job := range req.Jobs {
		if job.AbsoluteURL == "" {
			continue
		}

		normalizedSource := strings.ToLower(strings.TrimSpace(job.Source))
		if normalizedSource == "" {
			normalizedSource = "unknown"
		}
		batchSourceFound[normalizedSource]++

		savepointName := fmt.Sprintf("sp_%d", jobIndex)
		if _, spErr := tx.Exec(ctx, fmt.Sprintf("SAVEPOINT %s", savepointName)); spErr != nil {
			log.Printf("IngestRaw: failed to set savepoint for job %d: %v", jobIndex, spErr)
			continue
		}

		jobInserted, insertErr := insertSingleJob(ctx, tx, job, companyCache)
		if insertErr != nil {
			log.Printf("IngestRaw: rolling back job %d (%s) due to error: %v", jobIndex, job.AbsoluteURL, insertErr)
			if _, rbErr := tx.Exec(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepointName)); rbErr != nil {
				log.Printf("IngestRaw: savepoint rollback failed for job %d, aborting batch: %v", jobIndex, rbErr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit raw job ingestion"})
				return
			}
			continue
		}
		if _, relErr := tx.Exec(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", savepointName)); relErr != nil {
			log.Printf("IngestRaw: failed to release savepoint for job %d: %v", jobIndex, relErr)
		}
		if jobInserted {
			insertedCount++
			batchSourceAdded[normalizedSource]++
			extractedCompany := utils.ExtractCompanyName(job.Company, job.AbsoluteURL, job.Title)
			if extractedCompany != "" {
				batchCompanyAdded[extractedCompany]++
			}
		}
	}

	var existingSourcesJSON, existingCompaniesJSON []byte
	_ = tx.QueryRow(ctx, `SELECT COALESCE(sources_hit, '{}'::jsonb), COALESCE(companies_hit, '[]'::jsonb) FROM scraper_runs WHERE id = $1`, req.RunID).Scan(&existingSourcesJSON, &existingCompaniesJSON)

	cumulativeSources := make(map[string]utils.ScraperPlatformMetric)
	if len(existingSourcesJSON) > 0 {
		var rawMap map[string]interface{}
		if unmarshalErr := json.Unmarshal(existingSourcesJSON, &rawMap); unmarshalErr == nil {
			for sourceName, val := range rawMap {
				metric := utils.ScraperPlatformMetric{}
				if numCount, isNum := val.(float64); isNum {
					metric.JobsFound = int(numCount)
				} else if m, isMap := val.(map[string]interface{}); isMap {
					if jf, ok := m["jobs_found"].(float64); ok {
						metric.JobsFound = int(jf)
					}
					if ja, ok := m["jobs_added"].(float64); ok {
						metric.JobsAdded = int(ja)
					}
					if ds, ok := m["duration_seconds"].(float64); ok {
						metric.DurationSeconds = ds
					}
					if qc, ok := m["query_count"].(float64); ok {
						metric.QueryCount = int(qc)
					}
				}
				cumulativeSources[sourceName] = metric
			}
		}
	}
	for sourceName, foundCount := range batchSourceFound {
		metric := cumulativeSources[sourceName]
		metric.JobsFound += foundCount
		metric.JobsAdded += batchSourceAdded[sourceName]
		if metric.QueryCount == 0 {
			metric.QueryCount = 1
		}
		cumulativeSources[sourceName] = metric
	}
	updatedSourcesJSON, _ := json.Marshal(cumulativeSources)

	cumulativeCompanies := make(map[string]int)
	if len(existingCompaniesJSON) > 0 {
		var compList []struct {
			CompanyName string `json:"company_name"`
			JobsAdded   int    `json:"jobs_added"`
		}
		if unmarshalErr := json.Unmarshal(existingCompaniesJSON, &compList); unmarshalErr == nil {
			for _, item := range compList {
				cumulativeCompanies[item.CompanyName] += item.JobsAdded
			}
		}
	}
	for compName, addCount := range batchCompanyAdded {
		cumulativeCompanies[compName] += addCount
	}

	type companyCountItem struct {
		CompanyName string `json:"company_name"`
		JobsAdded   int    `json:"jobs_added"`
	}
	var topCompanyList []companyCountItem
	for compName, count := range cumulativeCompanies {
		topCompanyList = append(topCompanyList, companyCountItem{CompanyName: compName, JobsAdded: count})
	}
	sort.Slice(topCompanyList, func(i, j int) bool {
		return topCompanyList[i].JobsAdded > topCompanyList[j].JobsAdded
	})
	if len(topCompanyList) > 30 {
		topCompanyList = topCompanyList[:30]
	}
	updatedCompaniesJSON, _ := json.Marshal(topCompanyList)

	updateRunQuery := `
		UPDATE scraper_runs
		SET jobs_added = jobs_added + $1,
		    sources_hit = $2,
		    companies_hit = $3
		WHERE id = $4;
	`
	if _, execErr := tx.Exec(ctx, updateRunQuery, insertedCount, updatedSourcesJSON, updatedCompaniesJSON, req.RunID); execErr != nil {
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

// insertSingleJob performs all database operations for one job within an existing transaction,
// returning true if the job was newly inserted (not a duplicate) and any error that should
// trigger a savepoint rollback.
func insertSingleJob(ctx context.Context, tx pgx.Tx, job IngestJobPayload, companyCache map[string]string) (bool, error) {
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
				return false, fmt.Errorf("upsert company %q: %w", companyName, scanErr)
			}
		}
		companyCache[cacheKey] = companyID
	}

	if inferredDomain := utils.ExtractCompanyDomain(job.AbsoluteURL); inferredDomain != "" {
		updateDomainQuery := `UPDATE companies SET domain = $1 WHERE id = $2 AND (domain IS NULL OR domain = '')`
		_, _ = tx.Exec(ctx, updateDomainQuery, inferredDomain, companyID)
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
	if execErr != nil && execErr.Error() != "no rows in result set" {
		return false, fmt.Errorf("upsert job %q: %w", job.AbsoluteURL, execErr)
	}
	return insertedID != "", nil
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

	var err error
	var runSourcesMap map[string]utils.ScraperPlatformMetric
	if len(req.SourcesHit) > 0 && string(req.SourcesHit) != "null" {
		aggregatedSourcesPayload, aggregationError := utils.AggregateSourceStatistics(req.SourcesHit)
		if aggregationError == nil {
			_ = json.Unmarshal(aggregatedSourcesPayload, &runSourcesMap)
		}
	}
	if runSourcesMap == nil {
		runSourcesMap = make(map[string]utils.ScraperPlatformMetric)
	}

	var existingSourcesJSON, existingCompaniesJSON []byte
	_ = h.DB.QueryRow(context.Background(), `
		SELECT COALESCE(sources_hit, '{}'::jsonb), COALESCE(companies_hit, '[]'::jsonb)
		FROM scraper_runs WHERE id = $1
	`, req.RunID).Scan(&existingSourcesJSON, &existingCompaniesJSON)

	if len(existingSourcesJSON) > 0 {
		var existingMap map[string]utils.ScraperPlatformMetric
		if unmarshalErr := json.Unmarshal(existingSourcesJSON, &existingMap); unmarshalErr == nil {
			for src, metric := range existingMap {
				m := runSourcesMap[src]
				if m.JobsAdded == 0 {
					m.JobsAdded = metric.JobsAdded
				}
				if m.JobsFound == 0 {
					m.JobsFound = metric.JobsFound
				}
				runSourcesMap[src] = m
			}
		}
	}

	sourceCountsRows, srcErr := h.DB.Query(context.Background(), `
		SELECT LOWER(TRIM(source)), count(*)
		FROM jobs
		WHERE scraped_at >= (SELECT started_at - INTERVAL '2 minute' FROM scraper_runs WHERE id = $1)
		GROUP BY LOWER(TRIM(source));
	`, req.RunID)
	if srcErr == nil {
		for sourceCountsRows.Next() {
			var srcName string
			var count int
			if scanErr := sourceCountsRows.Scan(&srcName, &count); scanErr == nil && srcName != "" {
				metric := runSourcesMap[srcName]
				metric.JobsAdded = count
				if metric.JobsFound == 0 {
					metric.JobsFound = count
				}
				if metric.QueryCount == 0 {
					metric.QueryCount = 1
				}
				runSourcesMap[srcName] = metric
			}
		}
		sourceCountsRows.Close()
	}

	type companyCountItem struct {
		CompanyName string `json:"company_name"`
		JobsAdded   int    `json:"jobs_added"`
	}
	var topCompanyList []companyCountItem
	compRows, compErr := h.DB.Query(context.Background(), `
		SELECT c.name, count(*)
		FROM jobs j
		JOIN companies c ON j.company_id = c.id
		WHERE j.scraped_at >= (SELECT started_at - INTERVAL '2 minute' FROM scraper_runs WHERE id = $1)
		GROUP BY c.name
		ORDER BY count(*) DESC
		LIMIT 30;
	`, req.RunID)
	if compErr == nil {
		for compRows.Next() {
			var compName string
			var count int
			if scanErr := compRows.Scan(&compName, &count); scanErr == nil && strings.TrimSpace(compName) != "" {
				topCompanyList = append(topCompanyList, companyCountItem{
					CompanyName: compName,
					JobsAdded:   count,
				})
			}
		}
		compRows.Close()
	}
	if len(topCompanyList) == 0 && len(existingCompaniesJSON) > 0 {
		_ = json.Unmarshal(existingCompaniesJSON, &topCompanyList)
	}

	finalSourcesJSON, _ := json.Marshal(runSourcesMap)
	finalCompaniesJSON, _ := json.Marshal(topCompanyList)

	query := `
		UPDATE scraper_runs
		SET status = $1,
		    finished_at = CURRENT_TIMESTAMP,
		    error_message = $2,
		    sources_hit = $3,
		    companies_hit = $4
		WHERE id = $5;
	`
	_, err = h.DB.Exec(context.Background(), query, statusClean, req.ErrorMessage, finalSourcesJSON, finalCompaniesJSON, req.RunID)
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

// GetATSSlugs retrieves all active ATS platform and slug configurations.
func (handler *IngestHandler) GetATSSlugs(contextInstance *gin.Context) {
	rows, queryError := handler.DB.Query(
		context.Background(),
		`SELECT platform, slug FROM company_ats_boards WHERE is_active = true ORDER BY platform, slug`,
	)
	if queryError != nil {
		contextInstance.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query ATS slugs"})
		return
	}
	defer rows.Close()

	platformSlugMap := make(map[string][]string)
	for rows.Next() {
		var platformName string
		var companySlug string
		if scanError := rows.Scan(&platformName, &companySlug); scanError != nil {
			continue
		}
		platformSlugMap[platformName] = append(platformSlugMap[platformName], companySlug)
	}

	contextInstance.JSON(http.StatusOK, gin.H{"data": platformSlugMap})
}

// RegisterATSSlug registers or reactivates an ATS board slug.
func (handler *IngestHandler) RegisterATSSlug(contextInstance *gin.Context) {
	var requestPayload struct {
		Platform string `json:"platform" binding:"required"`
		Slug     string `json:"slug" binding:"required"`
	}
	if bindError := contextInstance.ShouldBindJSON(&requestPayload); bindError != nil {
		contextInstance.JSON(http.StatusBadRequest, gin.H{"error": "platform and slug are required"})
		return
	}

	_, executionError := handler.DB.Exec(
		context.Background(),
		`INSERT INTO company_ats_boards (platform, slug)
		 VALUES ($1, $2)
		 ON CONFLICT (platform, slug) DO UPDATE
		     SET is_active = true, last_seen_at = CURRENT_TIMESTAMP`,
		requestPayload.Platform,
		requestPayload.Slug,
	)
	if executionError != nil {
		contextInstance.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register ATS slug"})
		return
	}

	contextInstance.JSON(http.StatusOK, gin.H{"message": "ATS slug registered successfully"})
}

// CompanyProbeTarget represents a company entry with optional domain for career page probing.
type CompanyProbeTarget struct {
	Name   string `json:"name"`
	Domain string `json:"domain,omitempty"`
}

// GetAllCompanyNames retrieves distinct unmapped companies from the database for career page probing.
func (handler *IngestHandler) GetAllCompanyNames(contextInstance *gin.Context) {
	rows, queryError := handler.DB.Query(
		context.Background(),
		`SELECT c.name, COALESCE(c.domain, '')
		 FROM companies c
		 WHERE c.name != ''
		   AND NOT EXISTS (
		       SELECT 1 FROM company_ats_boards b
		       WHERE b.company_id = c.id OR LOWER(b.slug) = LOWER(c.name)
		   )
		 ORDER BY (c.domain IS NOT NULL AND c.domain != '') DESC, c.name`,
	)
	if queryError != nil {
		contextInstance.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query companies"})
		return
	}
	defer rows.Close()

	var targetsList []CompanyProbeTarget
	for rows.Next() {
		var target CompanyProbeTarget
		if scanError := rows.Scan(&target.Name, &target.Domain); scanError == nil && target.Name != "" {
			targetsList = append(targetsList, target)
		}
	}

	contextInstance.JSON(http.StatusOK, gin.H{"data": targetsList, "count": len(targetsList)})
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

type JobWithoutDescriptionItem struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
}

type EnrichJobDescriptionItem struct {
	ID              string `json:"id" binding:"required"`
	DescriptionText string `json:"description_text" binding:"required"`
}

type EnrichJobDescriptionsRequest struct {
	Updates []EnrichJobDescriptionItem `json:"updates" binding:"required"`
}

func (h *IngestHandler) GetJobsWithoutDescription(c *gin.Context) {
	source := c.DefaultQuery("source", "linkedin")
	limitStr := c.DefaultQuery("limit", "100")
	limit := 100
	if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 200 {
		limit = parsedLimit
	}

	sinceMinutesStr := c.DefaultQuery("since_minutes", "120")
	sinceMinutes := 120
	if parsedSinceMinutes, err := strconv.Atoi(sinceMinutesStr); err == nil && parsedSinceMinutes > 0 {
		sinceMinutes = parsedSinceMinutes
	}

	query := `
		SELECT id, url, title
		FROM jobs
		WHERE source = $1 
		  AND (raw_desc IS NULL OR raw_desc = '')
		  AND scraped_at >= NOW() - ($3 || ' minutes')::interval
		ORDER BY scraped_at DESC
		LIMIT $2;
	`
	rows, err := h.DB.Query(c.Request.Context(), query, source, limit, strconv.Itoa(sinceMinutes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pending description jobs"})
		return
	}
	defer rows.Close()

	var jobs []JobWithoutDescriptionItem
	for rows.Next() {
		var item JobWithoutDescriptionItem
		if scanErr := rows.Scan(&item.ID, &item.URL, &item.Title); scanErr == nil {
			jobs = append(jobs, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func (h *IngestHandler) EnrichJobDescriptions(c *gin.Context) {
	var req EnrichJobDescriptionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	updatedCount := 0
	for _, updateItem := range req.Updates {
		if updateItem.ID == "" || updateItem.DescriptionText == "" {
			continue
		}
		query := `UPDATE jobs SET raw_desc = $1 WHERE id = $2;`
		if _, execErr := h.DB.Exec(ctx, query, updateItem.DescriptionText, updateItem.ID); execErr == nil {
			updatedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Descriptions updated successfully",
		"updated_count": updatedCount,
	})
}
