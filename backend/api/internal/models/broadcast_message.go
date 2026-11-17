package models

import "time"

type BroadcastMessage struct {
	ID                     string     `json:"id" db:"id"`
	Message                string     `json:"message" db:"message"`
	Priority               string     `json:"priority" db:"priority"`
	DisplayDurationSeconds int        `json:"displayDurationSeconds" db:"display_duration_seconds"`
	FrequencySeconds       int        `json:"frequencySeconds" db:"frequency_seconds"`
	StartAt                time.Time  `json:"startAt" db:"start_at"`
	EndAt                  *time.Time `json:"endAt,omitempty" db:"end_at"`
	IsActive               bool       `json:"isActive" db:"is_active"`
	ShowOnLoungeTV         bool       `json:"showOnLoungeTV" db:"show_on_lounge_tv"`
	CreatedAt              time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt              time.Time  `json:"updatedAt" db:"updated_at"`
}

type BroadcastMessageRequest struct {
	Message                string  `json:"message"`
	Priority               string  `json:"priority"`
	DisplayDurationSeconds int     `json:"displayDurationSeconds"`
	FrequencySeconds       int     `json:"frequencySeconds"`
	StartAt                *string `json:"startAt"`
	EndAt                  *string `json:"endAt"`
	IsActive               *bool   `json:"isActive"`
	ShowOnLoungeTV         *bool   `json:"showOnLoungeTV"`
}
