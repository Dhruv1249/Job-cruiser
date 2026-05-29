package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

// AuthHandler holds the database connection pool so our functions can use it
type AuthHandler struct {
	DB *pgxpool.Pool
}

type SignupRequest struct {
	Email    string `json:"primary_email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
		return
	}

	var newUserID string
	query := `
		INSERT INTO users (primary_email, password_hash) 
		VALUES ($1, $2) 
		RETURNING id;
	`

	err = h.DB.QueryRow(context.Background(), query, req.Email, string(hashedPassword)).Scan(&newUserID)
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
		"message":     "User created successfully",
		"token":       tokenString,
		"user_id":     newUserID,
		"is_new_user": true, // Standardized routing flag
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
	query := `
		SELECT id,password_hash FROM users 
		WHERE primary_email = $1;
	`
	err := h.DB.QueryRow(context.Background(), query, req.Email).Scan(&userID, &passwordHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate the JWT VIP pass
	tokenString, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Send success response back to Flutter
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   tokenString,
		"user_id": userID,
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

	// 1. Ask Google if the token is real
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	payload, err := idtoken.Validate(context.Background(), req.IDToken, clientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token"})
		return
	}

	// 2. Extract the safe data
	email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)
	googleID := payload.Claims["sub"].(string) // Extract Google's unique account ID

	// Sometimes users don't have a picture set, handle gracefully
	var avatar string
	if val, ok := payload.Claims["picture"]; ok {
		avatar = val.(string)
	}

	// 3. Check if the user already exists
	var userID string
	isNewUser := false

	err = h.DB.QueryRow(context.Background(), "SELECT id FROM users WHERE primary_email = $1", email).Scan(&userID)

	if err == pgx.ErrNoRows {
		isNewUser = true
		insertQuery := `
			INSERT INTO users (primary_email, avatar_url, google_id, auth_provider)
			VALUES ($1, $2, $3, 'google')
			RETURNING id;
		`
		err = h.DB.QueryRow(context.Background(), insertQuery, email, avatar, googleID).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error creating new user"})
			return
		}
	} else if err != nil {
		// A real database connection error occurred
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking user"})
		return
	} else {
		// EXISTING USER: Just update their avatar in case they changed it on Google
		updateQuery := `UPDATE users SET avatar_url = $1, google_id = $2 WHERE id = $3`
		h.DB.Exec(context.Background(), updateQuery, avatar, googleID, userID)
	}

	//  Hand them our custom VIP pass
	tokenString, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}

	//  Send the payload back to Flutter, including the routing flag
	c.JSON(http.StatusOK, gin.H{
		"message":        "Google Login successful",
		"token":          tokenString,
		"user_id":        userID,
		"is_new_user":    isNewUser,
		"suggested_name": name, // Frontend can use this to pre-fill the name field!
	})
}
