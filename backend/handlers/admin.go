package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

/*
AdminHandler provides handlers for Master Admin control.
*/
type AdminHandler struct {
	DB *pgxpool.Pool
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

	query := `SELECT id, email, COALESCE(notes, ''), created_at FROM whitelisted_emails ORDER BY created_at DESC;`
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
		SELECT id, keyword, status, created_at
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
