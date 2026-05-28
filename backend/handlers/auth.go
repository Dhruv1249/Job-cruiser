package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

// AuthHandler holds the database connection pool so our functions can use it
type AuthHandler struct {
	DB *pgxpool.Pool
}

// Signup payload expected from Flutter
type SignupRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"primary_email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest

	// Validate the incoming JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Hash the password using Bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt password"})
		return
	}

	// Insert into CockroachDB
	var newUserID string
	query := `
		INSERT INTO users (full_name, primary_email, password_hash) 
		VALUES ($1, $2, $3) 
		RETURNING id;
	`

	err = h.DB.QueryRow(context.Background(), query, req.FullName, req.Email, string(hashedPassword)).Scan(&newUserID)
	if err != nil {
		// Basic check for duplicate emails (CockroachDB will throw a unique constraint error)
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists or database error"})
		return
	}

	// Generate the JWT VIP pass
	tokenString, err := utils.GenerateToken(newUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Send success response back to Flutter
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"token":   tokenString,
		"user_id": newUserID,
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

	// Sometimes users don't have a picture set, handle gracefully
	var avatar string
	if val, ok := payload.Claims["picture"]; ok {
		avatar = val.(string)
	}

	// 3. Upsert User: Insert if new, Update avatar/name if they already exist
	var userID string
	query := `
		INSERT INTO users (primary_email, full_name, avatar_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (primary_email) 
		DO UPDATE SET avatar_url = EXCLUDED.avatar_url, full_name = EXCLUDED.full_name
		RETURNING id;
	`

	err = h.DB.QueryRow(context.Background(), query, email, name, avatar).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error saving user"})
		return
	}

	// 4. Hand them our custom VIP pass
	tokenString, err := utils.GenerateToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Google Login successful",
		"token":   tokenString,
		"user_id": userID,
	})
}
