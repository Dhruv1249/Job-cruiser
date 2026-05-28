package handlers

import (
	"context"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

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
