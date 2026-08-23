package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TailorHandler manages HTTP REST requests for AI resume and cover letter tailoring.
type TailorHandler struct {
	TailorService *services.ResumeTailorService
	DB            *pgxpool.Pool
	AESKey        []byte
	MCPSecret     string
}

// TailorRequestPayload defines incoming JSON parameters for tailoring operations.
type TailorRequestPayload struct {
	JobID                  string `json:"job_id" binding:"required"`
	TargetPages            int    `json:"target_pages"`
	TargetResumePages      int    `json:"target_resume_pages"`
	TargetCoverLetterPages int    `json:"target_cover_letter_pages"`
}

// NewTailorHandler initializes a TailorHandler with all required dependencies.
func NewTailorHandler(
	tailorService *services.ResumeTailorService,
	db *pgxpool.Pool,
	aesKey []byte,
	mcpSecret string,
) *TailorHandler {
	return &TailorHandler{
		TailorService: tailorService,
		DB:            db,
		AESKey:        aesKey,
		MCPSecret:     mcpSecret,
	}
}

type jobTailoringRecord struct {
	Title     string
	Company   string
	Seniority string
	TechStack []string
	RawDesc   string
}

func (handler *TailorHandler) fetchJobTailoringRecord(ctx *gin.Context, jobID string) (*jobTailoringRecord, error) {
	var title, company, seniority, rawDesc string
	var techStackJSON []byte

	queryError := handler.DB.QueryRow(
		ctx.Request.Context(),
		`SELECT j.title, COALESCE(c.name, ''), COALESCE(j.seniority, ''), COALESCE(j.raw_desc, ''), COALESCE(j.tags, '[]'::jsonb)
		 FROM jobs j
		 LEFT JOIN companies c ON c.id = j.company_id
		 WHERE j.id = $1`,
		jobID,
	).Scan(&title, &company, &seniority, &rawDesc, &techStackJSON)

	if queryError != nil {
		return nil, fmt.Errorf("job not found: %w", queryError)
	}

	var techStack []string
	if unmarshalError := json.Unmarshal(techStackJSON, &techStack); unmarshalError != nil {
		techStack = []string{}
	}

	return &jobTailoringRecord{
		Title:     title,
		Company:   company,
		Seniority: seniority,
		TechStack: techStack,
		RawDesc:   rawDesc,
	}, nil
}

func (handler *TailorHandler) fetchUserBio(ctx *gin.Context, userID interface{}) string {
	var bioText string
	queryError := handler.DB.QueryRow(
		ctx.Request.Context(),
		`SELECT COALESCE(NULLIF(bio_experience_text, ''), master_cv_text, '') FROM user_preferences WHERE user_id = $1`,
		userID,
	).Scan(&bioText)
	if queryError != nil || bioText == "" {
		return "Experienced software engineer with strong backend and cloud infrastructure skills."
	}
	return bioText
}

func (handler *TailorHandler) fetchUserTargetPages(ctx *gin.Context, userID interface{}) (int, int) {
	var targetResumePages, targetCoverLetterPages int
	queryError := handler.DB.QueryRow(
		ctx.Request.Context(),
		`SELECT COALESCE(target_resume_pages, 1), COALESCE(target_cover_letter_pages, 1) FROM user_preferences WHERE user_id = $1`,
		userID,
	).Scan(&targetResumePages, &targetCoverLetterPages)
	if queryError != nil {
		return 1, 1
	}
	if targetResumePages <= 0 {
		targetResumePages = 1
	}
	if targetCoverLetterPages <= 0 {
		targetCoverLetterPages = 1
	}
	return targetResumePages, targetCoverLetterPages
}

func (handler *TailorHandler) buildPDFWebURL(credentials *services.UserOverleafCredentials) string {
	projectName := credentials.ProjectName
	if projectName == "" {
		projectName = "job_applications"
	}
	return fmt.Sprintf("%s/?project=%s", strings.TrimRight(credentials.DeploymentURL, "/"), projectName)
}

