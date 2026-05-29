package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PreferencesHandler struct {
	DB *pgxpool.Pool
}

type PreferencesRequest struct {
	FullName    string   `json:"full_name" binding:"required"`
	TargetRoles []string `json:"target_roles" binding:"required"`
	WorkModels  []string `json:"work_models" binding:"required"` // e.g., ["remote", "hybrid"]
	MinSalary   int      `json:"min_salary"`
	Currency    string   `json:"currency"`
}

// UpdatePreferences creates or updates a user's settings profile
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	// 1. Grab the secure user_id injected by the RequireAuth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Validate the incoming JSON
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// 3. Upsert into CockroachDB
	query := `
		INSERT INTO user_preferences (user_id, full_name, target_roles, work_models, min_salary, currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			full_name = EXCLUDED.full_name,
			target_roles = EXCLUDED.target_roles,
			work_models = EXCLUDED.work_models,
			min_salary = EXCLUDED.min_salary,
			currency = EXCLUDED.currency,
			updated_at = CURRENT_TIMESTAMP;
	`

	_, err := h.DB.Exec(context.Background(), query, userID, req.FullName, req.TargetRoles, req.WorkModels, req.MinSalary, req.Currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences saved successfully"})
}

// GetPreferences fetches a user's current settings
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	// Grab the secure user_id injected by the RequireAuth middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT full_name, target_roles, work_models, min_salary, currency 
		FROM user_preferences 
		WHERE user_id = $1;
	`

	var pref PreferencesRequest
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(
		&pref.FullName, &pref.TargetRoles, &pref.WorkModels, &pref.MinSalary, &pref.Currency,
	)

	if err != nil {
		// If they haven't set preferences yet, return a clean 200 with null data
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusOK, gin.H{"data": nil, "message": "No preferences set yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": pref})
}
