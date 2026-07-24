package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("DB connect failed: %v", err)
	}
	defer pool.Close()

	deleteQuery := `
		DELETE FROM user_job_matches 
		WHERE job_id IN (
			SELECT id FROM jobs 
			WHERE LOWER(title) LIKE '%senior%' 
			   OR LOWER(title) LIKE '%sr.%'
			   OR LOWER(title) LIKE '%sr %'
			   OR LOWER(title) LIKE '%lead%'
			   OR LOWER(title) LIKE '%principal%'
			   OR LOWER(title) LIKE '%staff%'
			   OR LOWER(title) LIKE '%architect%'
			   OR LOWER(title) LIKE '%director%'
			   OR LOWER(title) LIKE '%head of%'
			   OR LOWER(title) LIKE '%vp%'
			   OR LOWER(title) LIKE '%manager%'
		);
	`

	res, err := pool.Exec(ctx, deleteQuery)
	if err != nil {
		log.Fatalf("Failed to delete senior role matches: %v", err)
	}

	log.Printf("Purged %d senior/lead/staff matches from user_job_matches.", res.RowsAffected())
	fmt.Println("Senior role cleanup complete!")
}
