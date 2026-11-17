package services

import (
	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AdvertisementService struct {
	repo *database.AdvertisementRepository
}

// Grace window for one-time ads: if the scheduled time is within this window of now, still consider it playable
const nextPlayAtGraceWindow = 5 * time.Minute

func NewAdvertisementService(repo *database.AdvertisementRepository) *AdvertisementService {
	return &AdvertisementService{repo: repo}
}

func (s *AdvertisementService) EnsureCalculationSchema() error {
	return s.repo.EnsureCalculationSchema()
}

func (s *AdvertisementService) GetCalculationRates() ([]models.AdvertisementCalculationRate, error) {
	return s.repo.GetCalculationRates()
}

func (s *AdvertisementService) UpsertCalculationRate(trafficLevel string, costPerSecond float64) (*models.AdvertisementCalculationRate, error) {
	normalized := normalizeTrafficLevel(trafficLevel)
	if normalized == "" {
		return nil, fmt.Errorf("trafficLevel must be one of Peak, Moderate, Off-Peak")
	}
	if costPerSecond < 0 {
		return nil, fmt.Errorf("costPerSecond must be greater than or equal to 0")
	}
	return s.repo.UpsertCalculationRate(normalized, costPerSecond)
}

func (s *AdvertisementService) RecordPlaybackLog(req *models.AdvertisementPlaybackLogRequest) (*models.AdvertisementPlaybackLog, error) {
	adID := strings.TrimSpace(req.AdvertisementID)
	adName := strings.TrimSpace(req.AdvertisementName)
	if adID == "" {
		return nil, fmt.Errorf("advertisementId is required")
	}
	if adName == "" {
		return nil, fmt.Errorf("advertisementName is required")
	}
	if req.DurationSeconds <= 0 {
		return nil, fmt.Errorf("durationSeconds must be greater than zero")
	}

	playedAt := time.Now().UTC()
	if strings.TrimSpace(req.PlayedAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.PlayedAt))
		if err != nil {
			return nil, fmt.Errorf("playedAt must be in RFC3339 format")
		}
		playedAt = parsed.UTC()
	}

	trafficLevel := normalizeTrafficLevel(req.TrafficLevel)
	if trafficLevel == "" {
		trafficLevel = trafficLevelFromScheduledTime(playedAt)
	}

	item := &models.AdvertisementPlaybackLog{
		AdvertisementID:   adID,
		AdvertisementName: adName,
		TrafficLevel:      trafficLevel,
		DurationSeconds:   req.DurationSeconds,
		PlayedAt:          playedAt,
	}

	if err := s.repo.CreatePlaybackLog(item); err != nil {
		return nil, fmt.Errorf("create playback log: %w", err)
	}

	return item, nil
}

func (s *AdvertisementService) GetPlaybackLogs(startDate, endDate, advertisementID, trafficLevel string, limit int) ([]models.AdvertisementPlaybackLog, error) {
	var startAt *time.Time
	var endAt *time.Time

	if strings.TrimSpace(startDate) != "" {
		parsed, err := parseDateOrRFC3339(startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid startDate: %w", err)
		}
		startAt = &parsed
	}

	if strings.TrimSpace(endDate) != "" {
		parsed, err := parseDateOrRFC3339(endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid endDate: %w", err)
		}
		if len(strings.TrimSpace(endDate)) == len("2006-01-02") {
			parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endAt = &parsed
	}

	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return nil, fmt.Errorf("startDate cannot be after endDate")
	}

	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	normalizedTrafficLevel := ""
	if strings.TrimSpace(trafficLevel) != "" {
		normalizedTrafficLevel = normalizeTrafficLevel(trafficLevel)
		if normalizedTrafficLevel == "" {
			return nil, fmt.Errorf("trafficLevel must be one of Peak, Moderate, Off-Peak")
		}
	}

	return s.repo.GetPlaybackLogs(startAt, endAt, strings.TrimSpace(advertisementID), normalizedTrafficLevel, limit)
}

func (s *AdvertisementService) GetCostReport(startDate, endDate string) ([]models.AdvertisementCostReportRow, error) {
	var startAt *time.Time
	var endAt *time.Time

	if strings.TrimSpace(startDate) != "" {
		parsed, err := parseDateOrRFC3339(startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid startDate: %w", err)
		}
		startAt = &parsed
	}

	if strings.TrimSpace(endDate) != "" {
		parsed, err := parseDateOrRFC3339(endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid endDate: %w", err)
		}
		if len(strings.TrimSpace(endDate)) == len("2006-01-02") {
			parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endAt = &parsed
	}

	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return nil, fmt.Errorf("startDate cannot be after endDate")
	}

	return s.repo.GetCostReport(startAt, endAt)
}

