package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplicationHandler struct {
	DB *pgxpool.Pool
}

type CreateApplicationRequest struct {
	JobID  string `json:"job_id" binding:"required"`
	Status string `json:"status"` // e.g., 'bookmarked', 'applied', 'interviewing', 'rejected'
}

// CreateApplication saves a job to the user's pipeline
func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Default to bookmarked if they don't provide a status
	if req.Status == "" {
		req.Status = "bookmarked"
	}

	var appID string
	query := `
		INSERT INTO applications (user_id, job_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING -- Prevent saving the same job twice
		RETURNING id;
	`

	err := h.DB.QueryRow(context.Background(), query, userID, req.JobID, req.Status).Scan(&appID)
	if err != nil {
		// If it returns no rows, it means the ON CONFLICT caught a duplicate
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusConflict, gin.H{"error": "Job already saved"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save application"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Job saved successfully", "application_id": appID})
}

/*
GetUserApplications fetches all job applications tracked by the authenticated user with company details.
*/
func (h *ApplicationHandler) GetUserApplications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	query := `
		SELECT a.id, a.job_id, a.status, a.applied_at::text, 
		       j.title, COALESCE(comp.name, 'Unknown Company') as company_name, j.company_id, j.location,
		       COALESCE(m.match_score, 0), COALESCE(j.url, ''), COALESCE(j.is_remote, false), COALESCE(j.seniority, '')
		FROM applications a
		JOIN jobs j ON a.job_id = j.id
		LEFT JOIN companies comp ON j.company_id = comp.id
		LEFT JOIN user_job_matches m ON m.job_id = j.id AND m.user_id = $1
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC;
	`

	rows, err := h.DB.Query(c.Request.Context(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch applications"})
		return
	}
	defer rows.Close()

	var applications []gin.H
	for rows.Next() {
		var id, jobID, status, title, companyName, companyID string
		var location, appliedAt *string
		var matchScore int
		var jobURL, seniority string
		var isRemote bool

		if err := rows.Scan(&id, &jobID, &status, &appliedAt, &title, &companyName, &companyID, &location, &matchScore, &jobURL, &isRemote, &seniority); err != nil {
			continue
		}

		applications = append(applications, gin.H{
			"application_id": id,
			"job_id":         jobID,
			"status":         status,
			"title":          title,
			"company_name":   companyName,
			"company_id":     companyID,
			"location":       location,
			"applied_at":     appliedAt,
			"match_score":    matchScore,
			"url":            jobURL,
			"is_remote":      isRemote,
			"seniority":      seniority,
		})
	}

	if applications == nil {
		applications = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{"data": applications})
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

/*
UpdateApplicationStatus updates the lifecycle stage status for a tracked application.
*/
func (h *ApplicationHandler) UpdateApplicationStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")
	applicationID := c.Param("id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `
		UPDATE applications 
		SET status = $1 
		WHERE id = $2 AND user_id = $3
		RETURNING id;
	`

	var updatedID string
	err := h.DB.QueryRow(c.Request.Context(), query, req.Status, applicationID, userID).Scan(&updatedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

/*
DeleteApplication removes an application from the user's CRM pipeline.
*/
func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	applicationID := c.Param("id")

	query := `
		DELETE FROM applications 
		WHERE id = $1 AND user_id = $2
		RETURNING id;
	`

	var deletedID string
	err := h.DB.QueryRow(c.Request.Context(), query, applicationID, userID).Scan(&deletedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Application removed successfully", "id": deletedID})
}
