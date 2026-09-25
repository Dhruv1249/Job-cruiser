package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/Dhruv1249/Job-cruiser/backend/db"
	"github.com/Dhruv1249/Job-cruiser/backend/handlers"
	"github.com/Dhruv1249/Job-cruiser/backend/middleware"
	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/Dhruv1249/Job-cruiser/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	kolkataLocation, locationError := time.LoadLocation("Asia/Kolkata")
	if locationError == nil {
		time.Local = kolkataLocation
	}

	utils.InitColorLogger()
	utils.ClearRawAILogFile()
	gin.ForceConsoleColor()

	loadError := godotenv.Load()
	if loadError != nil {
		log.Println("Note: No local .env file found. Relying on system variables.")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("CRITICAL ERROR: DATABASE_URL environment variable is missing.")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		log.Fatal("CRITICAL ERROR: JWT_SECRET environment variable is missing.")
	}

	geminiAPIKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	if geminiAPIKey == "" {
		log.Fatal("CRITICAL ERROR: GEMINI_API_KEY environment variable is missing.")
	}

	geminiBatchModels := strings.TrimSpace(os.Getenv("GEMINI_BATCH_MODELS"))
	if geminiBatchModels == "" {
		log.Fatal("CRITICAL ERROR: GEMINI_BATCH_MODELS environment variable is missing.")
	}

	geminiModels := strings.TrimSpace(os.Getenv("GEMINI_MODELS"))
	if geminiModels == "" {
		log.Fatal("CRITICAL ERROR: GEMINI_MODELS environment variable is missing.")
	}

	nvidiaApiKey := strings.TrimSpace(os.Getenv("NVIDIA_API_KEY"))
	if nvidiaApiKey == "" {
		log.Fatal("CRITICAL ERROR: NVIDIA_API_KEY environment variable is missing.")
	}

	nvidiaModel := strings.TrimSpace(os.Getenv("NVIDIA_MODEL"))
	if nvidiaModel == "" {
		log.Fatal("CRITICAL ERROR: NVIDIA_MODEL environment variable is missing.")
	}

	overleafAESKeyHex := strings.TrimSpace(os.Getenv("OVERLEAF_AES_KEY"))
	if overleafAESKeyHex == "" {
		overleafAESKeyHex = strings.TrimSpace(os.Getenv("ENCRYPTION_KEY"))
	}

	var overleafAESKey []byte
	if overleafAESKeyHex != "" {
		decodedKey, hexError := hex.DecodeString(overleafAESKeyHex)
		if hexError == nil && len(decodedKey) == 32 {
			overleafAESKey = decodedKey
		} else {
			hash := sha256.Sum256([]byte(overleafAESKeyHex))
			overleafAESKey = hash[:]
		}
	} else {
		salt := strings.TrimSpace(os.Getenv("JWT_SECRET"))
		if salt == "" {
			salt = strings.TrimSpace(os.Getenv("DATABASE_URL"))
		}
		if salt == "" {
			log.Fatal("CRITICAL ERROR: No encryption key or salt configured (OVERLEAF_AES_KEY, ENCRYPTION_KEY, JWT_SECRET, or DATABASE_URL).")
		}
		hash := sha256.Sum256([]byte("job_cruiser_overleaf_aes_key:" + salt))
		overleafAESKey = hash[:]
	}

	ingestAPIKey := strings.TrimSpace(os.Getenv("INGEST_API_KEY"))
	if ingestAPIKey == "" {
		log.Fatal("CRITICAL ERROR: INGEST_API_KEY environment variable is missing.")
	}

	masterAdminEmail := strings.TrimSpace(os.Getenv("MASTER_ADMIN_EMAIL"))
	if masterAdminEmail == "" {
		log.Fatal("CRITICAL ERROR: MASTER_ADMIN_EMAIL environment variable is missing.")
	}

	googleClientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if googleClientID == "" {
		log.Fatal("CRITICAL ERROR: GOOGLE_CLIENT_ID environment variable is missing.")
	}

	serverPort := strings.TrimSpace(os.Getenv("PORT"))
	if serverPort == "" {
		serverPort = "8080"
	}

	firebaseProjectID := strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	if firebaseProjectID == "" {
		log.Fatal("CRITICAL ERROR: FIREBASE_PROJECT_ID environment variable is missing.")
	}
	firebaseClientEmail := strings.TrimSpace(os.Getenv("FIREBASE_CLIENT_EMAIL"))
	if firebaseClientEmail == "" {
		log.Fatal("CRITICAL ERROR: FIREBASE_CLIENT_EMAIL environment variable is missing.")
	}
	firebasePrivateKey := strings.TrimSpace(os.Getenv("FIREBASE_PRIVATE_KEY"))
	if firebasePrivateKey == "" {
		log.Fatal("CRITICAL ERROR: FIREBASE_PRIVATE_KEY environment variable is missing.")
	}

	backgroundContext := context.Background()
	databasePool, connectionError := pgxpool.New(backgroundContext, databaseURL)
	if connectionError != nil {
		log.Fatalf("CRITICAL ERROR: Failed to connect to the database. Details: %v", connectionError)
	}
	defer databasePool.Close()

	var serverTime time.Time
	queryError := databasePool.QueryRow(backgroundContext, "SELECT NOW()").Scan(&serverTime)
	if queryError != nil {
		log.Fatalf("CRITICAL ERROR: Connected to DB, but test query failed. Details: %v", queryError)
	}

	formattedTime := serverTime.Format(time.RFC3339)
	fmt.Printf("Successfully connected to PostgreSQL! Server time: %s\n", formattedTime)

	schemaError := db.InitSchema(databasePool)
	if schemaError != nil {
		log.Fatalf("CRITICAL ERROR: Failed to initialize database schema. Details: %v", schemaError)
	}
	println("Database schema initialized.")

	fcmService := services.NewFCMService()
	if fcmService == nil {
		log.Fatal("CRITICAL ERROR: Failed to initialize Firebase FCM service with provided credentials.")
	}

	nvidiaNimService := services.NewNvidiaNimService(databasePool, nvidiaApiKey)
	nvidiaNimService.FCMService = fcmService

	geminiBatchService := services.NewGeminiBatchMatchService(databasePool, geminiAPIKey)
	geminiBatchService.FCMService = fcmService
	hybridMatchService := services.NewHybridBatchMatchService(nvidiaNimService, geminiBatchService)

	hybridMatchService.StartBackgroundScheduler(context.Background())

	authHandler := &handlers.AuthHandler{DB: databasePool}
	jobHandler := &handlers.JobHandler{DB: databasePool}
	prefHandler := &handlers.PreferencesHandler{
		DB:           databasePool,
		MatchService: hybridMatchService,
		NimService:   nvidiaNimService,
		AESKey:       overleafAESKey,
		APIKey:       geminiAPIKey,
		MCPSecret:    "",
	}
	appHandler := &handlers.ApplicationHandler{DB: databasePool}
	ingestHandler := &handlers.IngestHandler{
		DB:           databasePool,
		MatchService: hybridMatchService,
	}
	matchedJobsHandler := &handlers.MatchedJobsHandler{DB: databasePool}
	adminHandler := &handlers.AdminHandler{DB: databasePool, MatchService: hybridMatchService}

	tailorService := services.NewResumeTailorService("https://generativelanguage.googleapis.com", geminiAPIKey, nil)
	tailorHandler := handlers.NewTailorHandler(tailorService, databasePool, overleafAESKey, "", fcmService)
	versionsHandler := handlers.NewVersionsHandler(databasePool, overleafAESKey, "")
	notificationsHandler := handlers.NewNotificationsHandler(databasePool)
	profileSyncService := services.NewProfileSyncService(databasePool, overleafAESKey, "")
	profileSyncService.StartBackgroundSyncScheduler(context.Background())

	webRouter := gin.Default()
	webRouter.Use(middleware.CORSMiddleware())

	public := webRouter.Group("/api")
	{
		public.GET("/health", func(ginContext *gin.Context) {
			ginContext.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"time":   time.Now().Format(time.RFC3339),
			})
		})
		public.POST("/signup", authHandler.Signup)
		public.POST("/login", authHandler.Login)
		public.POST("/auth/google", authHandler.GoogleLogin)
		public.GET("/keywords", jobHandler.GetMasterKeywords)
	}

	// Serverless Ingest & Scrapper Telemetry Routes
	scraperIngest := webRouter.Group("/api/scraper")
	scraperIngest.Use(middleware.RequireIngestKey())
	{
		scraperIngest.POST("/start", ingestHandler.StartRun)
		scraperIngest.POST("/ingest-raw", ingestHandler.IngestRaw)
		scraperIngest.POST("/finish", ingestHandler.FinishRun)
		scraperIngest.GET("/ats-slugs", ingestHandler.GetATSSlugs)
		scraperIngest.POST("/register-ats-slug", ingestHandler.RegisterATSSlug)
		scraperIngest.GET("/companies", ingestHandler.GetAllCompanyNames)
		scraperIngest.GET("/jobs-without-description", ingestHandler.GetJobsWithoutDescription)
		scraperIngest.POST("/enrich-descriptions", ingestHandler.EnrichJobDescriptions)
	}

	protected := webRouter.Group("/api")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/user/me", authHandler.GetMe)
		protected.GET("/jobs", jobHandler.GetJobs)
		protected.GET("/jobs/matched", matchedJobsHandler.GetMatchedJobs)
		protected.GET("/jobs/match-status", matchedJobsHandler.GetMatchStatus)
		protected.GET("/jobs/:id", matchedJobsHandler.GetMatchedJobByID)
		protected.POST("/jobs/:id/view", jobHandler.MarkJobViewed)
		protected.POST("/jobs/:id/dismiss", jobHandler.DismissJob)
		protected.POST("/jobs/:id/undismiss", jobHandler.UndismissJob)
		protected.POST("/preferences", prefHandler.UpdatePreferences)
		protected.GET("/preferences", prefHandler.GetPreferences)
		protected.POST("/user/profile", prefHandler.UpdateProfile)
		protected.POST("/user/parse-cv", prefHandler.ParseCV)
		protected.POST("/overleaf/config", prefHandler.UpdateOverleafConfig)
		protected.GET("/overleaf/config", prefHandler.GetOverleafConfig)
		protected.POST("/overleaf/sync-profile", prefHandler.SyncProfileToOverleaf)

		protected.POST("/tailor/resume", tailorHandler.TailorResume)
		protected.POST("/tailor/cover-letter", tailorHandler.GenerateCoverLetter)
		protected.POST("/tailor/application-async", tailorHandler.TailorApplicationAsync)
		protected.GET("/tailor/templates", tailorHandler.ListTemplates)
		protected.POST("/tailor/templates/seed", tailorHandler.SeedDefaultTemplates)
		protected.GET("/notifications", notificationsHandler.GetNotifications)
		protected.POST("/notifications/:id/read", notificationsHandler.MarkNotificationAsRead)
		protected.POST("/notifications/read-all", notificationsHandler.MarkAllNotificationsAsRead)
		protected.GET("/notifications/unread-count", notificationsHandler.GetUnreadNotificationsCount)
		protected.PUT("/notifications/fcm-token", notificationsHandler.RegisterFCMToken)

		protected.GET("/resume-versions", versionsHandler.ListResumeVersions)
		protected.GET("/resume-versions/:id/pdf", versionsHandler.GetResumeVersionPDF)
		protected.DELETE("/resume-versions/:id", versionsHandler.DeleteResumeVersion)
		protected.PUT("/resume-versions/:id/default", versionsHandler.SetDefaultResumeVersion)
		protected.GET("/cover-letters", versionsHandler.ListCoverLetterVersions)
		protected.GET("/cover-letters/:id/pdf", versionsHandler.GetCoverLetterPDF)
		protected.DELETE("/cover-letters/:id", versionsHandler.DeleteCoverLetterVersion)
		protected.DELETE("/tailor/jobs/:jobId/documents", versionsHandler.DeleteJobDocuments)

		protected.POST("/applications", appHandler.CreateApplication)
		protected.GET("/applications", appHandler.GetUserApplications)
		protected.PUT("/applications/:id/status", appHandler.UpdateApplicationStatus)
		protected.DELETE("/applications/:id", appHandler.DeleteApplication)
		protected.POST("/user/reset-matches", adminHandler.ResetUserMatches)

		// Master Admin Routes
		protected.GET("/admin/users", adminHandler.GetUsers)
		protected.GET("/admin/whitelisted-emails", adminHandler.GetWhitelistedEmails)
		protected.POST("/admin/whitelisted-emails", adminHandler.AddWhitelistedEmail)
		protected.DELETE("/admin/whitelisted-emails/:id", adminHandler.DeleteWhitelistedEmail)
		protected.GET("/admin/keywords/pending", adminHandler.GetPendingKeywords)
		protected.POST("/admin/keywords/approve", adminHandler.ApproveKeyword)
		protected.GET("/admin/keywords/master", adminHandler.GetMasterKeywordsForAdmin)
		protected.POST("/admin/keywords/manual", adminHandler.AddMasterKeyword)
		protected.DELETE("/admin/keywords/master/:id", adminHandler.DeleteMasterKeyword)
		protected.PUT("/admin/users/:id/ai-matching", adminHandler.ToggleUserAIMatching)
		protected.POST("/admin/reset-all-matches", adminHandler.ResetAndReevaluateMatches)
		protected.GET("/admin/scraper-stats", adminHandler.GetScraperStats)
		protected.GET("/admin/pipeline/status", adminHandler.GetAIPipelineStatus)
		protected.POST("/admin/pipeline/restart", adminHandler.RestartAIPipeline)
	}


	go func() {
		log.Println("[BackgroundMatcher] Startup pass: evaluating unscored jobs for active users.")
		hybridMatchService.EvaluatePendingForAllUsers(context.Background())
	}()

	fmt.Printf("Starting web server on port %s...\n", serverPort)

	serverError := webRouter.Run(":" + serverPort)
	if serverError != nil {
		log.Fatalf("CRITICAL ERROR: The web server crashed. Details: %v", serverError)
	}
}