func (s *AdvertisementService) CreateAdvertisement(req *models.AdvertisementCreateRequest) (*models.Advertisement, error) {
	// Parse OccursOnceAt if provided
	var occursOnceAt *time.Time
	if req.OccursOnceAt != nil && *req.OccursOnceAt != "" {
		fmt.Printf("=== Service Layer: Parsing OccursOnceAt ===\n")
		fmt.Printf("Raw OccursOnceAt string: %s\n", *req.OccursOnceAt)
		t, err := time.Parse(time.RFC3339, *req.OccursOnceAt)
		if err != nil {
			fmt.Printf("ERROR parsing OccursOnceAt: %v\n", err)
			return nil, fmt.Errorf("invalid occursOnceAt format: %w", err)
		}
		occursOnceAt = &t
		fmt.Printf("Successfully parsed OccursOnceAt: %v\n", t)
	} else {
		fmt.Printf("=== Service Layer: OccursOnceAt is nil or empty ===\n")
		if req.OccursOnceAt == nil {
			fmt.Printf("OccursOnceAt pointer is nil\n")
		} else {
			fmt.Printf("OccursOnceAt string is empty\n")
		}
	}

	ad := &models.Advertisement{
		AdvertisementName:     req.AdvertisementName,
		Description:           req.Description,
		AdvertisementCategory: req.AdvertisementCategory,
		MediaDuration:         req.MediaDuration,
		MediaURL:              req.MediaURL,
		MediaType:             req.MediaType,
		LoungeGroupName:       req.LoungeGroupName,
		Priority:              req.Priority,
		ScheduleType:          &req.ScheduleType,
		Frequency:             req.Frequency,
		RecurrenceInterval:    req.RecurrenceInterval,
		OccursOnceAt:          occursOnceAt,
		OccursEveryInterval:   req.OccursEveryInterval,
		WeeklyDays:            req.WeeklyDays,
		MonthlyDayOfMonth:     req.MonthlyDayOfMonth,
		MonthlyWeek:           req.MonthlyWeek,
		MonthlyDay:            req.MonthlyDay,
		StartDate:             req.StartDate,
		EndDate:               req.EndDate,
		StartTime:             req.StartTime,
		EndTime:               req.EndTime,
		MaxIdleLoopDuration:   req.MaxIdleLoopDuration,
		PlayTimeSlots:         normalizePlayTimeSlots(req.PlayTimeSlots, req.PlayTimeSlot),
		Status:                valueOrDefault(req.Status, "active"),
	}

	err := s.repo.Create(ad)
	if err != nil {
		return nil, fmt.Errorf("failed to create advertisement: %w", err)
	}

	return ad, nil
}

func (s *AdvertisementService) GetAllAdvertisements() ([]models.Advertisement, error) {
	return s.repo.GetAll()
}

func (s *AdvertisementService) SyncScheduledAdsCosts(startDate, endDate string) (int, error) {
	var startAt *time.Time
	var endAt *time.Time

	if strings.TrimSpace(startDate) != "" {
		parsed, err := parseDateOrRFC3339(startDate)
		if err != nil {
			return 0, fmt.Errorf("invalid startDate: %w", err)
		}
		startAt = &parsed
	}

	if strings.TrimSpace(endDate) != "" {
		parsed, err := parseDateOrRFC3339(endDate)
		if err != nil {
			return 0, fmt.Errorf("invalid endDate: %w", err)
		}
		if len(strings.TrimSpace(endDate)) == len("2006-01-02") {
			parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
		endAt = &parsed
	}

	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return 0, fmt.Errorf("startDate cannot be after endDate")
	}

	ads, err := s.repo.GetScheduledAdsForDateRange(startAt, endAt)
	if err != nil {
		return 0, fmt.Errorf("fetch scheduled ads: %w", err)
	}

	count := 0
	for _, ad := range ads {
		if ad.MediaDuration == nil || *ad.MediaDuration <= 0 {
			continue
		}

		playedAt := time.Now().UTC()
		if ad.OccursOnceAt != nil {
			playedAt = ad.OccursOnceAt.UTC()
		}

		trafficLevel := trafficLevelFromScheduledTime(playedAt)

		log := &models.AdvertisementPlaybackLog{
			AdvertisementID:   ad.ID,
			AdvertisementName: ad.AdvertisementName,
			TrafficLevel:      trafficLevel,
			DurationSeconds:   *ad.MediaDuration,
			PlayedAt:          playedAt,
		}

		if err := s.repo.CreatePlaybackLog(log); err != nil {
			fmt.Printf("warning: failed to create playback log for ad %s: %v\n", ad.ID, err)
			continue
		}
		count++
	}

	return count, nil
}

