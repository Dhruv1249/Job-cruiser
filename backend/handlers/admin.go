package handlers

import (
	"context"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
AdminHandler provides handlers for Master Admin control.
*/
type AdminHandler struct {
	DB           *pgxpool.Pool
	MatchService services.BatchMatchEvaluator
}

/*
EnsureMasterAdmin verifies that the authenticated user is a master admin.
*/
func (h *AdminHandler) EnsureMasterAdmin(c *gin.Context) bool {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return false
	}

	var isMasterAdmin bool
	query := `SELECT COALESCE(is_master_admin, false) FROM users WHERE id = $1;`
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(&isMasterAdmin)
	if err != nil || !isMasterAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Action requires Master Admin permissions"})
		return false
	}

	return true
}

/*
GetWhitelistedEmails retrieves all whitelisted email addresses.
*/
func (h *AdminHandler) GetWhitelistedEmails(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	query := `SELECT id, email, COALESCE(notes, ''), created_at::text FROM whitelisted_emails ORDER BY created_at DESC;`
	rows, err := h.DB.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch whitelisted emails"})
		return
	}
	defer rows.Close()

	var emails []gin.H
	for rows.Next() {
		var id, email, notes, createdAt string
		if err := rows.Scan(&id, &email, &notes, &createdAt); err != nil {
			continue
		}
		emails = append(emails, gin.H{
			"id":         id,
			"email":      email,
			"notes":      notes,
			"created_at": createdAt,
		})
	}

	if emails == nil {
		emails = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"data": emails})
}

type AddWhitelistRequest struct {
	Email string `json:"email" binding:"required,email"`
	Notes string `json:"notes"`
}

/*
AddWhitelistedEmail adds a new email to the access whitelist.
*/
func (h *AdminHandler) AddWhitelistedEmail(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	var req AddWhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `
		INSERT INTO whitelisted_emails (email, notes)
		VALUES (LOWER($1), $2)
		ON CONFLICT (email) DO NOTHING
		RETURNING id;
	`

	var id string
	err := h.DB.QueryRow(context.Background(), query, req.Email, req.Notes).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already whitelisted"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Email added to whitelist", "id": id})
}

/*
DeleteWhitelistedEmail removes an email from the whitelist.
*/
func (h *AdminHandler) DeleteWhitelistedEmail(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	id := c.Param("id")
	query := `DELETE FROM whitelisted_emails WHERE id = $1;`
	_, err := h.DB.Exec(context.Background(), query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Whitelisted email removed"})
}

/*
GetPendingKeywords retrieves AI-suggested keywords awaiting approval.
*/
func (h *AdminHandler) GetPendingKeywords(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	query := `
		SELECT id, keyword, status, created_at::text
		FROM pending_keyword_suggestions
		WHERE status = 'pending'
		ORDER BY created_at DESC;
	`

	rows, err := h.DB.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending keywords"})
		return
	}
	defer rows.Close()

	var keywords []gin.H
	for rows.Next() {
		var id, keyword, status, createdAt string
		if err := rows.Scan(&id, &keyword, &status, &createdAt); err != nil {
			continue
		}
		keywords = append(keywords, gin.H{
			"id":         id,
			"keyword":    keyword,
			"status":     status,
			"created_at": createdAt,
		})
	}

	if keywords == nil {
		keywords = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"data": keywords})
}

type ApproveKeywordRequest struct {
	SuggestionID string `json:"suggestion_id" binding:"required"`
	Approve      bool   `json:"approve"`
}

