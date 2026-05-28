package models

import "time"

type Company struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Domain       string    `json:"domain,omitempty"`
	POCName      string    `json:"poc_name,omitempty"`
	POCTitle     string    `json:"poc_title,omitempty"`
	POCEmail     string    `json:"poc_email,omitempty"`
	Description  string    `json:"description,omitempty"`
	CompanySize  string    `json:"company_size,omitempty"`
	Industry     string    `json:"industry,omitempty"`
	FundingStage string    `json:"funding_stage,omitempty"`
	HQLocation   string    `json:"hq_location,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type EmailAccount struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Email       string     `json:"email"`
	Provider    string     `json:"provider"`
	AuthType    string     `json:"auth_type"`
	Credentials string     `json:"-"` // Hidden from frontend
	IsDefault   bool       `json:"is_default"`
	DailyLimit  int        `json:"daily_limit"`
	SentToday   int        `json:"sent_today"`
	LastSentAt  *time.Time `json:"last_sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