func (s *AdvertisementService) GetTVAdsManifest(loungeGroup, baseURL string) ([]models.TVAdManifestItem, error) {
	ads, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}

	groups, err := s.repo.GetAllGroups()
	if err != nil {
		return nil, err
	}

	targetGroup := strings.TrimSpace(strings.ToLower(loungeGroup))
	allowedGroups := map[string]struct{}{}
	if targetGroup != "" {
		allowedGroups[targetGroup] = struct{}{}
	}

	for _, group := range groups {
		for _, lounge := range parseGroupLounges(group.Lounges) {
			if strings.TrimSpace(strings.ToLower(lounge)) == targetGroup {
				allowedGroups[strings.TrimSpace(strings.ToLower(group.GroupName))] = struct{}{}
				break
			}
		}
	}

	items := make([]models.TVAdManifestItem, 0)

	for _, ad := range ads {
		if strings.ToLower(strings.TrimSpace(ad.Status)) != "active" {
			continue
		}
		if ad.LoungeGroupName == nil || strings.TrimSpace(*ad.LoungeGroupName) == "" {
			continue
		}
		if !containsAnyAllowedGroup(*ad.LoungeGroupName, allowedGroups) {
			continue
		}
		if ad.MediaURL == nil || strings.TrimSpace(*ad.MediaURL) == "" {
			continue
		}

		resolvedURL := resolveMediaURL(baseURL, strings.TrimSpace(*ad.MediaURL))
		mediaHash := sha256Hex(resolvedURL + "|" + ad.UpdatedAt.UTC().Format(time.RFC3339Nano))

		items = append(items, models.TVAdManifestItem{
			ID:                ad.ID,
			AdvertisementName: ad.AdvertisementName,
			MediaURL:          resolvedURL,
			MediaType:         valueOrPtr(ad.MediaType),
			MediaDuration:     ad.MediaDuration,
			Priority:          ad.Priority,
			ScheduleType:      valueOrPtr(ad.ScheduleType),
			Frequency:         valueOrPtr(ad.Frequency),
			StartDate:         valueOrPtr(ad.StartDate),
			EndDate:           valueOrPtr(ad.EndDate),
			StartTime:         valueOrPtr(ad.StartTime),
			EndTime:           valueOrPtr(ad.EndTime),
			PlayTimeSlot:      s.formatExistingAdTimeSlot(&ad),
			PlayTimeSlots:     s.buildPlayTimeSlotsForManifest(&ad),
			NextPlayAt:        s.computeNextPlayAtUTC(&ad, time.Now().UTC()),
			MediaHash:         mediaHash,
			UpdatedAt:         ad.UpdatedAt,
		})
	}

	return items, nil
}

func containsAnyAllowedGroup(rawGroups string, allowedGroups map[string]struct{}) bool {
	if len(allowedGroups) == 0 {
		return false
	}

	for _, group := range strings.Split(rawGroups, ",") {
		normalized := strings.TrimSpace(strings.ToLower(group))
		if _, exists := allowedGroups[normalized]; exists {
			return true
		}
	}

	return false
}

func parseGroupLounges(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	var lounges []string
	if err := json.Unmarshal([]byte(trimmed), &lounges); err == nil {
		return lounges
	}

	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			result = append(result, value)
		}
	}

	return result
}

func (s *AdvertisementService) computeNextPlayAtUTC(ad *models.Advertisement, now time.Time) string {
	if ad == nil {
		return ""
	}

	// Keep recurring schedule evaluation aligned with local wall-clock time,
	// then return the result as UTC for storage/transport consistency.
	nowUTC := now.UTC()
	nowLocal := now.In(time.Local)

	scheduleType := strings.ToLower(strings.TrimSpace(valueOrPtr(ad.ScheduleType)))
	frequency := strings.ToLower(strings.TrimSpace(valueOrPtr(ad.Frequency)))
	interval := 1
	if ad.RecurrenceInterval != nil && *ad.RecurrenceInterval > 0 {
		interval = *ad.RecurrenceInterval
	}

	// Parse schedule dates in local time so recurring ads fire on the expected
	// local calendar day/time instead of shifting at UTC boundaries.
	scheduleStartDate := parseScheduleDate(valueOrPtr(ad.StartDate), time.Local)
	scheduleEndDate := parseScheduleDate(valueOrPtr(ad.EndDate), time.Local)
	// Resolve time of day from StartTime first, then fall back to OccursOnceAt
	todHour, todMinute := resolveAdTimeOfDay(ad)

	log.Printf("[DEBUG] computeNextPlayAtUTC: scheduleType=%s, frequency=%s, todHour=%d, todMinute=%d, startDate=%v, endDate=%v", scheduleType, frequency, todHour, todMinute, scheduleStartDate, scheduleEndDate)

	switch scheduleType {
	case "on startup", "on idle":
		return nowUTC.Format(time.RFC3339)
	case "one-time":
		if ad.OccursOnceAt == nil {
			return ""
		}
		// Allow ad to play if it's within grace window in the future or currently happening
		occurTime := ad.OccursOnceAt.UTC()
		if occurTime.After(nowUTC) || nowUTC.Sub(occurTime) <= nextPlayAtGraceWindow {
			return occurTime.Format(time.RFC3339)
		}
		return ""
	case "recurring":
		loc := time.Local

		var next time.Time
		switch frequency {
		case "weekly":
			next = nextWeeklyOccurrence(nowLocal, todHour, todMinute, interval, scheduleStartDate, scheduleEndDate, valueOrPtr(ad.WeeklyDays), loc)
		case "monthly":
			next = nextMonthlyOccurrence(nowLocal, todHour, todMinute, interval, scheduleStartDate, scheduleEndDate, ad.MonthlyDayOfMonth, valueOrPtr(ad.MonthlyWeek), valueOrPtr(ad.MonthlyDay), loc)
		default:
			next = nextDailyOccurrence(nowLocal, todHour, todMinute, interval, scheduleStartDate, scheduleEndDate, loc)
		}

		if next.IsZero() {
			log.Printf("[DEBUG] recurring next occurrence is zero")
			return ""
		}
		log.Printf("[DEBUG] next occurrence: %v", next)
		return next.UTC().Format(time.RFC3339)
	default:
		return ""
	}
}

