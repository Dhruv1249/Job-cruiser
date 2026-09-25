package models

import "time"

/*
Job represents a raw job listing entity in the database.
*/
type Job struct {
	ID                 string    `json:"id"`
	CompanyID          string    `json:"company_id"`
	Company            string    `json:"company"`
	Title              string    `json:"title"`
	Location           *string   `json:"location"`
	SalaryMin          *int      `json:"salary_min"`
	SalaryMax          *int      `json:"salary_max"`
	Currency           *string   `json:"currency"`
	SalaryPeriod       *string   `json:"salary_period"`
	EmploymentType     *string   `json:"employment_type"`
	ExperienceRequired *string   `json:"experience_required"`
	JobType            *string   `json:"job_type"`
	IsEasyApply        bool      `json:"is_easy_apply"`
	IsRemote           bool      `json:"is_remote"`
	Source             string    `json:"source"`
	URL                string    `json:"url"`
	PostedDate         *string   `json:"posted_date"`
	Tags               any       `json:"tags"`
	Summary            string    `json:"summary"`
	RawDescription     string    `json:"raw_description"`
	ScrapedAt          time.Time `json:"scraped_at"`
}
