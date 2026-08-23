package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VersionsHandler manages HTTP endpoints for resume and cover letter version retrieval and management.
type VersionsHandler struct {
	DB        *pgxpool.Pool
	AESKey    []byte
	MCPSecret string
}

// NewVersionsHandler constructs a VersionsHandler with database pool and encryption key.
func NewVersionsHandler(db *pgxpool.Pool, aesKey []byte, mcpSecret string) *VersionsHandler {
	return &VersionsHandler{
		DB:        db,
		AESKey:    aesKey,
		MCPSecret: mcpSecret,
	}
}

// ResumeVersionItem represents a single resume version record returned to the client.
type ResumeVersionItem struct {
	ID                  string    `json:"id"`
	JobID               *string   `json:"job_id"`
	Label               string    `json:"label"`
	OverleafProjectName string    `json:"overleaf_project_name"`
	OverleafFolderPath  string    `json:"overleaf_folder_path"`
	PDFUrl              string    `json:"pdf_url"`
	PageCount           int       `json:"page_count"`
	IsDefault           bool      `json:"is_default"`
	Status              string    `json:"status"`
	ErrorMessage        string    `json:"error_message"`
	CreatedAt           time.Time `json:"created_at"`
}

// CoverLetterVersionItem represents a single cover letter version record returned to the client.
type CoverLetterVersionItem struct {
	ID                  string    `json:"id"`
	JobID               *string   `json:"job_id"`
	ApplicationID       *string   `json:"application_id"`
	Label               string    `json:"label"`
	OverleafProjectName string    `json:"overleaf_project_name"`
	OverleafFolderPath  string    `json:"overleaf_folder_path"`
	PDFUrl              string    `json:"pdf_url"`
	PageCount           int       `json:"page_count"`
	Status              string    `json:"status"`
	ErrorMessage        string    `json:"error_message"`
	CreatedAt           time.Time `json:"created_at"`
}

// ListResumeVersions returns all resume versions for the authenticated user, ordered by newest first.
func (handler *VersionsHandler) ListResumeVersions(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, queryError := handler.DB.Query(
		ginContext.Request.Context(),
		`SELECT id, job_id, label, COALESCE(overleaf_project_name, 'job_applications'),
		        COALESCE(overleaf_folder_path, ''), COALESCE(pdf_url, ''),
		        COALESCE(page_count, 1), COALESCE(is_default, false),
		        COALESCE(status, 'ready'), COALESCE(error_message, ''), created_at
		 FROM resume_versions
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userIDValue,
	)
	if queryError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch resume versions"})
		return
	}
	defer rows.Close()

	versionItems := make([]ResumeVersionItem, 0)
	for rows.Next() {
		var item ResumeVersionItem
		scanError := rows.Scan(
			&item.ID, &item.JobID, &item.Label,
			&item.OverleafProjectName, &item.OverleafFolderPath,
			&item.PDFUrl, &item.PageCount, &item.IsDefault,
			&item.Status, &item.ErrorMessage, &item.CreatedAt,
		)
		if scanError != nil {
			continue
		}
		versionItems = append(versionItems, item)
	}

	ginContext.JSON(http.StatusOK, gin.H{"data": versionItems, "count": len(versionItems)})
}

// GetResumeVersionPDF re-fetches the PDF base64 for a saved resume version from open-overleaf via MCP.
func (handler *VersionsHandler) GetResumeVersionPDF(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	versionID := ginContext.Param("id")
	userID := fmt.Sprintf("%v", userIDValue)

	var folderPath, projectName string
	scanError := handler.DB.QueryRow(
		ginContext.Request.Context(),
		`SELECT COALESCE(overleaf_folder_path, ''), COALESCE(overleaf_project_name, 'job_applications')
		 FROM resume_versions WHERE id = $1 AND user_id = $2`,
		versionID, userIDValue,
	).Scan(&folderPath, &projectName)
	if scanError != nil {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Resume version not found"})
		return
	}

	mcpClient, _, mcpError := services.LoadUserMCPClient(
		ginContext.Request.Context(),
		handler.DB,
		userID,
		handler.AESKey,
		handler.MCPSecret,
	)
	if mcpError != nil {
		ginContext.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Configure open-overleaf first"})
		return
	}

	pdfFileName := fmt.Sprintf("%s/resume.pdf", folderPath)
	pdfResult, pdfError := mcpClient.GetProjectPDF(ginContext.Request.Context(), projectName, pdfFileName)
	if pdfError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch PDF from open-overleaf"})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{
		"version_id": versionID,
		"pdf_base64": pdfResult.Base64Data,
		"page_count": pdfResult.PageCount,
		"size_bytes": pdfResult.SizeBytes,
	})
}

// DeleteResumeVersion removes a resume version reference from Postgres.
// The LaTeX source in open-overleaf is retained (version history via GitHub).
func (handler *VersionsHandler) DeleteResumeVersion(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	versionID := ginContext.Param("id")

	commandTag, deleteError := handler.DB.Exec(
		ginContext.Request.Context(),
		`DELETE FROM resume_versions WHERE id = $1 AND user_id = $2`,
		versionID, userIDValue,
	)
	if deleteError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete resume version"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Resume version not found"})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{"message": "Resume version deleted"})
}

// SetDefaultResumeVersion marks one resume version as the default and clears the flag on all others.
func (handler *VersionsHandler) SetDefaultResumeVersion(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	versionID := ginContext.Param("id")

	_, clearError := handler.DB.Exec(
		ginContext.Request.Context(),
		`UPDATE resume_versions SET is_default = false WHERE user_id = $1`,
		userIDValue,
	)
	if clearError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update default flag"})
		return
	}

	commandTag, setError := handler.DB.Exec(
		ginContext.Request.Context(),
		`UPDATE resume_versions SET is_default = true WHERE id = $1 AND user_id = $2`,
		versionID, userIDValue,
	)
	if setError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default resume version"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Resume version not found"})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{"message": "Default resume version updated"})
}

// ListCoverLetterVersions returns all cover letter versions for the authenticated user, newest first.
func (handler *VersionsHandler) ListCoverLetterVersions(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, queryError := handler.DB.Query(
		ginContext.Request.Context(),
		`SELECT id, job_id, application_id, label,
		        COALESCE(overleaf_project_name, 'job_applications'),
		        COALESCE(overleaf_folder_path, ''), COALESCE(pdf_url, ''),
		        COALESCE(page_count, 1), COALESCE(status, 'ready'),
		        COALESCE(error_message, ''), created_at
		 FROM cover_letter_versions
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userIDValue,
	)
	if queryError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cover letter versions"})
		return
	}
	defer rows.Close()

	coverLetterItems := make([]CoverLetterVersionItem, 0)
	for rows.Next() {
		var item CoverLetterVersionItem
		scanError := rows.Scan(
			&item.ID, &item.JobID, &item.ApplicationID, &item.Label,
			&item.OverleafProjectName, &item.OverleafFolderPath,
			&item.PDFUrl, &item.PageCount, &item.Status,
			&item.ErrorMessage, &item.CreatedAt,
		)
		if scanError != nil {
			continue
		}
		coverLetterItems = append(coverLetterItems, item)
	}

	ginContext.JSON(http.StatusOK, gin.H{"data": coverLetterItems, "count": len(coverLetterItems)})
}