func resolveAdTimeOfDay(ad *models.Advertisement) (int, int) {
	if ad == nil {
		return 0, 0
	}

	// Priority 1: Use StartTime if available (set for recurring schedules with traffic levels)
	if ad.StartTime != nil && strings.TrimSpace(*ad.StartTime) != "" {
		if hour, minute, ok := parseHHMM(*ad.StartTime); ok {
			return hour, minute
		}
	}

	// Priority 2: Use OccursOnceAt if available (set for one-time schedules)
	if ad.OccursOnceAt != nil {
		utcTime := ad.OccursOnceAt.UTC()
		return utcTime.Hour(), utcTime.Minute()
	}

	// Default: use 0:00 (meaning the ad should play at any time)
	return 0, 0
}

func parseHHMM(value string) (int, int, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, 0, false
	}

	parts := strings.Split(trimmed, ":")
	if len(parts) < 2 {
		return 0, 0, false
	}

	hour, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	minute, errM := strconv.Atoi(strings.TrimSpace(parts[1]))

	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		log.Printf("[WARN] Invalid time format: %s (hour=%d, err=%v; minute=%d, err=%v)", trimmed, hour, errH, minute, errM)
		return 0, 0, false
	}

	return hour, minute, true
}

// parseScheduleDateUTC parses a date string (YYYY-MM-DD or RFC3339) and returns a UTC time at 00:00:00
func parseScheduleDateUTC(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}

	// Extract date portion (before T if present)
	dateOnly := strings.Split(trimmed, "T")[0]

	// Try parsing as YYYY-MM-DD
	if date, err := time.Parse("2006-01-02", dateOnly); err == nil {
		return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	}

	// Try parsing as full RFC3339
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		utcTime := parsed.UTC()
		return time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, time.UTC)
	}

	log.Printf("[WARN] Unable to parse schedule date: %s", trimmed)

	return time.Time{}
}

func parseScheduleDate(value string, loc *time.Location) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}

	dateOnly := strings.Split(trimmed, "T")[0]
	if date, err := time.ParseInLocation("2006-01-02", dateOnly, loc); err == nil {
		return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	}

	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		inLoc := parsed.In(loc)
		return time.Date(inLoc.Year(), inLoc.Month(), inLoc.Day(), 0, 0, 0, 0, loc)
	}

	return time.Time{}
}

func nextDailyOccurrence(now time.Time, hour, minute, interval int, startDate, endDate time.Time, loc *time.Location) time.Time {
	if interval <= 0 {
		interval = 1
	}

	baseDate := startOfDayInLocation(now, loc)

	// If schedule hasn't started yet, start from startDate
	if !startDate.IsZero() && startDate.After(baseDate) {
		baseDate = startDate
	}

	// Calculate the number of days from startDate to baseDate to ensure interval alignment
	var intervalOffset int
	if !startDate.IsZero() {
		intervalOffset = daysBetween(startDate, baseDate) % interval
		if intervalOffset != 0 {
			baseDate = baseDate.AddDate(0, 0, interval-intervalOffset)
		}
	}

	// Search for next occurrence within a reasonable window (10 years)
	for i := 0; i < 3650; i++ {
		if !isWithinDateWindow(baseDate, startDate, endDate) {
			if !endDate.IsZero() && baseDate.After(endDate) {
				return time.Time{} // Past the end date
			}
			baseDate = baseDate.AddDate(0, 0, interval)
			continue
		}

		candidate := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, minute, 0, 0, loc)
		if isAfterNow(candidate, now) {
			return candidate
		}

		baseDate = baseDate.AddDate(0, 0, interval)
	}

	return time.Time{}
}

func nextWeeklyOccurrence(now time.Time, hour, minute, interval int, startDate, endDate time.Time, weeklyDaysRaw string, loc *time.Location) time.Time {
	if interval <= 0 {
		interval = 1
	}

	allowed := parseWeeklyDays(weeklyDaysRaw)
	if len(allowed) == 0 {
		return time.Time{}
	}

	currentDate := startOfDayInLocation(now, loc)
	if !startDate.IsZero() && startDate.After(currentDate) {
		currentDate = startDate
	}

	for i := 0; i < 3700; i++ {
		if !isWithinDateWindow(currentDate, startDate, endDate) {
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}

		if _, ok := allowed[currentDate.Weekday()]; ok {
			if !startDate.IsZero() {
				weeks := daysBetween(startDate, currentDate) / 7
				if weeks < 0 || weeks%interval != 0 {
					currentDate = currentDate.AddDate(0, 0, 1)
					continue
				}
			}

			candidate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), hour, minute, 0, 0, loc)
			if isAfterNow(candidate, now) {
				return candidate
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return time.Time{}
}

func nextMonthlyOccurrence(now time.Time, hour, minute, interval int, startDate, endDate time.Time, monthlyDayOfMonth *int, monthlyWeekRaw, monthlyDayRaw string, loc *time.Location) time.Time {
	if interval <= 0 {
		interval = 1
	}

	base := startOfDayInLocation(now, loc)
	if !startDate.IsZero() && startDate.After(base) {
		base = startDate
	}

	startMonth := monthIndex(base)
	anchorMonth := startMonth
	if !startDate.IsZero() {
		anchorMonth = monthIndex(startDate)
	}

	for i := 0; i < 240; i++ {
		currentMonth := startMonth + i
		if (currentMonth-anchorMonth)%interval != 0 {
			continue
		}

		year, month := yearMonthFromIndex(currentMonth)
		candidateDate := time.Time{}

		if monthlyDayOfMonth != nil && *monthlyDayOfMonth > 0 {
			day := *monthlyDayOfMonth
			last := lastDayOfMonth(year, month, loc)
			if day > last {
				day = last
			}
			candidateDate = time.Date(year, month, day, 0, 0, 0, 0, loc)
		} else {
			candidateDate = nthWeekdayOfMonth(year, month, monthlyWeekRaw, monthlyDayRaw, loc)
		}

		if candidateDate.IsZero() || !isWithinDateWindow(candidateDate, startDate, endDate) {
			continue
		}

		candidate := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), hour, minute, 0, 0, loc)
		if isAfterNow(candidate, now) {
			return candidate
		}
	}

	return time.Time{}
}

