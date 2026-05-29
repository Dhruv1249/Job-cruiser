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

// GetUserApplications fetches all jobs the user has interacted with
func (h *ApplicationHandler) GetUserApplications(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// We use a JOIN here to get the actual job details alongside the application status
	query := `
		SELECT a.id, a.job_id, a.status, a.applied_at, 
		       j.title, j.company_id, j.location 
		FROM applications a
		JOIN jobs j ON a.job_id = j.id
		WHERE a.user_id = $1
		ORDER BY a.created_at DESC;
	`

	rows, err := h.DB.Query(context.Background(), query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch applications"})
		return
	}
	defer rows.Close()

	// Using a slice of maps (gin.H) for a quick, flexible JSON response
	var applications []gin.H
	for rows.Next() {
		var id, jobID, status, title, companyID string
		var location *string
		var appliedAt *string

		if err := rows.Scan(&id, &jobID, &status, &appliedAt, &title, &companyID, &location); err != nil {
			continue
		}

		applications = append(applications, gin.H{
			"application_id": id,
			"job_id":         jobID,
			"status":         status,
			"title":          title,
			"company_id":     companyID,
			"location":       location,
			"applied_at":     appliedAt,
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

// UpdateApplicationStatus moves a job across the Kanban board
func (h *ApplicationHandler) UpdateApplicationStatus(c *gin.Context) {
	userID, _ := c.Get("user_id")
	applicationID := c.Param("id") // Get the ID from the URL: /applications/:id/status

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
	err := h.DB.QueryRow(context.Background(), query, req.Status, applicationID, userID).Scan(&updatedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}
