package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Dhruv1249/Job-cruiser/backend/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set in environment or .env")
	}

	mistralKey := os.Getenv("MISTRAL_API_KEY")
	if mistralKey == "" {
		log.Println("WARNING: MISTRAL_API_KEY is not set!")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Clearing user_job_matches table...")
	res, err := pool.Exec(ctx, "DELETE FROM user_job_matches;")
	if err != nil {
		log.Fatalf("Failed to delete user_job_matches: %v", err)
	}
	log.Printf("Cleared %d existing user_job_matches records.", res.RowsAffected())

	log.Println("Resetting jobs.ai_evaluated flag to false...")
	res, err = pool.Exec(ctx, "UPDATE jobs SET ai_evaluated = false;")
	if err != nil {
		log.Fatalf("Failed to reset jobs.ai_evaluated: %v", err)
	}
	log.Printf("Reset %d jobs to unevaluated.", res.RowsAffected())

	if mistralKey != "" {
		log.Println("Starting fresh evaluation pass using updated hard-location filter prompt...")
		matcher := &services.MistralBatchMatchService{
			DB:         pool,
			MistralKey: mistralKey,
			HTTPClient: &http.Client{Timeout: 600 * time.Second},
		}

		matcher.EvaluatePendingForAllUsers(ctx)
		log.Println("Fresh evaluation pass completed successfully!")
	} else {
		log.Println("Skipping evaluation pass as MISTRAL_API_KEY is empty.")
	}

	fmt.Println("Done!")
}