// GetCoverLetterPDF re-fetches the PDF base64 for a saved cover letter version from open-overleaf.
func (handler *VersionsHandler) GetCoverLetterPDF(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	coverLetterID := ginContext.Param("id")
	userID := fmt.Sprintf("%v", userIDValue)

	var folderPath, projectName string
	scanError := handler.DB.QueryRow(
		ginContext.Request.Context(),
		`SELECT COALESCE(overleaf_folder_path, ''), COALESCE(overleaf_project_name, 'job_applications')
		 FROM cover_letter_versions WHERE id = $1 AND user_id = $2`,
		coverLetterID, userIDValue,
	).Scan(&folderPath, &projectName)
	if scanError != nil {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Cover letter not found"})
		return
	}

	mcpClient, _, mcpError := services.LoadUserMCPClient(
		ginContext.Request.Context(),
		handler.DB,
		userID,
		handler.AESKey,
		handler.MCPSecret,
	)
	if mcpError != nil {
		ginContext.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Configure open-overleaf first"})
		return
	}

	pdfFileName := fmt.Sprintf("%s/cover_letter.pdf", folderPath)
	pdfResult, pdfError := mcpClient.GetProjectPDF(ginContext.Request.Context(), projectName, pdfFileName)
	if pdfError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch PDF from open-overleaf"})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{
		"cover_letter_id": coverLetterID,
		"pdf_base64":      pdfResult.Base64Data,
		"page_count":      pdfResult.PageCount,
		"size_bytes":      pdfResult.SizeBytes,
	})
}

// DeleteCoverLetterVersion removes a cover letter version reference from Postgres.
func (handler *VersionsHandler) DeleteCoverLetterVersion(ginContext *gin.Context) {
	userIDValue, exists := ginContext.Get("user_id")
	if !exists {
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	coverLetterID := ginContext.Param("id")

	commandTag, deleteError := handler.DB.Exec(
		ginContext.Request.Context(),
		`DELETE FROM cover_letter_versions WHERE id = $1 AND user_id = $2`,
		coverLetterID, userIDValue,
	)
	if deleteError != nil {
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete cover letter"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		ginContext.JSON(http.StatusNotFound, gin.H{"error": "Cover letter not found"})
		return
	}

	ginContext.JSON(http.StatusOK, gin.H{"message": "Cover letter version deleted"})
}