func parseWeeklyDays(raw string) map[time.Weekday]struct{} {
	result := map[time.Weekday]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "sunday":
			result[time.Sunday] = struct{}{}
		case "monday":
			result[time.Monday] = struct{}{}
		case "tuesday":
			result[time.Tuesday] = struct{}{}
		case "wednesday":
			result[time.Wednesday] = struct{}{}
		case "thursday":
			result[time.Thursday] = struct{}{}
		case "friday":
			result[time.Friday] = struct{}{}
		case "saturday":
			result[time.Saturday] = struct{}{}
		}
	}
	return result
}

func nthWeekdayOfMonth(year int, month time.Month, weekRaw, dayRaw string, loc *time.Location) time.Time {
	weekday, ok := weekdayFromString(dayRaw)
	if !ok {
		return time.Time{}
	}

	week := strings.ToLower(strings.TrimSpace(weekRaw))
	if week == "last" {
		last := time.Date(year, month, lastDayOfMonth(year, month, loc), 0, 0, 0, 0, loc)
		for last.Weekday() != weekday {
			last = last.AddDate(0, 0, -1)
		}
		return last
	}

	ordinalMap := map[string]int{
		"first":  1,
		"second": 2,
		"third":  3,
		"fourth": 4,
	}
	ord, exists := ordinalMap[week]
	if !exists {
		return time.Time{}
	}

	date := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	for date.Weekday() != weekday {
		date = date.AddDate(0, 0, 1)
	}

	date = date.AddDate(0, 0, 7*(ord-1))
	if date.Month() != month {
		return time.Time{}
	}

	return date
}

func weekdayFromString(value string) (time.Weekday, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sunday":
		return time.Sunday, true
	case "monday":
		return time.Monday, true
	case "tuesday":
		return time.Tuesday, true
	case "wednesday":
		return time.Wednesday, true
	case "thursday":
		return time.Thursday, true
	case "friday":
		return time.Friday, true
	case "saturday":
		return time.Saturday, true
	default:
		return time.Sunday, false
	}
}

func isAfterNow(candidate, now time.Time) bool {
	// Keep the same play slot valid for a short grace period so clients polling
	// slightly after the exact second still receive the due ad instead of next cycle.
	return candidate.After(now) || candidate.Equal(now) || candidate.Add(nextPlayAtGraceWindow).After(now)
}

func isWithinDateWindow(date, startDate, endDate time.Time) bool {
	if !startDate.IsZero() && date.Before(startDate) {
		return false
	}
	if !endDate.IsZero() && date.After(endDate) {
		return false
	}
	return true
}

func startOfDayInLocation(t time.Time, loc *time.Location) time.Time {
	inLoc := t.In(loc)
	return time.Date(inLoc.Year(), inLoc.Month(), inLoc.Day(), 0, 0, 0, 0, loc)
}

func daysBetween(start, end time.Time) int {
	s := startOfDayInLocation(start, start.Location())
	e := startOfDayInLocation(end, start.Location())
	return int(e.Sub(s).Hours() / 24)
}

func monthIndex(t time.Time) int {
	return t.Year()*12 + int(t.Month()) - 1
}

func yearMonthFromIndex(index int) (int, time.Month) {
	year := index / 12
	month := index%12 + 1
	return year, time.Month(month)
}

func lastDayOfMonth(year int, month time.Month, loc *time.Location) int {
	firstNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, loc)
	return firstNextMonth.AddDate(0, 0, -1).Day()
}

