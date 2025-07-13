package models

import "time"

type SiteSetting struct {
	ID          int       `json:"id" db:"id"`
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	Description *string   `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type UpdateSettingRequest struct {
	Value string `json:"value" binding:"required"`
}

type SiteSettingsResponse struct {
	Settings []SiteSetting `json:"settings"`
}