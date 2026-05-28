package models

import "time"

type UserPreference struct {
	UserID            string    `json:"user_id"`
	TargetRoles       any       `json:"target_roles,omitempty"`
	WorkModels        any       `json:"work_models,omitempty"`
	MinSalary         int       `json:"min_salary"`
	Currency          string    `json:"currency"`
	CustomFormAnswers any       `json:"custom_form_answers,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CVTemplate struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	LatexCode string    `json:"latex_code"`
	CreatedAt time.Time `json:"created_at"`
}

type ResumeVersion struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TemplateID  *int      `json:"template_id,omitempty"` // Pointer because it can be null
	Label       string    `json:"label"`
	LatexSource string    `json:"latex_source"`
	PageLimit   int       `json:"page_limit"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
}

type AIPrompt struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	PromptType string    `json:"prompt_type"`
	Label      string    `json:"label"`
	Template   string    `json:"template"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type JobFilter struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	KeywordWhitelist  any       `json:"keyword_whitelist,omitempty"`
	KeywordBlacklist  any       `json:"keyword_blacklist,omitempty"`
	ExcludedCompanies any       `json:"excluded_companies,omitempty"`
	MinScore          int       `json:"min_score"`
	MaxExperienceYrs  int       `json:"max_experience_yrs"`
	PreferredSources  any       `json:"preferred_sources,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}
