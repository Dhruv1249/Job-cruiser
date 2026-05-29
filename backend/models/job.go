package models

import "time"

type Job struct {
	ID                 string    `json:"id"`
	CompanyID          string    `json:"company_id"`
	Title              string    `json:"title"`
	Location           *string   `json:"location"` // Pointer because it might be null
	SalaryMin          *int      `json:"salary_min"`
	SalaryMax          *int      `json:"salary_max"`
	Currency           *string   `json:"currency"`
	ExperienceRequired *string   `json:"experience_required"`
	JobType            *string   `json:"job_type"`
	IsEasyApply        bool      `json:"is_easy_apply"`
	IsRemote           bool      `json:"is_remote"`
	Source             string    `json:"source"`
	URL                string    `json:"url"`
	PostedDate         *string   `json:"posted_date"`
	Tags               any       `json:"tags"` // or json.RawMessage
	ScrapedAt          time.Time `json:"scraped_at"`
}
