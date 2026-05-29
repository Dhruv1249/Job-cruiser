package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Dhruv1249/Job-cruiser/backend/services"
)

type MatchHandler struct {
	DB           *pgxpool.Pool
	AIService    *services.AIMatcherService
	BasicService *services.BasicMatcherService
}

type MatchRequest struct {
	JobID string `json:"job_id" binding:"required"`
	UseAI bool   `json:"use_ai"` // Flag explicitly indicating evaluation model choice
}

func (h *MatchHandler) EvaluateJobMatch(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req MatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload parameters"})
		return
	}

	var res *services.MatchResponse
	var err error
	isAIMatched := false

	if req.UseAI {
		// Premium Tier Enforcement Check
		var tier string
		err = h.DB.QueryRow(context.Background(), "SELECT subscription_tier FROM users WHERE id = $1", userID).Scan(&tier)
		if err != nil || tier != "premium" {
			c.JSON(http.StatusForbidden, gin.H{"error": "AI deep matching requires a premium subscription tier"})
			return
		}
		
		res, err = h.AIService.ComputePremiumMatch(userID.(string), req.JobID)
		isAIMatched = true
	} else {
		// Free/Basic execution calculation fallback
		res, err = h.BasicService.ComputeBasicMatch(userID.(string), req.JobID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist matching output state down to CockroachDB
	reasonsJSON, _ := json.Marshal(res.MatchReasons)
	query := `
		INSERT INTO user_job_matches (user_id, job_id, match_score, match_reasons, suggested_action, is_ai_matched)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, job_id) 
		DO UPDATE SET match_score = EXCLUDED.match_score, 
		              match_reasons = EXCLUDED.match_reasons, 
		              suggested_action = EXCLUDED.suggested_action,
		              is_ai_matched = EXCLUDED.is_ai_matched;
	`
	_, err = h.DB.Exec(context.Background(), query, userID, req.JobID, res.MatchScore, reasonsJSON, res.SuggestedAction, isAIMatched)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist evaluation state metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"match_score":      res.MatchScore,
		"match_reasons":    res.MatchReasons,
		"suggested_action": res.SuggestedAction,
		"is_ai_matched":    isAIMatched,
	})
}
