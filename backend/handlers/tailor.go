package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
	JobID       string `json:"job_id" binding:"required"`
	TargetPages int    `json:"target_pages"`
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

func (handler *TailorHandler) buildPDFWebURL(credentials *services.UserOverleafCredentials) string {
	return fmt.Sprintf("%s/?project=%s", strings.TrimRight(credentials.DeploymentURL, "/"), "job_applications")
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
		if errors.Is(mcpError, services.ErrNoOverleafConfig) {
			ginContext.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Configure open-overleaf first in Preferences"})
			return
		}
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load open-overleaf config"})
		return
	}

	targetPages := payload.TargetPages
	if targetPages <= 0 {
		targetPages = 1
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
		userID, payload.JobID, versionLabel, "job_applications", folderPath, result.PDFWebURL, result.CompileResult.PageCount,
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
		if errors.Is(mcpError, services.ErrNoOverleafConfig) {
			ginContext.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Configure open-overleaf first in Preferences"})
			return
		}
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load open-overleaf config"})
		return
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
		userID, payload.JobID, versionLabel, "job_applications", folderPath, result.PDFWebURL, result.CompileResult.PageCount,
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
