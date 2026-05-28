package models

import (
	"time"
)

// User represents a user identity in the system.
type User struct {
	ID               string     `json:"id"`
	FullName         string     `json:"full_name"`
	PrimaryEmail     string     `json:"primary_email"`
	PasswordHash     string     `json:"-"` // The hyphen ensures the password hash is NEVER sent to the Flutter frontend
	Phone            string     `json:"phone,omitempty"`
	Location         string     `json:"location,omitempty"`
	Timezone         string     `json:"timezone"`
	Links            any        `json:"links,omitempty"` // Maps to JSONB
	LatexCV          string     `json:"latex_cv,omitempty"`
	AvatarURL        string     `json:"avatar_url,omitempty"`
	CVUpdatedAt      *time.Time `json:"cv_updated_at,omitempty"`
	ParsedExperience any        `json:"parsed_experience,omitempty"` // Maps to JSONB
	CreatedAt        time.Time  `json:"created_at"`
}
