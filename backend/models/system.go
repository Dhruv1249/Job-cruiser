package models

import "time"

type SystemCommand struct {
	ID        int       `json:"id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ScraperRun struct {
	ID           string     `json:"id"`
	CommandID    *int       `json:"command_id,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       string     `json:"status"`
	JobsAdded    int        `json:"jobs_added"`
	SourcesHit   any        `json:"sources_hit,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

type SystemLog struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Metadata  any       `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
