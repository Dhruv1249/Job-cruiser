package models

import "time"

type Application struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	JobID            string     `json:"job_id"`
	ResumeVersionID  *string    `json:"resume_version_id,omitempty"`
	Status           string     `json:"status"`
	GeneratedAnswers any        `json:"generated_answers,omitempty"`
	CoverLetter      string     `json:"cover_letter,omitempty"`
	Notes            string     `json:"notes,omitempty"`
	FollowUpAt       *time.Time `json:"follow_up_at,omitempty"`
	AppliedAt        *time.Time `json:"applied_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type InterviewRound struct {
	ID              string     `json:"id"`
	ApplicationID   string     `json:"application_id"`
	RoundNumber     int        `json:"round_number"`
	RoundType       string     `json:"round_type"`
	ScheduledAt     *time.Time `json:"scheduled_at,omitempty"`
	Outcome         string     `json:"outcome,omitempty"`
	InterviewerName string     `json:"interviewer_name,omitempty"`
	Notes           string     `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ColdEmail struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	CompanyID     string     `json:"company_id"`
	JobID         *string    `json:"job_id,omitempty"`
	TargetEmail   string     `json:"target_email"`
	Subject       string     `json:"subject"`
	Body          string     `json:"body"`
	Status        string     `json:"status"`
	OpenedAt      *time.Time `json:"opened_at,omitempty"`
	ReplyReceived bool       `json:"reply_received"`
	ThreadID      string     `json:"thread_id,omitempty"`
	ScheduledFor  *time.Time `json:"scheduled_for,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type FollowUp struct {
	ID           string     `json:"id"`
	ColdEmailID  string     `json:"cold_email_id"`
	Body         string     `json:"body"`
	Status       string     `json:"status"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
}
