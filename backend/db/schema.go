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
		google_id TEXT,
		auth_provider VARCHAR(50) DEFAULT 'local',
		phone TEXT,
		location TEXT,
		timezone VARCHAR(50) DEFAULT 'Asia/Kolkata',
		links JSONB DEFAULT '{}'::jsonb,
		latex_cv TEXT,
		avatar_url TEXT,
		cv_updated_at TIMESTAMP,
		parsed_experience JSONB DEFAULT '[]'::jsonb,
		subscription_tier VARCHAR(20) DEFAULT 'free',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`ALTER TABLE users ADD COLUMN IF NOT EXISTS is_master_admin BOOLEAN DEFAULT false;`,
	`ALTER TABLE users ADD COLUMN IF NOT EXISTS ai_matching_enabled BOOLEAN DEFAULT false;`,

	`CREATE TABLE IF NOT EXISTS whitelisted_emails (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email TEXT UNIQUE NOT NULL,
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS cv_templates (
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		latex_code TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS user_preferences (
		user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		full_name TEXT,
		target_roles JSONB DEFAULT '[]'::jsonb,
		work_models JSONB DEFAULT '[]'::jsonb,
		min_salary INTEGER DEFAULT 0,
		currency VARCHAR(10) DEFAULT 'USD',
		master_cv_text TEXT,
		bio_experience_text TEXT,
		custom_form_answers JSONB DEFAULT '{}'::jsonb,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS master_keywords (
		id SERIAL PRIMARY KEY,
		keyword VARCHAR(100) UNIQUE NOT NULL,
		category VARCHAR(50) DEFAULT 'general',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS pending_keyword_suggestions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		keyword VARCHAR(100) NOT NULL,
		discovered_from_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS user_overleaf_config (
		user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		deployment_url TEXT NOT NULL,
		github_username TEXT NOT NULL,
		github_repo_name TEXT NOT NULL,
		encrypted_access_token TEXT NOT NULL,
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
		seniority VARCHAR(50),
		summary TEXT,
		scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`,

	`CREATE TABLE IF NOT EXISTS user_job_matches (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
		match_score INTEGER DEFAULT 0,
		match_reasons JSONB DEFAULT '[]'::jsonb,
		suggested_action VARCHAR(30) DEFAULT 'review',
		is_dismissed BOOLEAN DEFAULT false,
		is_ai_matched BOOLEAN DEFAULT false,
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

	// 6. Incremental schema migrations (idempotent ALTER TABLE statements)
	`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS ai_evaluated BOOLEAN DEFAULT false;`,
	`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS ai_evaluated_at TIMESTAMP;`,
	`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS summary TEXT;`,
	`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS seniority VARCHAR(50);`,

	`ALTER TABLE user_job_matches ADD COLUMN IF NOT EXISTS match_score INTEGER DEFAULT 0;`,
	`ALTER TABLE user_job_matches ADD COLUMN IF NOT EXISTS match_reasoning TEXT;`,
	`ALTER TABLE user_job_matches ADD COLUMN IF NOT EXISTS seniority VARCHAR(50);`,
	`ALTER TABLE user_job_matches ADD COLUMN IF NOT EXISTS tech_stack JSONB DEFAULT '[]'::jsonb;`,
	`ALTER TABLE user_job_matches ADD COLUMN IF NOT EXISTS ai_model VARCHAR(100) DEFAULT 'mistral-small-2506';`,
	`ALTER TABLE user_job_matches ADD COLUMN IF NOT EXISTS evaluated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;`,

	`ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS target_industries JSONB DEFAULT '[]'::jsonb;`,
	`ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS target_locations JSONB DEFAULT '["India (On-site & Hybrid)", "India (Remote)", "Global Remote"]'::jsonb;`,

	`CREATE TABLE IF NOT EXISTS user_job_views (
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
		viewed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, job_id)
	);`,

	`CREATE INDEX IF NOT EXISTS idx_user_job_views_lookup ON user_job_views(user_id, job_id);`,

	`CREATE INDEX IF NOT EXISTS idx_jobs_ai_evaluated ON jobs(ai_evaluated) WHERE ai_evaluated = false;`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_source ON jobs(source);`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_scraped_at ON jobs(scraped_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_user_job_matches_score ON user_job_matches(user_id, match_score DESC);`,

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

	scraperKeywords := []string{
		"backend engineer", "backend developer", "software engineer backend", "backend intern",
		"junior backend engineer", "node.js developer", "express.js developer", "api developer",
		"rest api developer", "microservices engineer", "distributed systems engineer",
		"cloud backend engineer", "server-side developer", "platform backend engineer",
		"full stack engineer", "full stack developer", "fullstack developer", "mern stack developer",
		"next.js developer", "react developer", "frontend engineer", "javascript developer",
		"typescript developer", "software developer", "web developer", "cloud engineer",
		"cloud developer", "cloud infrastructure engineer", "cloud software engineer",
		"aws engineer", "aws developer", "gcp engineer", "google cloud engineer",
		"cloud platform engineer", "devops engineer", "devops intern", "junior devops engineer",
		"platform engineer", "infrastructure engineer", "site reliability engineer", "sre intern",
		"build engineer", "release engineer", "ci/cd engineer", "kubernetes engineer",
		"docker engineer", "container platform engineer", "cloud native engineer",
		"kubernetes developer", "systems engineer", "systems programmer", "systems software engineer",
		"kernel engineer", "kernel developer", "operating systems engineer", "embedded systems engineer",
		"low level software engineer", "firmware engineer", "rust developer", "rust systems engineer",
		"rust backend engineer", "rust systems developer", "c developer", "c systems programmer",
		"systems research engineer", "ai infrastructure engineer", "ai platform engineer",
		"ml infrastructure engineer", "mlops engineer", "ml platform engineer", "ai backend engineer",
		"ai systems engineer", "genai infrastructure engineer", "python developer",
		"python backend engineer", "fastapi developer", "python software engineer",
		"observability engineer", "monitoring engineer", "reliability engineer", "automation engineer",
		"infrastructure automation engineer", "devops automation engineer", "forward deployed engineer",
		"forward deployed software engineer", "founding engineer", "founding software engineer",
		"founding backend engineer", "founding full stack engineer", "early stage engineer",
		"startup software engineer", "startup backend engineer", "software engineer i",
		"graduate software engineer", "new grad software engineer", "software engineer intern",
		"software development engineer", "sde i", "graduate backend engineer", "graduate cloud engineer",
		"graduate devops engineer", "entry level software engineer", "infrastructure software engineer",
		"platform software engineer", "cloud platform developer", "infrastructure developer",
		"reliability platform engineer", "backend engineer new grad", "cloud engineer entry level",
		"devops engineer graduate", "platform engineer new grad", "sre new grad",
		"kubernetes engineer junior", "aws backend engineer", "software engineer cloud",
		"software engineer infrastructure", "software engineer",
	}
	for _, kw := range scraperKeywords {
		_, _ = databasePool.Exec(ctx, "INSERT INTO master_keywords (keyword, category) VALUES ($1, 'scraper') ON CONFLICT (keyword) DO NOTHING;", kw)
	}

	return nil
}