/*
ApproveKeyword processes a pending keyword suggestion.
*/
func (h *AdminHandler) ApproveKeyword(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	var req ApproveKeywordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.Approve {
		var keyword string
		err := h.DB.QueryRow(context.Background(), "SELECT keyword FROM pending_keyword_suggestions WHERE id = $1", req.SuggestionID).Scan(&keyword)
		if err == nil {
			h.DB.Exec(context.Background(), "INSERT INTO master_keywords (keyword) VALUES ($1) ON CONFLICT DO NOTHING", keyword)
		}
		h.DB.Exec(context.Background(), "UPDATE pending_keyword_suggestions SET status = 'approved' WHERE id = $1", req.SuggestionID)
	} else {
		h.DB.Exec(context.Background(), "UPDATE pending_keyword_suggestions SET status = 'rejected' WHERE id = $1", req.SuggestionID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Keyword recommendation updated"})
}

type AddMasterKeywordRequest struct {
	Keyword  string `json:"keyword" binding:"required"`
	Category string `json:"category"`
}

/*
GetMasterKeywordsForAdmin retrieves all current master keywords from the dictionary.
*/
func (h *AdminHandler) GetMasterKeywordsForAdmin(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	query := `
		SELECT id, keyword, category, created_at::text
		FROM master_keywords
		ORDER BY keyword ASC;
	`
	rows, err := h.DB.Query(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch master keywords"})
		return
	}
	defer rows.Close()

	var list []gin.H
	for rows.Next() {
		var id int
		var kw, category, createdAt string
		if err := rows.Scan(&id, &kw, &category, &createdAt); err == nil {
			list = append(list, gin.H{
				"id":         id,
				"keyword":    kw,
				"category":   category,
				"created_at": createdAt,
			})
		}
	}
	if list == nil {
		list = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

/*
AddMasterKeyword manually inserts a new keyword into the master dictionary.
*/
func (h *AdminHandler) AddMasterKeyword(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	var req AddMasterKeywordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Keyword required"})
		return
	}

	cleanKeyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	if cleanKeyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Keyword cannot be empty"})
		return
	}

	cat := strings.TrimSpace(req.Category)
	if cat == "" {
		cat = "scraper"
	}

	var existingID int
	checkErr := h.DB.QueryRow(c.Request.Context(), "SELECT id FROM master_keywords WHERE LOWER(keyword) = $1 LIMIT 1;", cleanKeyword).Scan(&existingID)
	if checkErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Keyword already exists in master dictionary"})
		return
	}

	query := `
		INSERT INTO master_keywords (keyword, category)
		VALUES ($1, $2)
		ON CONFLICT (keyword) DO NOTHING
		RETURNING id;
	`
	var id int
	err := h.DB.QueryRow(c.Request.Context(), query, cleanKeyword, cat).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Keyword already exists in master dictionary"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Master keyword added successfully",
		"id":      id,
		"keyword": cleanKeyword,
	})
}

/*
DeleteMasterKeyword removes a keyword from the master dictionary by ID.
*/
func (h *AdminHandler) DeleteMasterKeyword(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	id := c.Param("id")
	query := `DELETE FROM master_keywords WHERE id = $1;`
	_, err := h.DB.Exec(c.Request.Context(), query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete keyword"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Master keyword deleted"})
}

type ToggleAIMatchingRequest struct {
	Enabled bool `json:"enabled"`
}

/*
ToggleUserAIMatching enables or disables AI matching evaluation for a specified user.
*/
func (h *AdminHandler) ToggleUserAIMatching(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	targetUserID := c.Param("id")
	var req ToggleAIMatchingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `UPDATE users SET ai_matching_enabled = $1 WHERE id = $2;`
	_, err := h.DB.Exec(context.Background(), query, req.Enabled, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update AI matching state"})
		return
	}

	if req.Enabled && h.MatchService != nil {
		go func() {
			log.Printf("[AdminHandler] AI matching enabled for user %s — triggering single-user evaluation.", targetUserID)
			h.MatchService.EvaluateForSingleUser(context.Background(), targetUserID)
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "User AI matching state updated"})
}

/*
GetUsers lists registered users with their AI matching status for Master Admin control.
*/
func (h *AdminHandler) GetUsers(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	query := `
		SELECT u.id, u.primary_email, COALESCE(up.full_name, ''), COALESCE(u.ai_matching_enabled, false), COALESCE(u.is_master_admin, false), u.created_at
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		ORDER BY u.created_at DESC;
	`

	rows, err := h.DB.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var usersList []gin.H
	for rows.Next() {
		var id, email, fullName string
		var aiMatchingEnabled, isMasterAdmin bool
		var createdAt time.Time
		if err := rows.Scan(&id, &email, &fullName, &aiMatchingEnabled, &isMasterAdmin, &createdAt); err != nil {
			continue
		}
		usersList = append(usersList, gin.H{
			"id":                  id,
			"primary_email":       email,
			"full_name":           fullName,
			"ai_matching_enabled": aiMatchingEnabled,
			"is_master_admin":     isMasterAdmin,
			"created_at":          createdAt.Format(time.RFC3339),
		})
	}

	if usersList == nil {
		usersList = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"data": usersList})
}