func resolveMediaURL(baseURL, mediaURL string) string {
	u, err := url.Parse(mediaURL)
	if err != nil {
		return mediaURL
	}
	if u.IsAbs() {
		return mediaURL
	}
	if strings.TrimSpace(baseURL) == "" {
		return mediaURL
	}
	b, err := url.Parse(strings.TrimRight(baseURL, "/") + "/")
	if err != nil {
		return mediaURL
	}
	return b.ResolveReference(u).String()
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (s *AdvertisementService) GetAdvertisementByID(id string) (*models.Advertisement, error) {
	return s.repo.GetByID(id)
}

func (s *AdvertisementService) UpdateAdvertisement(id string, req *models.AdvertisementCreateRequest) error {
	ad, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("advertisement not found: %w", err)
	}

	// Parse OccursOnceAt if provided
	var occursOnceAt *time.Time
	if req.OccursOnceAt != nil && *req.OccursOnceAt != "" {
		t, err := time.Parse(time.RFC3339, *req.OccursOnceAt)
		if err != nil {
			return fmt.Errorf("invalid occursOnceAt format: %w", err)
		}
		occursOnceAt = &t
	}

	ad.AdvertisementName = req.AdvertisementName
	ad.Description = req.Description
	ad.AdvertisementCategory = req.AdvertisementCategory
	ad.MediaDuration = req.MediaDuration
	ad.MediaURL = req.MediaURL
	ad.MediaType = req.MediaType
	ad.LoungeGroupName = req.LoungeGroupName
	ad.Priority = req.Priority
	ad.ScheduleType = &req.ScheduleType
	ad.Frequency = req.Frequency
	ad.RecurrenceInterval = req.RecurrenceInterval
	ad.OccursOnceAt = occursOnceAt
	ad.OccursEveryInterval = req.OccursEveryInterval
	ad.WeeklyDays = req.WeeklyDays
	ad.MonthlyDayOfMonth = req.MonthlyDayOfMonth
	ad.MonthlyWeek = req.MonthlyWeek
	ad.MonthlyDay = req.MonthlyDay
	ad.StartDate = req.StartDate
	ad.EndDate = req.EndDate
	ad.StartTime = req.StartTime
	ad.EndTime = req.EndTime
	ad.MaxIdleLoopDuration = req.MaxIdleLoopDuration
	ad.PlayTimeSlots = normalizePlayTimeSlots(req.PlayTimeSlots, req.PlayTimeSlot)
	ad.Status = valueOrDefault(req.Status, ad.Status)

	return s.repo.Update(ad)
}

func (s *AdvertisementService) DeleteAdvertisement(id string) error {
	return s.repo.Delete(id)
}

// Conflict checking
func (s *AdvertisementService) CheckConflicts(req *models.ConflictCheckRequest) (*models.ConflictResponse, error) {
	// Fetch advertisements limited by schedule type for efficiency
	ads, err := s.repo.CheckConflicts(req.ScheduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to load advertisements: %w", err)
	}

	groups, err := s.repo.GetAllGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to load groups: %w", err)
	}

	groupLounges := buildGroupLoungeMap(groups)
	requestedGroup := strings.TrimSpace(req.LoungeGroupName)
	requestedLounges := groupLounges[requestedGroup]

	// Fallback: if the group exists but lounges cannot be resolved, treat the group name as a pseudo-lounge
	if len(requestedLounges) == 0 && requestedGroup != "" {
		requestedLounges = []string{requestedGroup}
	}

	fmt.Printf("DEBUG: CheckConflicts for group '%s' - requestedLounges: %v\n", requestedGroup, requestedLounges)

	response := &models.ConflictResponse{
		HasConflict:     false,
		ConflictingAds:  []string{},
		AffectedLounges: []string{},
	}

	if len(requestedLounges) == 0 {
		// If no lounges resolved, no conflict to check
		return response, nil
	}

	affectedSet := map[string]struct{}{}

	for _, ad := range ads {
		if req.ExcludeAdvertisementID != nil && ad.ID == *req.ExcludeAdvertisementID {
			continue
		}

		if ad.ScheduleType == nil || *ad.ScheduleType == "" {
			continue
		}

		adGroup := strings.TrimSpace(valueOrPtr(ad.LoungeGroupName))
		adLounges := groupLounges[adGroup]
		if len(adLounges) == 0 && adGroup != "" {
			adLounges = []string{adGroup}
		}
		sharedLounges := intersectStrings(requestedLounges, adLounges)

		// Fallback: if no lounges resolved but group names match exactly, treat as overlapping lounge set
		if len(sharedLounges) == 0 && len(requestedLounges) == 0 && requestedGroup != "" && adGroup == requestedGroup {
			sharedLounges = []string{requestedGroup}
		}

		fmt.Printf("DEBUG: Checking ad '%s' - adGroup: '%s', adLounges: %v, requestedLounges: %v, sharedLounges: %v\n",
			ad.AdvertisementName, adGroup, adLounges, requestedLounges, sharedLounges)

		if len(sharedLounges) == 0 {
			continue
		}

		if !s.hasScheduleConflict(req, &ad) {
			continue
		}

		response.HasConflict = true
		response.ConflictingAds = append(response.ConflictingAds, ad.AdvertisementName)
		for _, l := range sharedLounges {
			affectedSet[l] = struct{}{}
		}
	}

	if response.HasConflict {
		for l := range affectedSet {
			response.AffectedLounges = append(response.AffectedLounges, l)
		}
		fmt.Printf("DEBUG: Conflict response - AffectedLounges: %v, ConflictingAds: %v\n", response.AffectedLounges, response.ConflictingAds)
		response.ConflictTimeSlot = s.formatTimeSlot(req)
		response.ConflictMessage = "Schedule conflict detected in overlapping lounges"
		response.SuggestedAction = "Remove the overlapping lounge or adjust the time slot"
	}

	return response, nil
}

