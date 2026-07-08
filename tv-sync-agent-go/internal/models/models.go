package models

import (
	"strings"
	"time"
)

const (
	TVPurposeBoth     = "both"
	TVPurposeSchedule = "schedule"
	TVPurposeAds      = "ads"
)

type Config struct {
	ServerURL             string `json:"serverUrl"`
	LoungeID              string `json:"loungeId"`
	LoungeGroup           string `json:"loungeGroup"`
	Language              string `json:"language"`
	DisplayMode           string `json:"displayMode"`
	LayoutMode            string `json:"layoutMode"`
	SyncFrequencySeconds  int    `json:"syncFrequencySeconds"`
	DisplayResolution     string `json:"displayResolution"`
	TVPurpose             string `json:"tvPurpose"`
	ScheduleIntervalCron  string `json:"scheduleIntervalCron"`
	AdsIntervalCron       string `json:"adsIntervalCron"`
	BroadcastIntervalCron string `json:"broadcastIntervalCron"`
	LocalBridgePort       int    `json:"localBridgePort"`
	RequestTimeoutMs      int    `json:"requestTimeoutMs"`
	DownloadTimeoutMs     int    `json:"downloadTimeoutMs"`
	StoreDir              string `json:"storeDir"`
}

func (c Config) ScheduleEnabled() bool {
	p := strings.ToLower(strings.TrimSpace(c.TVPurpose))
	return p == TVPurposeBoth || p == TVPurposeSchedule
}

func (c Config) AdsEnabled() bool {
	p := strings.ToLower(strings.TrimSpace(c.TVPurpose))
	return p == TVPurposeBoth || p == TVPurposeAds
}

type ScheduleSnapshot struct {
	LoungeID      string      `json:"loungeId"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	SourceURL     string      `json:"sourceUrl"`
	DeparturesRaw interface{} `json:"departuresRaw"`
	ArrivalsRaw   interface{} `json:"arrivalsRaw"`
}

type Advertisement struct {
	ID                string     `json:"id"`
	AdvertisementName string     `json:"advertisementName"`
	MediaURL          *string    `json:"mediaUrl"`
	MediaType         *string    `json:"mediaType"`
	LoungeGroupName   *string    `json:"loungeGroupName"`
	Priority          string     `json:"priority"`
	Status            string     `json:"status"`
	UpdatedAt         *time.Time `json:"updatedAt"`
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
	PlayTimeSlot      string    `json:"playTimeSlot"`
	PlayTimeSlots     []string  `json:"playTimeSlots,omitempty"`
	NextPlayAt        string    `json:"nextPlayAt"`
	MediaHash         string    `json:"mediaHash"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type AdManifestItem struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MediaURL      string   `json:"mediaUrl"`
	MediaType     string   `json:"mediaType"`
	MediaDuration *int     `json:"mediaDuration,omitempty"`
	LocalFile     string   `json:"localFile"`
	MediaHash     string   `json:"mediaHash"`
	SourceVersion string   `json:"sourceVersion"`
	Priority      string   `json:"priority"`
	ScheduleType  string   `json:"scheduleType"`
	Frequency     string   `json:"frequency"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	StartTime     string   `json:"startTime"`
	EndTime       string   `json:"endTime"`
	PlayTimeSlot  string   `json:"playTimeSlot"`
	PlayTimeSlots []string `json:"playTimeSlots,omitempty"`
	NextPlayAt    string   `json:"nextPlayAt"`
}

type AdsSnapshot struct {
	LoungeGroup string           `json:"loungeGroup"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	SourceURL   string           `json:"sourceUrl"`
	Items       []AdManifestItem `json:"items"`
}

