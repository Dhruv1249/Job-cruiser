package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var schemaQueries = []string{
	// 1. Core Identity & Templates
	`CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		primary_email TEXT UNIQUE NOT NULL,
		password_hash TEXT,
		phone TEXT,
		location TEXT,
		timezone VARCHAR(50) DEFAULT 'Asia/Kolkata',
		links JSONB DEFAULT '{}'::jsonb,
		latex_cv TEXT,
		avatar_url TEXT,
		cv_updated_at TIMESTAMP,
		parsed_experience JSONB DEFAULT '[]'::jsonb,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS cv_templates (
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		latex_code TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	// 2. User Dependencies
	`CREATE TABLE IF NOT EXISTS user_preferences (
		user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		full_name TEXT,
		target_roles JSONB DEFAULT '[]'::jsonb,
		work_models JSONB DEFAULT '[]'::jsonb,
		min_salary INTEGER DEFAULT 0,
		currency VARCHAR(10) DEFAULT 'USD',
		custom_form_answers JSONB DEFAULT '{}'::jsonb,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS resume_versions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		template_id INTEGER REFERENCES cv_templates(id) ON DELETE SET NULL,
		label TEXT NOT NULL,
		latex_source TEXT NOT NULL,
		page_limit INTEGER DEFAULT 1,
		is_default BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS ai_prompts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		prompt_type VARCHAR(50) NOT NULL,
		label TEXT NOT NULL,
		template TEXT NOT NULL,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS job_filters (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		keyword_whitelist JSONB DEFAULT '[]'::jsonb,
		keyword_blacklist JSONB DEFAULT '[]'::jsonb,
		excluded_companies JSONB DEFAULT '[]'::jsonb,
		min_score INTEGER DEFAULT 5,
		max_experience_yrs INTEGER DEFAULT 2,
		preferred_sources JSONB DEFAULT '[]'::jsonb,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS email_accounts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		email TEXT NOT NULL,
		provider VARCHAR(20) DEFAULT 'gmail',
		auth_type VARCHAR(20) DEFAULT 'oauth',
		credentials TEXT NOT NULL,
		is_default BOOLEAN DEFAULT false,
		daily_limit INTEGER DEFAULT 20,
		sent_today INTEGER DEFAULT 0,
		last_sent_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS notifications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		title VARCHAR(100) NOT NULL,
		message TEXT NOT NULL,
		is_read BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	// 3. The Market
	`CREATE TABLE IF NOT EXISTS companies (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL,
		domain TEXT UNIQUE,
		poc_name TEXT,
		poc_title TEXT,
		poc_email TEXT,
		description TEXT,
		company_size VARCHAR(20),
		industry TEXT,
		funding_stage TEXT,
		hq_location TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	

	`CREATE TABLE IF NOT EXISTS jobs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		location TEXT,
		salary_min INTEGER,
		salary_max INTEGER,
		currency VARCHAR(10) DEFAULT 'USD',
		experience_required TEXT,
		job_type VARCHAR(20),
		is_easy_apply BOOLEAN DEFAULT false,
		is_remote BOOLEAN DEFAULT false,
		source VARCHAR(50) NOT NULL,
		url TEXT UNIQUE NOT NULL, -- Added UNIQUE
		posted_date TEXT,
		tags JSONB DEFAULT '[]'::jsonb,
		raw_desc TEXT,
		scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS user_job_matches (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
		match_score INTEGER DEFAULT 0,
		match_reasons JSONB DEFAULT '[]'::jsonb,
		is_dismissed BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, job_id)
	);`,

	// 4. The Pipeline
	`CREATE TABLE IF NOT EXISTS applications (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
		resume_version_id UUID REFERENCES resume_versions(id) ON DELETE SET NULL,
		status VARCHAR(50) DEFAULT 'bookmarked',
		generated_answers JSONB DEFAULT '{}'::jsonb,
		cover_letter TEXT,
		notes TEXT,
		follow_up_at TIMESTAMP,
		applied_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS interview_rounds (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		application_id UUID REFERENCES applications(id) ON DELETE CASCADE,
		round_number INTEGER NOT NULL,
		round_type VARCHAR(50),
		scheduled_at TIMESTAMP,
		outcome VARCHAR(20),
		interviewer_name TEXT,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS cold_emails (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		company_id UUID REFERENCES companies(id) ON DELETE CASCADE,
		job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
		target_email TEXT NOT NULL,
		subject TEXT,
		body TEXT,
		status VARCHAR(20) DEFAULT 'draft',
		opened_at TIMESTAMP,
		reply_received BOOLEAN DEFAULT false,
		thread_id TEXT,
		scheduled_for TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS follow_ups (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		cold_email_id UUID REFERENCES cold_emails(id) ON DELETE CASCADE,
		body TEXT,
		status VARCHAR(20) DEFAULT 'pending',
		scheduled_for TIMESTAMP,
		sent_at TIMESTAMP
	);`,

	// 5. Telemetry & Commands
	`CREATE TABLE IF NOT EXISTS system_commands (
		id SERIAL PRIMARY KEY,
		command VARCHAR(50) NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS scraper_runs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		command_id INTEGER REFERENCES system_commands(id) ON DELETE SET NULL,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		status VARCHAR(20) DEFAULT 'running',
		jobs_added INTEGER DEFAULT 0,
		sources_hit JSONB DEFAULT '[]'::jsonb,
		error_message TEXT
	);`,

	`CREATE TABLE IF NOT EXISTS system_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		source VARCHAR(50),
		level VARCHAR(10),
		message TEXT,
		metadata JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,
}

// InitSchema executes the queries in sequence.
func InitSchema(databasePool *pgxpool.Pool) error {
	ctx := context.Background()

	for i, query := range schemaQueries {
		_, err := databasePool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed on query index %d: %v\nQuery: %s", i, err, query)
		}
	}
	return nil
}
