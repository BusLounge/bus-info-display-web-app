package models

import "time"

type LoungeAd struct {
	ID                string    `json:"id" db:"id"`
	LoungeID          *string   `json:"loungeId,omitempty" db:"lounge_id"`
	AdvertisementName string    `json:"advertisementName" db:"advertisement_name"`
	MediaURL          string    `json:"mediaUrl" db:"media_url"`
	MediaType         string    `json:"mediaType" db:"media_type"`
	DurationSeconds   int       `json:"durationSeconds" db:"duration_seconds"`
	Priority          string    `json:"priority" db:"priority"`
	IsActive          bool      `json:"isActive" db:"is_active"`
	IsDefaultForAll   bool      `json:"isDefaultForAll" db:"is_default_for_all"`
	CreatedAt         time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time `json:"updatedAt" db:"updated_at"`
}

type LoungeAdRequest struct {
	LoungeID          *string `json:"loungeId"`
	AdvertisementName string  `json:"advertisementName"`
	MediaURL          string  `json:"mediaUrl"`
	MediaType         string  `json:"mediaType"`
	DurationSeconds   int     `json:"durationSeconds"`
	Priority          string  `json:"priority"`
	IsActive          *bool   `json:"isActive"`
	IsDefaultForAll   *bool   `json:"isDefaultForAll"`
}

type LoungeAdSlotSummary struct {
	ScheduleWindowSeconds int                `json:"scheduleWindowSeconds"`
	AdWindowSeconds       int                `json:"adWindowSeconds"`
	BookedSeconds         int                `json:"bookedSeconds"`
	RemainingSeconds      int                `json:"remainingSeconds"`
	TimeSlots             []LoungeAdTimeSlot `json:"timeSlots,omitempty"`
	AvailableSlots        []LoungeAdTimeSlot `json:"availableSlots,omitempty"`
	BookedByScheduleType  map[string]int     `json:"bookedByScheduleType,omitempty"`
	EffectiveLoungeGroups []string           `json:"effectiveLoungeGroups,omitempty"`
}

type LoungeAdTimeSlot struct {
	Type            string `json:"type"`
	Label           string `json:"label"`
	StartSecond     int    `json:"startSecond"`
	EndSecond       int    `json:"endSecond"`
	DurationSeconds int    `json:"durationSeconds"`
	Interactive     bool   `json:"interactive"`
}
