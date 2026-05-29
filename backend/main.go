package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/db"
	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/Dhruv1249/Job-cruiser/backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load the .env file. If it fails, we log a warning but don't crash,
	loadError := godotenv.Load()
	if loadError != nil {
		log.Println("Note: No local .env file found. Relying on system variables.")
	}
	// Fetch the database connection string from the environment variables.
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("CRITICAL ERROR: DATABASE_URL environment variable is missing.")
	}

	// Create a background context for the database connection process.
	backgroundContext := context.Background()

	// Initialize a pool of connections to CockroachDB.
	// We use a pool instead of a single connection so multiple users can hit the API at once.
	databasePool, connectionError := pgxpool.New(backgroundContext, databaseURL)
	if connectionError != nil {
		log.Fatalf("CRITICAL ERROR: Failed to connect to the database. Details: %v", connectionError)
	}

	// 'defer' ensures the database connections are properly closed when the program shuts down.
	defer databasePool.Close()

	var serverTime time.Time

	// Send a test query to ask the database for its current time.
	queryError := databasePool.QueryRow(backgroundContext, "SELECT NOW()").Scan(&serverTime)
	if queryError != nil {
		log.Fatalf("CRITICAL ERROR: Connected to DB, but test query failed. Details: %v", queryError)
	}

	// Format the raw time data into a readable text string.
	formattedTime := serverTime.Format(time.RFC3339)
	fmt.Printf("Successfully connected to CockroachDB! Server time: %s\n", formattedTime)

	schemaError := db.InitSchema(databasePool)
	if schemaError != nil {
		log.Fatalf("CRITICAL ERROR: Failed to initialize database schema. Details: %v", schemaError)
	}
	println("Database schema initialized.")

	authHandler := &handlers.AuthHandler{DB: databasePool}
	jobHandler := &handlers.JobHandler{DB: databasePool}
	prefHandler := &handlers.PreferencesHandler{DB: databasePool}
	appHandler := &handlers.ApplicationHandler{DB: databasePool}

	// Initialize the default Gin web router with basic logging and crash-recovery built in.
	webRouter := gin.Default()

	public := webRouter.Group("/api")
	{
		public.POST("/signup", authHandler.Signup)
		public.POST("/login", authHandler.Login)
		public.POST("/auth/google", authHandler.GoogleLogin)
	}
	// Protected Routes (Requires JWT)
	protected := webRouter.Group("/api")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/jobs", jobHandler.GetJobs)
		protected.POST("/preferences", prefHandler.UpdatePreferences)
		protected.GET("/preferences", prefHandler.GetPreferences)

		protected.POST("/applications", appHandler.CreateApplication)
		protected.GET("/applications", appHandler.GetUserApplications)
		protected.PUT("/applications/:id/status", appHandler.UpdateApplicationStatus)
	}

	// Check if a specific network port was requested in the .env file.
	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080" // Default to port 8080 if none is specified.
	}

	fmt.Printf("Starting web server on port %s...\n", serverPort)

	// Turn the server on and lock it in an infinite loop listening for internet traffic.
	serverError := webRouter.Run(":" + serverPort)
	if serverError != nil {
		log.Fatalf("CRITICAL ERROR: The web server crashed. Details: %v", serverError)
	}
}
