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

	var totalJobs int
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM jobs;").Scan(&totalJobs)

	var emptyDescCount int
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM jobs WHERE raw_desc IS NULL OR TRIM(raw_desc) = '';").Scan(&emptyDescCount)

	var sampleTitle, sampleCompany, sampleDesc string
	_ = pool.QueryRow(ctx, "SELECT title, COALESCE(c.name, ''), LEFT(raw_desc, 100) FROM jobs j LEFT JOIN companies c ON j.company_id = c.id WHERE raw_desc IS NOT NULL AND TRIM(raw_desc) != '' LIMIT 1;").Scan(&sampleTitle, &sampleCompany, &sampleDesc)

	fmt.Printf("Total Jobs: %d\n", totalJobs)
	fmt.Printf("Jobs with Empty raw_desc: %d\n", emptyDescCount)
	fmt.Printf("Sample Job with Description: %s at %s -> %s...\n", sampleTitle, sampleCompany, sampleDesc)
}