// TailorResume generates a tailored resume using the job context and user bio via Gemini,
// compiles it in the user's open-overleaf instance, saves a reference to resume_versions, and returns
// the compiled PDF as base64 along with the open-overleaf IDE URL for human review.
func (handler *TailorHandler) TailorResume(ginContext *gin.Context) {
	var payload TailorRequestPayload
	if bindError := ginContext.ShouldBindJSON(&payload); bindError != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + bindError.Error()})
		return
	}

	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	jobRecord, jobError := handler.fetchJobTailoringRecord(ginContext, payload.JobID)
	if jobError != nil {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	userBio := handler.fetchUserBio(ginContext, userIDValue)

	mcpClient, credentials, mcpError := services.LoadUserMCPClient(
		ginContext.Request.Context(),
		handler.DB,
		userID,
		handler.AESKey,
		handler.MCPSecret,
	)
	if mcpError != nil {
		ginContext.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":        "Open-Overleaf is not configured. Please set your Open-Overleaf Server URL in Preferences.",
			"unconfigured": true,
		})
		return
	}

	targetPages := payload.TargetPages
	if targetPages <= 0 {
		targetPages = payload.TargetResumePages
	}
	if targetPages <= 0 {
		prefResumePages, _ := handler.fetchUserTargetPages(ginContext, userIDValue)
		targetPages = prefResumePages
	}

	projectName := credentials.ProjectName
	if projectName == "" {
		projectName = "job_applications"
	}

	folderPath := services.BuildJobFolderPath(jobRecord.Company, jobRecord.Title)
	jobContext := services.JobTailoringContext{
		Title:     jobRecord.Title,
		Company:   jobRecord.Company,
		Seniority: jobRecord.Seniority,
		TechStack: jobRecord.TechStack,
		RawDesc:   jobRecord.RawDesc,
	}

	result, tailorError := handler.TailorService.TailorResumeToFolder(
		ginContext.Request.Context(),
		mcpClient,
		userBio,
		jobContext,
		folderPath,
		projectName,
		targetPages,
	)
	if tailorError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": tailorError.Error()})
		return
	}

	result.PDFWebURL = handler.buildPDFWebURL(credentials)

	versionLabel := fmt.Sprintf("%s — %s (%s)", jobRecord.Company, jobRecord.Title, time.Now().UTC().Format("2006-01-02"))
	var savedVersionID string
	insertError := handler.DB.QueryRow(
		ginContext.Request.Context(),
		`INSERT INTO resume_versions (user_id, job_id, label, overleaf_project_name, overleaf_folder_path, pdf_url, page_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		userID, payload.JobID, versionLabel, projectName, folderPath, result.PDFWebURL, result.CompileResult.PageCount,
	).Scan(&savedVersionID)

	if insertError != nil {
		savedVersionID = ""
	}

	pdfBase64 := ""
	if result.PDFResult != nil {
		pdfBase64 = result.PDFResult.Base64Data
	}

	ginContext.JSON(http.StatusOK, gin.H{
		"message":        "Resume tailored and compiled successfully",
		"version_id":     savedVersionID,
		"folder_path":    result.FolderPath,
		"file_path":      result.FilePath,
		"pdf_web_url":    result.PDFWebURL,
		"compile_result": result.CompileResult,
		"pdf_base64":     pdfBase64,
		"page_count":     result.CompileResult.PageCount,
	})
}

// GenerateCoverLetter generates a personalized cover letter using the job context and user bio via Gemini,
// compiles it in the user's open-overleaf instance, saves a reference to cover_letter_versions,
// and returns the compiled PDF as base64 along with the open-overleaf IDE URL.
func (handler *TailorHandler) GenerateCoverLetter(ginContext *gin.Context) {
	var payload TailorRequestPayload
	if bindError := ginContext.ShouldBindJSON(&payload); bindError != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + bindError.Error()})
		return
	}

	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	jobRecord, jobError := handler.fetchJobTailoringRecord(ginContext, payload.JobID)
	if jobError != nil {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	userBio := handler.fetchUserBio(ginContext, userIDValue)

	mcpClient, credentials, mcpError := services.LoadUserMCPClient(
		ginContext.Request.Context(),
		handler.DB,
		userID,
		handler.AESKey,
		handler.MCPSecret,
	)
	if mcpError != nil {
		ginContext.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":        "Open-Overleaf is not configured. Please set your Open-Overleaf Server URL in Preferences.",
			"unconfigured": true,
		})
		return
	}

	targetPages := payload.TargetPages
	if targetPages <= 0 {
		targetPages = payload.TargetCoverLetterPages
	}
	if targetPages <= 0 {
		_, prefCoverPages := handler.fetchUserTargetPages(ginContext, userIDValue)
		targetPages = prefCoverPages
	}

	projectName := credentials.ProjectName
	if projectName == "" {
		projectName = "job_applications"
	}

	folderPath := services.BuildJobFolderPath(jobRecord.Company, jobRecord.Title)
	jobContext := services.JobTailoringContext{
		Title:     jobRecord.Title,
		Company:   jobRecord.Company,
		Seniority: jobRecord.Seniority,
		TechStack: jobRecord.TechStack,
		RawDesc:   jobRecord.RawDesc,
	}

	result, generateError := handler.TailorService.GenerateCoverLetterToFolder(
		ginContext.Request.Context(),
		mcpClient,
		userBio,
		jobContext,
		folderPath,
		projectName,
		targetPages,
	)
	if generateError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": generateError.Error()})
		return
	}

	result.PDFWebURL = handler.buildPDFWebURL(credentials)

	versionLabel := fmt.Sprintf("%s — %s (Cover Letter %s)", jobRecord.Company, jobRecord.Title, time.Now().UTC().Format("2006-01-02"))
	var savedCoverLetterID string
	insertError := handler.DB.QueryRow(
		ginContext.Request.Context(),
		`INSERT INTO cover_letter_versions (user_id, job_id, label, overleaf_project_name, overleaf_folder_path, pdf_url, page_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		userID, payload.JobID, versionLabel, projectName, folderPath, result.PDFWebURL, result.CompileResult.PageCount,
	).Scan(&savedCoverLetterID)

	if insertError != nil {
		savedCoverLetterID = ""
	}

	pdfBase64 := ""
	if result.PDFResult != nil {
		pdfBase64 = result.PDFResult.Base64Data
	}

	ginContext.JSON(http.StatusOK, gin.H{
		"message":         "Cover letter generated and compiled successfully",
		"cover_letter_id": savedCoverLetterID,
		"folder_path":     result.FolderPath,
		"file_path":       result.FilePath,
		"pdf_web_url":     result.PDFWebURL,
		"compile_result":  result.CompileResult,
		"pdf_base64":      pdfBase64,
		"page_count":      result.CompileResult.PageCount,
	})
}

