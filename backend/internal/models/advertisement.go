package models

import "time"

type Advertisement struct {
	ID                    string     `json:"id" db:"id"`
	AdvertisementName     string     `json:"advertisementName" db:"advertisement_name"`
	Description           *string    `json:"description" db:"description"`
	AdvertisementCategory string     `json:"advertisementCategory" db:"advertisement_category"`
	MediaDuration         *int       `json:"mediaDuration" db:"media_duration"`
	MediaURL              *string    `json:"mediaUrl" db:"media_url"`
	MediaType             *string    `json:"mediaType" db:"media_type"`
	LoungeGroupName       *string    `json:"loungeGroupName" db:"lounge_group_name"`
	Priority              string     `json:"priority" db:"priority"`
	Version               int        `json:"version" db:"version"`
	ScheduleType          *string    `json:"scheduleType" db:"schedule_type"`
	Frequency             *string    `json:"frequency" db:"frequency"`
	RecurrenceInterval    *int       `json:"recurrenceInterval" db:"recurrence_interval"`
	OccursOnceAt          *time.Time `json:"occursOnceAt" db:"occurs_once_at"`
	OccursEveryInterval   *int       `json:"occursEveryInterval" db:"occurs_every_interval"`
	WeeklyDays            *string    `json:"weeklyDays" db:"weekly_days"`
	MonthlyDayOfMonth     *int       `json:"monthlyDayOfMonth" db:"monthly_day_of_month"`
	MonthlyWeek           *string    `json:"monthlyWeek" db:"monthly_week"`
	MonthlyDay            *string    `json:"monthlyDay" db:"monthly_day"`
	StartDate             *string    `json:"startDate" db:"start_date"`
	EndDate               *string    `json:"endDate" db:"end_date"`
	StartTime             *string    `json:"startTime" db:"start_time"`
	EndTime               *string    `json:"endTime" db:"end_time"`
	MaxIdleLoopDuration   *int       `json:"maxIdleLoopDuration" db:"max_idle_loop_duration"`
	// PlayTimeSlots stores multiple traffic-level/time-slot labels (e.g. ["Peak", "Moderate"]).
	// Legacy single-value PlayTimeSlot is derived from this on the way out.
	PlayTimeSlots         []string   `json:"playTimeSlots" db:"play_time_slots"`
	Status                string     `json:"status" db:"status"`
	CreatedAt             time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time  `json:"updatedAt" db:"updated_at"`
}

type AdvertisementGroup struct {
	ID                 string    `json:"id" db:"id"`
	GroupName          string    `json:"groupName" db:"group_name"`
	Lounges            string    `json:"lounges" db:"lounges"`
	NoOfAdvertisements int       `json:"noOfAdvertisements" db:"no_of_advertisements"`
	CreatedAt          time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updated_at"`
}

type AdvertisementCreateRequest struct {
	AdvertisementName     string   `json:"advertisementName" binding:"required"`
	Description           *string  `json:"description"`
	AdvertisementCategory string   `json:"advertisementCategory" binding:"required"`
	MediaDuration         *int     `json:"mediaDuration"`
	MediaURL              *string  `json:"mediaUrl"`
	MediaType             *string  `json:"mediaType"`
	LoungeGroupName       *string  `json:"loungeGroupName"`
	Priority              string   `json:"priority" binding:"required"`
	ScheduleType          string   `json:"scheduleType" binding:"required"`
	Frequency             *string  `json:"frequency"`
	RecurrenceInterval    *int     `json:"recurrenceInterval"`
	OccursOnceAt          *string  `json:"occursOnceAt"`
	OccursEveryInterval   *int     `json:"occursEveryInterval"`
	WeeklyDays            *string  `json:"weeklyDays"`
	MonthlyDayOfMonth     *int     `json:"monthlyDayOfMonth"`
	MonthlyWeek           *string  `json:"monthlyWeek"`
	MonthlyDay            *string  `json:"monthlyDay"`
	StartDate             *string  `json:"startDate"`
	EndDate               *string  `json:"endDate"`
	StartTime             *string  `json:"startTime"`
	EndTime               *string  `json:"endTime"`
	// Legacy single-slot string support (can be comma-separated).
	PlayTimeSlot          *string  `json:"playTimeSlot"`
	MaxIdleLoopDuration   *int     `json:"maxIdleLoopDuration"`
	// PlayTimeSlots supports multiple traffic-level time slots, e.g. ["Peak", "Moderate"]
	PlayTimeSlots         []string `json:"playTimeSlots"`
	Status                *string  `json:"status"`
}

type AdvertisementGroupCreateRequest struct {
	GroupName string   `json:"groupName" binding:"required"`
	Lounges   []string `json:"lounges" binding:"required"`
}

type TVAdManifestItem struct {
	ID                string    `json:"id"`
	AdvertisementName string    `json:"advertisementName"`
	MediaURL          string    `json:"mediaUrl"`
	MediaType         string    `json:"mediaType"`
	MediaDuration     *int      `json:"mediaDuration,omitempty"`
	Priority          string    `json:"priority"`
	ScheduleType      string    `json:"scheduleType"`
	Frequency         string    `json:"frequency"`
	StartDate         string    `json:"startDate"`
	EndDate           string    `json:"endDate"`
	StartTime         string    `json:"startTime"`
	EndTime           string    `json:"endTime"`
	// PlayTimeSlot is the legacy single-value field (kept for backwards compat with old TV agents)
	PlayTimeSlot      string    `json:"playTimeSlot"`
	// PlayTimeSlots is the new multi-value version — preferred
	PlayTimeSlots     []string  `json:"playTimeSlots"`
	NextPlayAt        string    `json:"nextPlayAt"`
	MediaHash         string    `json:"mediaHash"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type AdvertisementCalculationRate struct {
	TrafficLevel  string    `json:"trafficLevel" db:"traffic_level"`
	CostPerSecond float64   `json:"costPerSecond" db:"cost_per_second"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

type AdvertisementCalculationRateRequest struct {
	CostPerSecond float64 `json:"costPerSecond"`
}

type AdvertisementPlaybackLog struct {
	ID              int64     `json:"id" db:"id"`
	AdvertisementID string    `json:"advertisementId" db:"advertisement_id"`
	AdvertisementName string  `json:"advertisementName" db:"advertisement_name"`
	TrafficLevel    string    `json:"trafficLevel" db:"traffic_level"`
	DurationSeconds int       `json:"durationSeconds" db:"duration_seconds"`
	PlayedAt        time.Time `json:"playedAt" db:"played_at"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
}

type AdvertisementPlaybackLogRequest struct {
	AdvertisementID   string `json:"advertisementId"`
	AdvertisementName string `json:"advertisementName"`
	TrafficLevel      string `json:"trafficLevel"`
	DurationSeconds   int    `json:"durationSeconds"`
	PlayedAt          string `json:"playedAt,omitempty"`
}

type AdvertisementCostReportRow struct {
	AdvertisementID   string  `json:"advertisementId"`
	AdvertisementName string  `json:"advertisementName"`
	TrafficLevel      string  `json:"trafficLevel"`
	PlayCount         int     `json:"playCount"`
	TotalSeconds      int     `json:"totalSeconds"`
	CostPerSecond     float64 `json:"costPerSecond"`
	TotalCost         float64 `json:"totalCost"`
}
