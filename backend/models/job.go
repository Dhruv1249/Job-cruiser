package models

import (
	"time"
)

// Job represents a scraped job listing.
type Job struct {
	ID                 string    `json:"id"`
	CompanyID          string    `json:"company_id"`
	Title              string    `json:"title"`
	Location           string    `json:"location,omitempty"`
	Salary             string    `json:"salary,omitempty"`
	ExperienceRequired string    `json:"experience_required,omitempty"`
	JobType            string    `json:"job_type,omitempty"`
	IsEasyApply        bool      `json:"is_easy_apply"`
	IsRemote           bool      `json:"is_remote"`
	Source             string    `json:"source"`
	URL                string    `json:"url"`
	PostedDate         string    `json:"posted_date,omitempty"`
	Tags               any       `json:"tags,omitempty"` // Maps to JSONB array
	RawDesc            string    `json:"-"` // Hidden from standard API lists to save huge amounts of data transfer
	Score              int       `json:"score"`
	ScrapedAt          time.Time `json:"scraped_at"`
}