/*
ResetAndReevaluateMatches clears all user_job_matches and resets jobs.ai_evaluated to false,
then triggers a fresh batch evaluation pass across all active users.
*/
func (h *AdminHandler) ResetAndReevaluateMatches(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	ctx := context.Background()

	// 1. Clear old matches
	_, err := h.DB.Exec(ctx, "DELETE FROM user_job_matches;")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear user_job_matches: " + err.Error()})
		return
	}

	// 2. Reset ai_evaluated flag on jobs
	_, err = h.DB.Exec(ctx, "UPDATE jobs SET ai_evaluated = false;")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset jobs.ai_evaluated: " + err.Error()})
		return
	}

	// 3. Trigger fresh evaluation pass in background
	if h.MatchService != nil {
		go func() {
			log.Println("[AdminHandler] Reset completed. Triggering fresh multi-candidate evaluation pass.")
			h.MatchService.EvaluatePendingForAllUsers(context.Background())
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully cleared old matches and reset evaluation flag. Fresh AI evaluation pipeline triggered in background.",
	})
}

/*
ResetUserMatches clears user_job_matches for the calling user and re-evaluates all jobs for them.
*/
func (h *AdminHandler) ResetUserMatches(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	uidStr, ok := userID.(string)
	if !ok || uidStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx := context.Background()

	// Delete matches for this specific user
	_, err := h.DB.Exec(ctx, "DELETE FROM user_job_matches WHERE user_id = $1;", uidStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear matches for user: " + err.Error()})
		return
	}

	if h.MatchService != nil {
		go func() {
			log.Printf("[AdminHandler] Triggering re-evaluation for user %s", uidStr)
			h.MatchService.EvaluateForSingleUser(context.Background(), uidStr)
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User matches reset successfully. Re-evaluation pass triggered in background.",
	})
}

