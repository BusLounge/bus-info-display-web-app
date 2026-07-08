package models

type ConflictCheckRequest struct {
	LoungeGroupName        string  `json:"loungeGroupName"`
	ScheduleType           string  `json:"scheduleType"`
	Frequency              *string `json:"frequency"`
	OccursOnceAt           *string `json:"occursOnceAt"`
	StartTime              *string `json:"startTime"`
	EndTime                *string `json:"endTime"`
	StartDate              *string `json:"startDate"`
	EndDate                *string `json:"endDate"`
	WeeklyDays             *string `json:"weeklyDays"`
	MonthlyDayOfMonth      *int    `json:"monthlyDayOfMonth"`
	RecurrenceInterval     *int    `json:"recurrenceInterval"`
	ExcludeAdvertisementID *string `json:"excludeAdvertisementId"`
}

type ConflictResponse struct {
	HasConflict      bool     `json:"hasConflict"`
	ConflictingAds   []string `json:"conflictingAds"`
	ConflictMessage  string   `json:"conflictMessage"`
	ConflictTimeSlot string   `json:"conflictTimeSlot"`
	AffectedLounges  []string `json:"affectedLounges"`
	SuggestedAction  string   `json:"suggestedAction"`
}
