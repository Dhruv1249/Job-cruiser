package models

import (
	"time"
)
type User struct {
	ID               string    `json:"id"`
	PrimaryEmail     string    `json:"primary_email"`
	PasswordHash     string    `json:"-"`
	Phone            string    `json:"phone"`
	Location         string    `json:"location"`
	Timezone         string    `json:"timezone"`
	Links            string    `json:"links"` 
	LatexCV          string    `json:"latex_cv"`
	AvatarURL        string    `json:"avatar_url"`
	CVUpdatedAt      *time.Time `json:"cv_updated_at"`
	ParsedExperience string    `json:"parsed_experience"` 
	SubscriptionTier string    `json:"subscription_tier"`
	CreatedAt        time.Time `json:"created_at"`
}