func buildGroupLoungeMap(groups []models.AdvertisementGroup) map[string][]string {
	result := make(map[string][]string)
	for _, g := range groups {
		var lounges []string
		if err := json.Unmarshal([]byte(g.Lounges), &lounges); err != nil || len(lounges) == 0 {
			// Fallback: handle non-JSON stored lounges like "A, B, C"
			parts := strings.Split(g.Lounges, ",")
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					lounges = append(lounges, trimmed)
				}
			}
		}
		result[g.GroupName] = lounges
		fmt.Printf("DEBUG: Group '%s' has lounges: %v (raw: %s)\n", g.GroupName, lounges, g.Lounges)
	}
	return result
}

func intersectStrings(a, b []string) []string {
	set := make(map[string]struct{})
	for _, v := range a {
		set[strings.ToLower(strings.TrimSpace(v))] = struct{}{}
	}
	var res []string
	for _, v := range b {
		if _, ok := set[strings.ToLower(strings.TrimSpace(v))]; ok {
			res = append(res, v)
		}
	}
	return res
}

func (s *AdvertisementService) hasScheduleConflict(req *models.ConflictCheckRequest, existing *models.Advertisement) bool {
	// Same schedule type only
	if existing.ScheduleType == nil || *existing.ScheduleType != req.ScheduleType {
		return false
	}

	switch req.ScheduleType {
	case "on startup", "on idle":
		return true
	case "one-time":
		if req.OccursOnceAt == nil || existing.OccursOnceAt == nil {
			return false
		}
		return *req.OccursOnceAt == existing.OccursOnceAt.Format(time.RFC3339)
	case "recurring":
		return s.recurringConflict(req, existing)
	default:
		return false
	}
}

func (s *AdvertisementService) recurringConflict(req *models.ConflictCheckRequest, existing *models.Advertisement) bool {
	reqFreq := strings.ToLower(valueOrEmpty(req.Frequency))
	existFreq := strings.ToLower(valueOrPtr(existing.Frequency))

	if reqFreq != existFreq {
		return false
	}

	// weekly: need overlapping day and time
	if reqFreq == "weekly" {
		if req.WeeklyDays == nil || existing.WeeklyDays == nil {
			return false
		}
		if !weeklyOverlap(*req.WeeklyDays, *existing.WeeklyDays) {
			return false
		}
		return timeRangeOverlap(req.StartTime, req.EndTime, existing.StartTime, existing.EndTime, req.OccursOnceAt, existing.OccursOnceAt)
	}

	// monthly: match day-of-month and time overlap
	if reqFreq == "monthly" {
		if req.MonthlyDayOfMonth == nil || existing.MonthlyDayOfMonth == nil {
			return false
		}
		if *req.MonthlyDayOfMonth != *existing.MonthlyDayOfMonth {
			return false
		}
		return timeRangeOverlap(req.StartTime, req.EndTime, existing.StartTime, existing.EndTime, req.OccursOnceAt, existing.OccursOnceAt)
	}

	// daily or unspecified recurrence: just check time overlap
	return timeRangeOverlap(req.StartTime, req.EndTime, existing.StartTime, existing.EndTime, req.OccursOnceAt, existing.OccursOnceAt)
}

func weeklyOverlap(reqDays, existDays string) bool {
	reqList := strings.Split(strings.ToLower(reqDays), ",")
	existList := strings.Split(strings.ToLower(existDays), ",")
	set := make(map[string]struct{})
	for _, d := range reqList {
		set[strings.TrimSpace(d)] = struct{}{}
	}
	for _, d := range existList {
		if _, ok := set[strings.TrimSpace(d)]; ok {
			return true
		}
	}
	return false
}

func timeRangeOverlap(reqStart, reqEnd, existStart, existEnd *string, reqOnce *string, existOnce *time.Time) bool {
	// If both have start/end times, check ranges: reqStart < existEnd && reqEnd > existStart
	if reqStart != nil && reqEnd != nil && existStart != nil && existEnd != nil {
		return *reqStart < *existEnd && *reqEnd > *existStart
	}

	// Fallback to occursOnceAt hour/minute
	if reqOnce != nil && existOnce != nil {
		reqTime, err := time.Parse(time.RFC3339, *reqOnce)
		if err != nil {
			return false
		}
		return reqTime.Hour() == existOnce.Hour() && reqTime.Minute() == existOnce.Minute()
	}

	return false
}

func (s *AdvertisementService) formatTimeSlot(req *models.ConflictCheckRequest) string {
	if req.ScheduleType == "on startup" {
		return "On Startup"
	}
	if req.ScheduleType == "on idle" {
		return "On Idle"
	}
	if req.OccursOnceAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.OccursOnceAt); err == nil {
			// Convert UTC time back to local timezone for display
			localTime := t.Local()
			hour := localTime.Hour()
			minute := localTime.Minute()
			period := "AM"
			displayHour := hour

			if hour >= 12 {
				period = "PM"
				if hour > 12 {
					displayHour = hour - 12
				}
			}
			if displayHour == 0 {
				displayHour = 12
			}

			return fmt.Sprintf("%02d:%02d %s", displayHour, minute, period)
		}
	}
	if req.StartTime != nil && req.EndTime != nil {
		return fmt.Sprintf("%s to %s", *req.StartTime, *req.EndTime)
	}
	if req.StartTime != nil {
		return *req.StartTime
	}
	return "Unspecified time"
}

