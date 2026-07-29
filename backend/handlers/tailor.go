package handlers

import (
	"net/http"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TailorHandler manages HTTP REST requests for AI resume and cover letter tailoring.
type TailorHandler struct {
	TailorService *services.ResumeTailorService
	DB            *pgxpool.Pool
}

// TailorRequestPayload defines incoming JSON parameters for tailoring operations.
type TailorRequestPayload struct {
	JobID          string `json:"job_id"`
	UserBio        string `json:"user_bio"`
	JobDescription string `json:"job_description"`
	ProjectName    string `json:"project_name"`
	FilePath       string `json:"file_path"`
	TargetPages    int    `json:"target_pages"`
}

// NewTailorHandler initializes a TailorHandler struct.
func NewTailorHandler(tailorService *services.ResumeTailorService, db *pgxpool.Pool) *TailorHandler {
	return &TailorHandler{
		TailorService: tailorService,
		DB:            db,
	}
}

// TailorResume generates a tailored resume .tex file and compiles it via open-overleaf MCP.
func (handler *TailorHandler) TailorResume(context *gin.Context) {
	var payload TailorRequestPayload
	if err := context.ShouldBindJSON(&payload); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	userBioText := payload.UserBio
	jobDescText := payload.JobDescription

	if payload.JobID != "" && handler.DB != nil {
		var fetchedDesc string
		queryError := handler.DB.QueryRow(context.Request.Context(), "SELECT raw_description FROM jobs WHERE id = $1", payload.JobID).Scan(&fetchedDesc)
		if queryError == nil && fetchedDesc != "" {
			jobDescText = fetchedDesc
		}
	}

	userIDValue, exists := context.Get("user_id")
	if exists && userBioText == "" && handler.DB != nil {
		var fetchedBio string
		queryError := handler.DB.QueryRow(context.Request.Context(), "SELECT bio_summary FROM user_preferences WHERE user_id = $1", userIDValue).Scan(&fetchedBio)
		if queryError == nil {
			userBioText = fetchedBio
		}
	}

	if userBioText == "" {
		userBioText = "Experienced software engineer specializing in backend systems, Go, cloud infrastructure, and AI integration."
	}
	if jobDescText == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "job_description or valid job_id is required"})
		return
	}

	projectName := payload.ProjectName
	if projectName == "" {
		projectName = "master_resume"
	}

	filePath := payload.FilePath
	if filePath == "" {
		filePath = "main.tex"
	}

	targetPages := payload.TargetPages
	if targetPages <= 0 {
		targetPages = 1
	}

	result, err := handler.TailorService.TailorResumeDirect(context.Request.Context(), userBioText, jobDescText, projectName, filePath, targetPages)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"message":        "Resume tailored and compiled successfully via open-overleaf MCP",
		"project_name":   result.ProjectName,
		"file_path":      result.FilePath,
		"target_pages":   result.TargetPages,
		"tailored_tex":   result.TailoredTeX,
		"compile_result": result.CompileResult,
		"pdf_result":     result.PDFResult,
	})
}

// GenerateCoverLetter drafts a custom LaTeX cover letter and compiles it via open-overleaf MCP.
func (handler *TailorHandler) GenerateCoverLetter(context *gin.Context) {
	var payload TailorRequestPayload
	if err := context.ShouldBindJSON(&payload); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	userBioText := payload.UserBio
	jobDescText := payload.JobDescription

	if payload.JobID != "" && handler.DB != nil {
		var fetchedDesc string
		queryError := handler.DB.QueryRow(context.Request.Context(), "SELECT raw_description FROM jobs WHERE id = $1", payload.JobID).Scan(&fetchedDesc)
		if queryError == nil && fetchedDesc != "" {
			jobDescText = fetchedDesc
		}
	}

	userIDValue, exists := context.Get("user_id")
	if exists && userBioText == "" && handler.DB != nil {
		var fetchedBio string
		queryError := handler.DB.QueryRow(context.Request.Context(), "SELECT bio_summary FROM user_preferences WHERE user_id = $1", userIDValue).Scan(&fetchedBio)
		if queryError == nil {
			userBioText = fetchedBio
		}
	}

	if userBioText == "" {
		userBioText = "Experienced software engineer specializing in backend systems, Go, cloud infrastructure, and AI integration."
	}
	if jobDescText == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "job_description or valid job_id is required"})
		return
	}

	projectName := payload.ProjectName
	if projectName == "" {
		projectName = "cover_letters"
	}

	filePath := payload.FilePath
	if filePath == "" {
		filePath = "cover_letter.tex"
	}

	result, err := handler.TailorService.GenerateCoverLetterDirect(context.Request.Context(), userBioText, jobDescText, projectName, filePath)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"message":        "Cover letter drafted and compiled successfully via open-overleaf MCP",
		"project_name":   result.ProjectName,
		"file_path":      result.FilePath,
		"target_pages":   result.TargetPages,
		"tailored_tex":   result.TailoredTeX,
		"compile_result": result.CompileResult,
		"pdf_result":     result.PDFResult,
	})
}
