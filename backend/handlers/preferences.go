package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
PreferencesHandler manages user settings, bio text, master CV, and overleaf configuration.
*/
type PreferencesHandler struct {
	DB *pgxpool.Pool
}

type PreferencesRequest struct {
	FullName          string   `json:"full_name" binding:"required"`
	TargetRoles       []string `json:"target_roles" binding:"required"`
	WorkModels        []string `json:"work_models" binding:"required"`
	MinSalary         int      `json:"min_salary"`
	Currency          string   `json:"currency"`
	MasterCVText      string   `json:"master_cv_text"`
	BioExperienceText string   `json:"bio_experience_text"`
}

/*
UpdatePreferences saves or updates a user's preferences profile.
*/
func (h *PreferencesHandler) UpdatePreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	query := `
		INSERT INTO user_preferences (user_id, full_name, target_roles, work_models, min_salary, currency, master_cv_text, bio_experience_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			full_name = EXCLUDED.full_name,
			target_roles = EXCLUDED.target_roles,
			work_models = EXCLUDED.work_models,
			min_salary = EXCLUDED.min_salary,
			currency = EXCLUDED.currency,
			master_cv_text = EXCLUDED.master_cv_text,
			bio_experience_text = EXCLUDED.bio_experience_text,
			updated_at = CURRENT_TIMESTAMP;
	`

	_, err := h.DB.Exec(context.Background(), query, userID, req.FullName, req.TargetRoles, req.WorkModels, req.MinSalary, req.Currency, req.MasterCVText, req.BioExperienceText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences saved successfully"})
}

/*
GetPreferences retrieves a user's settings profile.
*/
func (h *PreferencesHandler) GetPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT full_name, target_roles, work_models, min_salary, currency, COALESCE(master_cv_text, ''), COALESCE(bio_experience_text, '') 
		FROM user_preferences 
		WHERE user_id = $1;
	`

	var pref PreferencesRequest
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(
		&pref.FullName, &pref.TargetRoles, &pref.WorkModels, &pref.MinSalary, &pref.Currency, &pref.MasterCVText, &pref.BioExperienceText,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusOK, gin.H{"data": nil, "message": "No preferences set yet"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": pref})
}

type OverleafConfigRequest struct {
	DeploymentURL string `json:"deployment_url" binding:"required"`
	GitHubUsername string `json:"github_username" binding:"required"`
	GitHubRepoName string `json:"github_repo_name" binding:"required"`
	AccessToken    string `json:"access_token" binding:"required"`
}

/*
UpdateOverleafConfig saves self-hosted open-overleaf configuration for the user.
*/
func (h *PreferencesHandler) UpdateOverleafConfig(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OverleafConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := `
		INSERT INTO user_overleaf_config (user_id, deployment_url, github_username, github_repo_name, encrypted_access_token)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id)
		DO UPDATE SET
			deployment_url = EXCLUDED.deployment_url,
			github_username = EXCLUDED.github_username,
			github_repo_name = EXCLUDED.github_repo_name,
			encrypted_access_token = EXCLUDED.encrypted_access_token,
			updated_at = CURRENT_TIMESTAMP;
	`

	_, err := h.DB.Exec(context.Background(), query, userID, req.DeploymentURL, req.GitHubUsername, req.GitHubRepoName, req.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save open-overleaf configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Self-hosted open-overleaf configured successfully"})
}

/*
GetOverleafConfig retrieves self-hosted open-overleaf settings for the user.
*/
func (h *PreferencesHandler) GetOverleafConfig(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT deployment_url, github_username, github_repo_name
		FROM user_overleaf_config
		WHERE user_id = $1;
	`

	var url, username, repo string
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(&url, &username, &repo)
	if err != nil {
		if err.Error() == "no rows in result set" {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch open-overleaf config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"deployment_url":  url,
			"github_username": username,
			"github_repo_name": repo,
		},
	})
}