// buildPlayTimeSlotsForManifest returns the ad's PlayTimeSlots array for TV manifest.
// If PlayTimeSlots is empty, falls back to deriving a single slot from the schedule.
func (s *AdvertisementService) buildPlayTimeSlotsForManifest(ad *models.Advertisement) []string {
	if len(ad.PlayTimeSlots) > 0 {
		return ad.PlayTimeSlots
	}
	// Derive a single legacy slot label for backwards compat
	legacy := s.formatExistingAdTimeSlot(ad)
	if legacy != "" && legacy != "Unspecified time" {
		return []string{legacy}
	}
	return []string{}
}

// formatExistingAdTimeSlot formats the time slot of an existing advertisement
func (s *AdvertisementService) formatExistingAdTimeSlot(ad *models.Advertisement) string {
	if ad.ScheduleType != nil {
		if *ad.ScheduleType == "on startup" {
			return "On Startup"
		}
		if *ad.ScheduleType == "on idle" {
			return "On Idle"
		}
	}

	// For one-time schedules
	if ad.OccursOnceAt != nil {
		// Convert UTC time back to local timezone for display
		localTime := ad.OccursOnceAt.Local()
		hour := localTime.Hour()
		minute := localTime.Minute()
		period := "AM"
		displayHour := hour

		if hour >= 12 {
			period = "PM"
			if hour > 12 {
				displayHour = hour - 12
			}
		}
		if displayHour == 0 {
			displayHour = 12
		}

		return fmt.Sprintf("%02d:%02d %s", displayHour, minute, period)
	}

	// For recurring schedules with time range
	if ad.StartTime != nil && ad.EndTime != nil {
		return fmt.Sprintf("%s to %s", *ad.StartTime, *ad.EndTime)
	}

	if ad.StartTime != nil {
		return *ad.StartTime
	}

	return "Unspecified time"
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func normalizePlayTimeSlots(playTimeSlots []string, playTimeSlot *string) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})

	addToken := func(token string) {
		t := strings.TrimSpace(token)
		if t == "" {
			return
		}
		if _, ok := seen[t]; ok {
			return
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}

	for _, slot := range playTimeSlots {
		for _, part := range strings.Split(slot, ",") {
			addToken(part)
		}
	}

	if playTimeSlot != nil {
		for _, part := range strings.Split(*playTimeSlot, ",") {
			addToken(part)
		}
	}

	return result
}

func valueOrPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func valueOrDefault(v *string, defaultValue string) string {
	if v == nil || *v == "" {
		return defaultValue
	}
	return *v
}

func normalizeTrafficLevel(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "peak":
		return "Peak"
	case "moderate":
		return "Moderate"
	case "off-peak", "off peak", "off_peak":
		return "Off-Peak"
	default:
		return ""
	}
}

func parseDateOrRFC3339(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == len("2006-01-02") {
		parsed, err := time.Parse("2006-01-02", trimmed)
		if err != nil {
			return time.Time{}, err
		}
		return parsed.UTC(), nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func trafficLevelFromScheduledTime(t time.Time) string {
	hour := t.In(time.Local).Hour()

	if (hour >= 7 && hour < 11) || (hour >= 17 && hour < 21) {
		return "Peak"
	}

	if hour >= 11 && hour < 17 {
		return "Moderate"
	}

	return "Off-Peak"
}

// Advertisement Group services
func (s *AdvertisementService) CreateGroup(req *models.AdvertisementGroupCreateRequest) (*models.AdvertisementGroup, error) {
	loungesJSON, err := json.Marshal(req.Lounges)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal lounges: %w", err)
	}

	group := &models.AdvertisementGroup{
		GroupName:          req.GroupName,
		Lounges:            string(loungesJSON),
		NoOfAdvertisements: 0,
	}

	err = s.repo.CreateGroup(group)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return group, nil
}

func (s *AdvertisementService) GetAllGroups() ([]models.AdvertisementGroup, error) {
	return s.repo.GetAllGroups()
}

func (s *AdvertisementService) GetGroupByID(id string) (*models.AdvertisementGroup, error) {
	return s.repo.GetGroupByID(id)
}

func (s *AdvertisementService) UpdateGroup(id string, req *models.AdvertisementGroupCreateRequest) error {
	group, err := s.repo.GetGroupByID(id)
	if err != nil {
		return err
	}

	loungesJSON, err := json.Marshal(req.Lounges)
	if err != nil {
		return fmt.Errorf("failed to marshal lounges: %w", err)
	}

	group.GroupName = req.GroupName
	group.Lounges = string(loungesJSON)

	return s.repo.UpdateGroup(group)
}

func (s *AdvertisementService) DeleteGroup(id string) error {
	return s.repo.DeleteGroup(id)
}