/*
TailorApplicationAsync starts a non-blocking background job to tailor both Resume and Cover Letter for a job,
saving the compiled versions to open-overleaf and PostgreSQL and creating a user notification upon completion.
*/
func (handler *TailorHandler) TailorApplicationAsync(ginContext *gin.Context) {
	var payload TailorRequestPayload
	if bindError := ginContext.ShouldBindJSON(&payload); bindError != nil {
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + bindError.Error()})
		return
	}

	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := fmt.Sprintf("%v", userIDValue)

	if handler.DB == nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Database unavailable"})
		return
	}

	jobRecord, jobError := handler.fetchJobTailoringRecord(ginContext, payload.JobID)
	if jobError != nil {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	_, _, mcpError := services.LoadUserMCPClient(
		ginContext.Request.Context(),
		handler.DB,
		userID,
		handler.AESKey,
		handler.MCPSecret,
	)
	if mcpError != nil {
		ginContext.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":        "Open-Overleaf is not configured. Please set your Open-Overleaf Server URL in Preferences.",
			"unconfigured": true,
		})
		return
	}

	userBio := handler.fetchUserBio(ginContext, userIDValue)
	prefResumePages, prefCoverPages := handler.fetchUserTargetPages(ginContext, userIDValue)

	targetResumePages := payload.TargetResumePages
	if targetResumePages <= 0 {
		targetResumePages = payload.TargetPages
	}
	if targetResumePages <= 0 {
		targetResumePages = prefResumePages
	}

	targetCoverLetterPages := payload.TargetCoverLetterPages
	if targetCoverLetterPages <= 0 {
		targetCoverLetterPages = prefCoverPages
	}

	folderPath := services.BuildJobFolderPath(jobRecord.Company, jobRecord.Title)
	resumeLabel := fmt.Sprintf("%s — %s (%s)", jobRecord.Company, jobRecord.Title, time.Now().UTC().Format("2006-01-02"))
	coverLabel := fmt.Sprintf("%s — %s (Cover Letter %s)", jobRecord.Company, jobRecord.Title, time.Now().UTC().Format("2006-01-02"))

	var resumeVersionID, coverVersionID string
	_ = handler.DB.QueryRow(
		ginContext.Request.Context(),
		`INSERT INTO resume_versions (user_id, job_id, label, overleaf_project_name, overleaf_folder_path, page_count, status)
		 VALUES ($1, $2, $3, 'job_applications', $4, $5, 'generating')
		 RETURNING id`,
		userID, payload.JobID, resumeLabel, folderPath, targetResumePages,
	).Scan(&resumeVersionID)

	_ = handler.DB.QueryRow(
		ginContext.Request.Context(),
		`INSERT INTO cover_letter_versions (user_id, job_id, label, overleaf_project_name, overleaf_folder_path, page_count, status)
		 VALUES ($1, $2, $3, 'job_applications', $4, $5, 'generating')
		 RETURNING id`,
		userID, payload.JobID, coverLabel, folderPath, targetCoverLetterPages,
	).Scan(&coverVersionID)

	go func(backgroundCtx context.Context, uID string, jID string, jRecord *jobTailoringRecord, bio string, resPages int, covPages int, resVerID string, covVerID string, targetFolder string) {
		log.Printf("[TailorAsync] Starting generation for %s at %s (userID: %s)...", jRecord.Title, jRecord.Company, uID)

		client, credentials, clientError := services.LoadUserMCPClient(
			backgroundCtx,
			handler.DB,
			uID,
			handler.AESKey,
			handler.MCPSecret,
		)
		if clientError != nil {
			log.Printf("[TailorAsync] Open-Overleaf client error for %s: %v", jRecord.Company, clientError)
			if resVerID != "" {
				_, _ = handler.DB.Exec(backgroundCtx, `UPDATE resume_versions SET status = 'failed', error_message = $1 WHERE id = $2`, clientError.Error(), resVerID)
			}
			if covVerID != "" {
				_, _ = handler.DB.Exec(backgroundCtx, `UPDATE cover_letter_versions SET status = 'failed', error_message = $1 WHERE id = $2`, clientError.Error(), covVerID)
			}
			failTitle := fmt.Sprintf("Tailoring Failed: %s", jRecord.Company)
			failMessage := fmt.Sprintf("Failed connecting to Open-Overleaf for %s: %s", jRecord.Title, clientError.Error())
			_, _ = handler.DB.Exec(backgroundCtx, `INSERT INTO notifications (user_id, title, message, is_read) VALUES ($1, $2, $3, false)`, uID, failTitle, failMessage)
			return
		}

		projectName := credentials.ProjectName
		if projectName == "" {
			projectName = "job_applications"
		}
		jobContext := services.JobTailoringContext{
			Title:     jRecord.Title,
			Company:   jRecord.Company,
			Seniority: jRecord.Seniority,
			TechStack: jRecord.TechStack,
			RawDesc:   jRecord.RawDesc,
		}

		resumeResult, resumeError := handler.TailorService.TailorResumeToFolder(
			backgroundCtx,
			client,
			bio,
			jobContext,
			targetFolder,
			projectName,
			resPages,
		)
		if resumeError != nil {
			log.Printf("[TailorAsync] Resume generation error for %s: %v", jRecord.Company, resumeError)
			if resVerID != "" {
				_, _ = handler.DB.Exec(backgroundCtx, `UPDATE resume_versions SET status = 'failed', error_message = $1 WHERE id = $2`, resumeError.Error(), resVerID)
			}
			failTitle := fmt.Sprintf("Tailoring Failed: %s", jRecord.Company)
			failMessage := fmt.Sprintf("Failed generating resume for %s: %s", jRecord.Title, resumeError.Error())
			_, _ = handler.DB.Exec(backgroundCtx, `INSERT INTO notifications (user_id, title, message, is_read) VALUES ($1, $2, $3, false)`, uID, failTitle, failMessage)
			return
		}

		resumeWebURL := handler.buildPDFWebURL(credentials)
		if resVerID != "" {
			_, _ = handler.DB.Exec(
				backgroundCtx,
				`UPDATE resume_versions
				 SET status = 'ready', overleaf_project_name = $1, pdf_url = $2, page_count = $3
				 WHERE id = $4`,
				projectName, resumeWebURL, resumeResult.CompileResult.PageCount, resVerID,
			)
		}

		coverResult, coverError := handler.TailorService.GenerateCoverLetterToFolder(
			backgroundCtx,
			client,
			bio,
			jobContext,
			targetFolder,
			projectName,
			covPages,
		)
		if coverError != nil {
			log.Printf("[TailorAsync] Cover letter generation error for %s: %v", jRecord.Company, coverError)
			if covVerID != "" {
				_, _ = handler.DB.Exec(backgroundCtx, `UPDATE cover_letter_versions SET status = 'failed', error_message = $1 WHERE id = $2`, coverError.Error(), covVerID)
			}
			failTitle := fmt.Sprintf("Cover Letter Failed: %s", jRecord.Company)
			failMessage := fmt.Sprintf("Resume succeeded but cover letter failed for %s: %s", jRecord.Title, coverError.Error())
			_, _ = handler.DB.Exec(backgroundCtx, `INSERT INTO notifications (user_id, title, message, is_read) VALUES ($1, $2, $3, false)`, uID, failTitle, failMessage)
			return
		}

		if covVerID != "" {
			_, _ = handler.DB.Exec(
				backgroundCtx,
				`UPDATE cover_letter_versions
				 SET status = 'ready', overleaf_project_name = $1, pdf_url = $2, page_count = $3
				 WHERE id = $4`,
				projectName, resumeWebURL, coverResult.CompileResult.PageCount, covVerID,
			)
		}

		log.Printf("[TailorAsync] Tailoring completed successfully for %s at %s (resume=%dp, cover=%dp)", jRecord.Title, jRecord.Company, resumeResult.CompileResult.PageCount, coverResult.CompileResult.PageCount)

		successTitle := fmt.Sprintf("Application Ready: %s", jRecord.Company)
		successMessage := fmt.Sprintf("Your tailored resume (%d p) and cover letter (%d p) for %s at %s are ready in Open-Overleaf.", resumeResult.CompileResult.PageCount, coverResult.CompileResult.PageCount, jRecord.Title, jRecord.Company)
		_, _ = handler.DB.Exec(
			backgroundCtx,
			`INSERT INTO notifications (user_id, title, message, is_read) VALUES ($1, $2, $3, false)`,
			uID, successTitle, successMessage,
		)
	}(context.Background(), userID, payload.JobID, jobRecord, userBio, targetResumePages, targetCoverLetterPages, resumeVersionID, coverVersionID, folderPath)

	ginContext.JSON(http.StatusAccepted, gin.H{
		"status":                    "processing",
		"message":                   "Application tailoring started in background. Resume and cover letter are being generated.",
		"job_id":                    payload.JobID,
		"target_resume_pages":       targetResumePages,
		"target_cover_letter_pages": targetCoverLetterPages,
		"resume_version_id":         resumeVersionID,
		"cover_letter_version_id":   coverVersionID,
	})
}
