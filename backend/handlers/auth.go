package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

/*
AuthHandler holds the database connection pool so our functions can use it
*/
type AuthHandler struct {
	DB *pgxpool.Pool
}

type SignupRequest struct {
	Email    string `json:"primary_email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func isMasterAdminEmail(email string) bool {
	configuredAdminEmail := os.Getenv("MASTER_ADMIN_EMAIL")
	if configuredAdminEmail == "" {
		configuredAdminEmail = "dhr1249.lm@gmail.com"
	}
	return strings.EqualFold(email, configuredAdminEmail)
}

func (h *AuthHandler) isEmailWhitelisted(email string) bool {
	var count int
	query := `SELECT COUNT(*) FROM whitelisted_emails WHERE LOWER(email) = LOWER($1);`
	err := h.DB.QueryRow(context.Background(), query, email).Scan(&count)
	return err == nil && count > 0
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	if !h.isEmailWhitelisted(req.Email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access restricted. Your email is not whitelisted by Master Admin."})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
		return
	}

	isMasterAdmin := isMasterAdminEmail(req.Email)
	var newUserID string
	query := `
		INSERT INTO users (primary_email, password_hash, is_master_admin) 
		VALUES ($1, $2, $3) 
		RETURNING id;
	`

	err = h.DB.QueryRow(context.Background(), query, req.Email, string(hashedPassword), isMasterAdmin).Scan(&newUserID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists or database error"})
		return
	}

	tokenString, err := utils.GenerateToken(newUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "User created successfully",
		"token":           tokenString,
		"user_id":         newUserID,
		"is_new_user":     true,
		"is_master_admin": isMasterAdmin,
	})
}

type LoginRequest struct {
	Email    string `json:"primary_email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest

	// Validate the incoming JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Query the database for the user
	var userID string
	var passwordHash string
	var isMasterAdmin bool
	userQuery := `
		SELECT id, password_hash, COALESCE(is_master_admin, false) FROM users
		WHERE primary_email = $1;
	`
	err := h.DB.QueryRow(context.Background(), userQuery, req.Email).Scan(&userID, &passwordHash, &isMasterAdmin)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	tokenString, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	var hasPreferences bool
	var aiMatchingEnabled bool
	checkPrefQuery := `
		SELECT
			COALESCE(u.ai_matching_enabled, false),
			(up.user_id IS NOT NULL AND jsonb_array_length(COALESCE(up.target_roles, '[]'::jsonb)) > 0)
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.id = $1;
	`
	_ = h.DB.QueryRow(context.Background(), checkPrefQuery, userID).Scan(&aiMatchingEnabled, &hasPreferences)

	c.JSON(http.StatusOK, gin.H{
		"message":             "Login successful",
		"token":               tokenString,
		"user_id":             userID,
		"has_preferences":     hasPreferences,
		"ai_matching_enabled": aiMatchingEnabled,
		"is_master_admin":     isMasterAdmin,
	})
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var req GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing id_token"})
		return
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	var email, name, googleID, avatar string

	payload, err := idtoken.Validate(context.Background(), req.IDToken, clientID)
	if err == nil {
		email = payload.Claims["email"].(string)
		if n, ok := payload.Claims["name"].(string); ok {
			name = n
		}
		googleID = payload.Claims["sub"].(string)
		if val, ok := payload.Claims["picture"]; ok {
			avatar = val.(string)
		}
	} else {
		// Fallback for access_token (common on Flutter Web GIS)
		userInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo"
		httpReq, httpErr := http.NewRequestWithContext(context.Background(), "GET", userInfoURL, nil)
		if httpErr != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token"})
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+req.IDToken)

		resp, respErr := http.DefaultClient.Do(httpReq)
		if respErr != nil || resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token"})
			return
		}
		defer resp.Body.Close()

		var userInfo struct {
			Sub     string `json:"sub"`
			Name    string `json:"name"`
			Email   string `json:"email"`
			Picture string `json:"picture"`
		}
		if jsonErr := json.NewDecoder(resp.Body).Decode(&userInfo); jsonErr != nil || userInfo.Email == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token response"})
			return
		}

		email = userInfo.Email
		name = userInfo.Name
		googleID = userInfo.Sub
		avatar = userInfo.Picture
	}

	var userID string
	var isMasterAdmin bool
	isNewUser := false

	err = h.DB.QueryRow(context.Background(), "SELECT id, is_master_admin FROM users WHERE primary_email = $1", email).Scan(&userID, &isMasterAdmin)

	if err == pgx.ErrNoRows {
		if !h.isEmailWhitelisted(email) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access restricted. Your email is not whitelisted by Master Admin."})
			return
		}

		isNewUser = true
		isMasterAdmin = isMasterAdminEmail(email)
		insertQuery := `
			INSERT INTO users (primary_email, avatar_url, google_id, auth_provider, is_master_admin, ai_matching_enabled)
			VALUES ($1, $2, $3, 'google', $4, false)
			RETURNING id;
		`
		err = h.DB.QueryRow(context.Background(), insertQuery, email, avatar, googleID, isMasterAdmin).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error creating new user"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking user"})
		return
	} else {
		updateQuery := `UPDATE users SET avatar_url = $1, google_id = $2 WHERE id = $3`
		h.DB.Exec(context.Background(), updateQuery, avatar, googleID, userID)
	}

	tokenString, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}

	var hasPreferences bool
	var aiMatchingEnabled bool
	checkPrefQuery := `
		SELECT 
			COALESCE(u.ai_matching_enabled, false),
			(up.user_id IS NOT NULL AND jsonb_array_length(COALESCE(up.target_roles, '[]'::jsonb)) > 0)
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.id = $1;
	`
	_ = h.DB.QueryRow(context.Background(), checkPrefQuery, userID).Scan(&aiMatchingEnabled, &hasPreferences)

	c.JSON(http.StatusOK, gin.H{
		"message":             "Google Login successful",
		"token":               tokenString,
		"user_id":             userID,
		"is_new_user":         isNewUser,
		"is_master_admin":     isMasterAdmin,
		"suggested_name":      name,
		"has_preferences":     hasPreferences,
		"ai_matching_enabled": aiMatchingEnabled,
	})
}

/*
GetMe retrieves the authenticated user's profile details.
*/
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	query := `
		SELECT u.id, u.primary_email, COALESCE(up.full_name, ''), COALESCE(u.avatar_url, ''),
		       COALESCE(u.ai_matching_enabled, false),
		       COALESCE(u.is_master_admin, false),
		       (up.user_id IS NOT NULL AND jsonb_array_length(COALESCE(up.target_roles, '[]'::jsonb)) > 0) AS has_preferences
		FROM users u
		LEFT JOIN user_preferences up ON u.id = up.user_id
		WHERE u.id = $1;
	`

	var idString string
	var primaryEmailString string
	var fullNameString string
	var avatarURLString string
	var aiMatchingEnabled bool
	var isMasterAdmin bool
	var hasPreferences bool
	err := h.DB.QueryRow(context.Background(), query, userID).Scan(&idString, &primaryEmailString, &fullNameString, &avatarURLString, &aiMatchingEnabled, &isMasterAdmin, &hasPreferences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  idString,
		"primary_email":       primaryEmailString,
		"full_name":           fullNameString,
		"avatar_url":          avatarURLString,
		"ai_matching_enabled": aiMatchingEnabled,
		"is_master_admin":     isMasterAdmin,
		"has_preferences":     hasPreferences,
	})
}