/*
GetScraperStats retrieves comprehensive scraper telemetry including source volume breakdown,
match quality by source, 14-day ingestion timeline, score tier distribution, and run health metrics.
*/
func (h *AdminHandler) GetScraperStats(c *gin.Context) {
	if !h.EnsureMasterAdmin(c) {
		return
	}

	ctx := context.Background()

	var totalJobs int
	_ = h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM jobs;`).Scan(&totalJobs)

	var jobsLast24h int
	_ = h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE scraped_at >= NOW() - INTERVAL '24 hours';`).Scan(&jobsLast24h)

	var jobsLast7d int
	_ = h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE scraped_at >= NOW() - INTERVAL '7 days';`).Scan(&jobsLast7d)

	var uniqueCompanies int
	_ = h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM companies;`).Scan(&uniqueCompanies)

	var remoteJobsCount int
	_ = h.DB.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE is_remote = true;`).Scan(&remoteJobsCount)

	var remoteJobsPct float64
	if totalJobs > 0 {
		remoteJobsPct = math.Round((float64(remoteJobsCount)/float64(totalJobs))*1000) / 10
	}

	sourcesVolumeRows, sourcesVolumeErr := h.DB.Query(ctx, `
		SELECT source,
		       COUNT(*) AS total_jobs,
		       COUNT(*) FILTER (WHERE scraped_at >= NOW() - INTERVAL '24 hours') AS jobs_24h,
		       COUNT(*) FILTER (WHERE scraped_at >= NOW() - INTERVAL '7 days') AS jobs_7d,
		       COUNT(*) FILTER (WHERE is_remote = true) AS remote_jobs,
		       COUNT(*) FILTER (WHERE is_remote = false) AS onsite_jobs
		FROM jobs
		GROUP BY source
		ORDER BY total_jobs DESC;
	`)

	var sourcesVolume []gin.H
	var topVolumeSource string
	if sourcesVolumeErr == nil {
		defer sourcesVolumeRows.Close()
		for sourcesVolumeRows.Next() {
			var sourceName string
			var sourceTotal, source24h, source7d, remoteCount, onsiteCount int
			if scanErr := sourcesVolumeRows.Scan(&sourceName, &sourceTotal, &source24h, &source7d, &remoteCount, &onsiteCount); scanErr == nil {
				if topVolumeSource == "" {
					topVolumeSource = sourceName
				}
				var sharePct float64
				if totalJobs > 0 {
					sharePct = math.Round((float64(sourceTotal)/float64(totalJobs))*1000) / 10
				}
				sourcesVolume = append(sourcesVolume, gin.H{
					"source":        sourceName,
					"total_jobs":    sourceTotal,
					"jobs_last_24h": source24h,
					"jobs_last_7d":  source7d,
					"remote_jobs":   remoteCount,
					"onsite_jobs":   onsiteCount,
					"share_pct":     sharePct,
				})
			}
		}
	}
	if sourcesVolume == nil {
		sourcesVolume = []gin.H{}
	}

	sourcesQualityRows, sourcesQualityErr := h.DB.Query(ctx, `
		SELECT j.source,
		       COUNT(m.job_id) AS evaluated_count,
		       COALESCE(ROUND(AVG(m.match_score), 1), 0) AS avg_score,
		       COUNT(*) FILTER (WHERE m.match_score >= 80) AS elite_matches,
		       COUNT(*) FILTER (WHERE m.match_score >= 60 AND m.match_score < 80) AS good_matches,
		       COUNT(*) FILTER (WHERE m.match_score < 60) AS low_matches
		FROM jobs j
		JOIN user_job_matches m ON j.id = m.job_id
		GROUP BY j.source
		ORDER BY avg_score DESC, elite_matches DESC;
	`)

	var sourcesQuality []gin.H
	var topQualitySource string
	if sourcesQualityErr == nil {
		defer sourcesQualityRows.Close()
		for sourcesQualityRows.Next() {
			var sourceName string
			var evaluatedCount, eliteMatches, goodMatches, lowMatches int
			var avgScore float64
			if scanErr := sourcesQualityRows.Scan(&sourceName, &evaluatedCount, &avgScore, &eliteMatches, &goodMatches, &lowMatches); scanErr == nil {
				if topQualitySource == "" && evaluatedCount >= 3 {
					topQualitySource = sourceName
				}
				var highMatchYieldPct float64
				if evaluatedCount > 0 {
					highMatchYieldPct = math.Round((float64(eliteMatches)/float64(evaluatedCount))*1000) / 10
				}
				sourcesQuality = append(sourcesQuality, gin.H{
					"source":               sourceName,
					"evaluated_count":      evaluatedCount,
					"avg_score":            avgScore,
					"elite_matches":        eliteMatches,
					"good_matches":         goodMatches,
					"low_matches":          lowMatches,
					"high_match_yield_pct": highMatchYieldPct,
				})
			}
		}
	}
	if sourcesQuality == nil {
		sourcesQuality = []gin.H{}
	}

	timelineRows, timelineErr := h.DB.Query(ctx, `
		SELECT TO_CHAR(scraped_at, 'YYYY-MM-DD') AS scrape_date,
		       COUNT(*) AS jobs_count
		FROM jobs
		WHERE scraped_at >= NOW() - INTERVAL '14 days'
		GROUP BY TO_CHAR(scraped_at, 'YYYY-MM-DD')
		ORDER BY scrape_date ASC;
	`)

	var ingestionTimeline []gin.H
	if timelineErr == nil {
		defer timelineRows.Close()
		for timelineRows.Next() {
			var scrapeDate string
			var jobsCount int
			if scanErr := timelineRows.Scan(&scrapeDate, &jobsCount); scanErr == nil {
				ingestionTimeline = append(ingestionTimeline, gin.H{
					"date":       scrapeDate,
					"jobs_count": jobsCount,
				})
			}
		}
	}
	if ingestionTimeline == nil {
		ingestionTimeline = []gin.H{}
	}

	var tier90To100, tier80To89, tier60To79, tierBelow60 int
	var overallAvgScore float64
	_ = h.DB.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE match_score >= 90) AS tier_90_100,
		       COUNT(*) FILTER (WHERE match_score >= 80 AND match_score < 90) AS tier_80_89,
		       COUNT(*) FILTER (WHERE match_score >= 60 AND match_score < 80) AS tier_60_79,
		       COUNT(*) FILTER (WHERE match_score < 60) AS tier_below_60,
		       COALESCE(ROUND(AVG(match_score), 1), 0) AS avg_score
		FROM user_job_matches;
	`).Scan(&tier90To100, &tier80To89, &tier60To79, &tierBelow60, &overallAvgScore)

	evaluatedJobsCount := tier90To100 + tier80To89 + tier60To79 + tierBelow60
	unevaluatedCount := totalJobs - evaluatedJobsCount
	if unevaluatedCount < 0 {
		unevaluatedCount = 0
	}

	var evaluationCoveragePct float64
	if totalJobs > 0 {
		evaluationCoveragePct = math.Round((float64(evaluatedJobsCount)/float64(totalJobs))*1000) / 10
	}

	scoreDistribution := gin.H{
		"tier_90_100":       tier90To100,
		"tier_80_89":        tier80To89,
		"tier_60_79":        tier60To79,
		"tier_below_60":     tierBelow60,
		"unevaluated_count": unevaluatedCount,
		"avg_score":         overallAvgScore,
	}

	topCompaniesRows, topCompaniesErr := h.DB.Query(ctx, `
		SELECT c.name, COUNT(j.id) AS job_count
		FROM companies c
		JOIN jobs j ON c.id = j.company_id
		GROUP BY c.id, c.name
		ORDER BY job_count DESC
		LIMIT 10;
	`)

	var topCompanies []gin.H
	if topCompaniesErr == nil {
		defer topCompaniesRows.Close()
		for topCompaniesRows.Next() {
			var companyName string
			var companyJobCount int
			if scanErr := topCompaniesRows.Scan(&companyName, &companyJobCount); scanErr == nil {
				topCompanies = append(topCompanies, gin.H{
					"company_name": companyName,
					"job_count":    companyJobCount,
				})
			}
		}
	}
	if topCompanies == nil {
		topCompanies = []gin.H{}
	}

	runsRows, runsErr := h.DB.Query(ctx, `
		SELECT id, started_at, COALESCE(finished_at, started_at), status, jobs_added,
		       COALESCE(sources_hit::text, '[]'),
		       COALESCE(error_message, ''),
		       EXTRACT(EPOCH FROM (COALESCE(finished_at, started_at) - started_at))::INT AS duration_seconds
		FROM scraper_runs
		ORDER BY started_at DESC
		LIMIT 50;
	`)

	var runs []gin.H
	var successfulRunsCount, failedRunsCount int
	var totalDurationSeconds int
	if runsErr == nil {
		defer runsRows.Close()
		for runsRows.Next() {
			var runID, status, sourcesRaw, errorMessage string
			var startedAt, finishedAt time.Time
			var jobsAdded, durationSeconds int
			if scanErr := runsRows.Scan(&runID, &startedAt, &finishedAt, &status, &jobsAdded, &sourcesRaw, &errorMessage, &durationSeconds); scanErr == nil {
				if status == "completed" {
					successfulRunsCount++
				} else if status == "failed" {
					failedRunsCount++
				}
				totalDurationSeconds += durationSeconds

				runs = append(runs, gin.H{
					"run_id":           runID,
					"started_at":       startedAt.Format(time.RFC3339),
					"finished_at":      finishedAt.Format(time.RFC3339),
					"status":           status,
					"jobs_added":       jobsAdded,
					"sources_hit":      sourcesRaw,
					"error_message":    errorMessage,
					"duration_seconds": durationSeconds,
				})
			}
		}
	}
	if runs == nil {
		runs = []gin.H{}
	}

	totalRunsRecorded := len(runs)
	var successRatePct float64
	var avgDurationSeconds int
	if totalRunsRecorded > 0 {
		successRatePct = math.Round((float64(successfulRunsCount)/float64(totalRunsRecorded))*1000) / 10
		avgDurationSeconds = totalDurationSeconds / totalRunsRecorded
	}

	runHealth := gin.H{
		"total_runs_recorded":  totalRunsRecorded,
		"successful_runs":      successfulRunsCount,
		"failed_runs":          failedRunsCount,
		"success_rate_pct":     successRatePct,
		"avg_duration_seconds": avgDurationSeconds,
	}

	kpis := gin.H{
		"total_jobs":              totalJobs,
		"jobs_last_24h":           jobsLast24h,
		"jobs_last_7d":            jobsLast7d,
		"unique_companies":        uniqueCompanies,
		"evaluated_jobs_count":    evaluatedJobsCount,
		"evaluation_coverage_pct": evaluationCoveragePct,
		"overall_avg_match_score": overallAvgScore,
		"remote_jobs_count":       remoteJobsCount,
		"remote_jobs_pct":         remoteJobsPct,
		"top_volume_source":       topVolumeSource,
		"top_quality_source":      topQualitySource,
	}

	c.JSON(http.StatusOK, gin.H{
		"total_jobs":         totalJobs,
		"jobs_last_24h":      jobsLast24h,
		"jobs_last_7d":       jobsLast7d,
		"unique_companies":   uniqueCompanies,
		"kpis":               kpis,
		"sources_volume":     sourcesVolume,
		"sources_quality":    sourcesQuality,
		"ingestion_timeline": ingestionTimeline,
		"score_distribution": scoreDistribution,
		"run_health":         runHealth,
		"top_companies":      topCompanies,
		"runs":               runs,
	})
}