type BroadcastMessage struct {
	ID                     string     `json:"id"`
	Message                string     `json:"message"`
	Priority               string     `json:"priority"`
	DisplayDurationSeconds int        `json:"displayDurationSeconds"`
	FrequencySeconds       int        `json:"frequencySeconds"`
	StartAt                time.Time  `json:"startAt"`
	EndAt                  *time.Time `json:"endAt,omitempty"`
	IsActive               bool       `json:"isActive"`
	ShowOnLoungeTV         bool       `json:"showOnLoungeTV"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type BroadcastSnapshot struct {
	UpdatedAt time.Time          `json:"updatedAt"`
	SourceURL string             `json:"sourceUrl"`
	Items     []BroadcastMessage `json:"items"`
}

type LoungeAd struct {
	ID                string  `json:"id"`
	LoungeID          *string `json:"loungeId,omitempty"`
	AdvertisementName string  `json:"advertisementName"`
	MediaURL          string  `json:"mediaUrl"`
	MediaType         string  `json:"mediaType"`
	DurationSeconds   int     `json:"durationSeconds"`
	Priority          string  `json:"priority"`
	IsActive          bool    `json:"isActive"`
	IsDefaultForAll   bool    `json:"isDefaultForAll"`
}

type LoungeAdsSnapshot struct {
	LoungeID  string     `json:"loungeId"`
	UpdatedAt time.Time  `json:"updatedAt"`
	SourceURL string     `json:"sourceUrl"`
	Items     []LoungeAd `json:"items"`
}

type LoungeAdCreateRequest struct {
	AdvertisementName string `json:"advertisementName"`
	MediaURL          string `json:"mediaUrl"`
	MediaType         string `json:"mediaType"`
	DurationSeconds   int    `json:"durationSeconds"`
	Priority          string `json:"priority"`
	IsActive          *bool  `json:"isActive"`
	IsDefaultForAll   *bool  `json:"isDefaultForAll"`
}

type LoungeAdUpdateRequest struct {
	AdvertisementName string `json:"advertisementName"`
	MediaURL          string `json:"mediaUrl"`
	MediaType         string `json:"mediaType"`
	IsActive          *bool  `json:"isActive"`
	IsDefaultForAll   *bool  `json:"isDefaultForAll"`
}

type LoungeAdTimeSlot struct {
	Type            string `json:"type"`
	Label           string `json:"label"`
	StartSecond     int    `json:"startSecond"`
	EndSecond       int    `json:"endSecond"`
	DurationSeconds int    `json:"durationSeconds"`
	Interactive     bool   `json:"interactive"`
}

type LoungeAdLocalSlotSummary struct {
	ScheduleWindowSeconds int                `json:"scheduleWindowSeconds"`
	AdWindowSeconds       int                `json:"adWindowSeconds"`
	BookedSeconds         int                `json:"bookedSeconds"`
	RemainingSeconds      int                `json:"remainingSeconds"`
	TimeSlots             []LoungeAdTimeSlot `json:"timeSlots,omitempty"`
	AvailableSlots        []LoungeAdTimeSlot `json:"availableSlots,omitempty"`
	FallbackRequired      bool               `json:"fallbackRequired"`
	FallbackSeconds       int                `json:"fallbackSeconds"`
}

type AgentStatus struct {
	StartedAt            time.Time `json:"startedAt"`
	TVPurpose            string    `json:"tvPurpose"`
	Language             string    `json:"language"`
	DisplayMode          string    `json:"displayMode"`
	LayoutMode           string    `json:"layoutMode"`
	SyncFrequencySeconds int       `json:"syncFrequencySeconds"`
	DisplayResolution    string    `json:"displayResolution"`
	BroadcastsEnabled    bool      `json:"broadcastsEnabled"`
	ScheduleEnabled      bool      `json:"scheduleEnabled"`
	AdsEnabled           bool      `json:"adsEnabled"`
	LastScheduleSync     time.Time `json:"lastScheduleSync"`
	LastAdsSync          time.Time `json:"lastAdsSync"`
	LastBroadcastSync    time.Time `json:"lastBroadcastSync"`
	LastLoungeAdsSync    time.Time `json:"lastLoungeAdsSync"`
	LastScheduleErr      string    `json:"lastScheduleError,omitempty"`
	LastAdsErr           string    `json:"lastAdsError,omitempty"`
	LastBroadcastErr     string    `json:"lastBroadcastError,omitempty"`
	LastLoungeAdsErr     string    `json:"lastLoungeAdsError,omitempty"`
	AdsCount             int       `json:"adsCount"`
	BroadcastCount       int       `json:"broadcastCount"`
	LoungeAdsCount       int       `json:"loungeAdsCount"`
	SchedulePath         string    `json:"schedulePath"`
	AdsManifestPath      string    `json:"adsManifestPath"`
	BroadcastPath        string    `json:"broadcastPath"`
	LoungeAdsPath        string    `json:"loungeAdsPath"`
}
