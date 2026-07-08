package database

import (
	"bus-schedule-lounge/internal/models"
	"encoding/json"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
)

type AdvertisementRepository struct {
	db *sql.DB
}

func NewAdvertisementRepository(db *sql.DB) *AdvertisementRepository {
	return &AdvertisementRepository{db: db}
}

func (r *AdvertisementRepository) columnExists(tableName, columnName string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
				AND table_name = $1
				AND column_name = $2
		)
	`
	var exists bool
	err := r.db.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists, err
}

func normalizeSlots(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}

const (
	playTimeSlotsMetaPrefix = "[[PLAY_TIME_SLOTS:"
	playTimeSlotsMetaSuffix = "]]"
)

func stripPlayTimeSlotsMetadata(raw string) string {
	start := strings.LastIndex(raw, playTimeSlotsMetaPrefix)
	if start < 0 {
		return strings.TrimSpace(raw)
	}
	endRelative := strings.Index(raw[start:], playTimeSlotsMetaSuffix)
	if endRelative < 0 {
		return strings.TrimSpace(raw)
	}
	end := start + endRelative + len(playTimeSlotsMetaSuffix)
	cleaned := strings.TrimSpace(raw[:start] + raw[end:])
	return cleaned
}

func encodePlayTimeSlotsInDescription(desc *string, slots []string) *string {
	base := ""
	if desc != nil {
		base = stripPlayTimeSlotsMetadata(*desc)
	}

	normalized := make([]string, 0, len(slots))
	seen := make(map[string]struct{})
	for _, slot := range slots {
		token := strings.TrimSpace(slot)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		normalized = append(normalized, token)
	}

	if len(normalized) == 0 {
		if strings.TrimSpace(base) == "" {
			return nil
		}
		clean := strings.TrimSpace(base)
		return &clean
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		if strings.TrimSpace(base) == "" {
			return nil
		}
		clean := strings.TrimSpace(base)
		return &clean
	}

	meta := playTimeSlotsMetaPrefix + string(encoded) + playTimeSlotsMetaSuffix
	if strings.TrimSpace(base) == "" {
		combined := meta
		return &combined
	}

	combined := strings.TrimSpace(base) + "\n" + meta
	return &combined
}

func decodePlayTimeSlotsFromDescription(desc *string) (*string, []string) {
	if desc == nil {
		return nil, nil
	}

	raw := strings.TrimSpace(*desc)
	if raw == "" {
		return nil, nil
	}

	start := strings.LastIndex(raw, playTimeSlotsMetaPrefix)
	if start < 0 {
		clean := strings.TrimSpace(raw)
		return &clean, nil
	}
	endRelative := strings.Index(raw[start:], playTimeSlotsMetaSuffix)
	if endRelative < 0 {
		clean := strings.TrimSpace(raw)
		return &clean, nil
	}
	end := start + endRelative + len(playTimeSlotsMetaSuffix)

	payload := raw[start+len(playTimeSlotsMetaPrefix) : start+endRelative]
	var parsed []string
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		clean := strings.TrimSpace(raw)
		return &clean, nil
	}

	cleanedRaw := strings.TrimSpace(raw[:start] + raw[end:])
	if cleanedRaw == "" {
		return nil, parsed
	}
	return &cleanedRaw, parsed
}

func (r *AdvertisementRepository) EnsureCalculationSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS advertisement_calculation (
			traffic_level TEXT PRIMARY KEY,
			cost_per_second NUMERIC(12,4) NOT NULL CHECK (cost_per_second >= 0),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS advertisement_playback_logs (
			id BIGSERIAL PRIMARY KEY,
			advertisement_id TEXT NOT NULL,
			advertisement_name TEXT NOT NULL,
			traffic_level TEXT NOT NULL,
			duration_seconds INTEGER NOT NULL CHECK (duration_seconds > 0),
			played_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_ad_calc_traffic_level
				FOREIGN KEY (traffic_level) REFERENCES advertisement_calculation(traffic_level)
		);

		CREATE TABLE IF NOT EXISTS advertisement_cost_aggregates (
			advertisement_id TEXT NOT NULL,
			advertisement_name TEXT NOT NULL,
			traffic_level TEXT NOT NULL,
			cost_date DATE NOT NULL,
			play_count INTEGER NOT NULL DEFAULT 0,
			total_seconds INTEGER NOT NULL DEFAULT 0,
			total_cost NUMERIC(14,2) NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (advertisement_id, traffic_level, cost_date),
			CONSTRAINT fk_ad_cost_agg_traffic_level
				FOREIGN KEY (traffic_level) REFERENCES advertisement_calculation(traffic_level)
		);

		CREATE INDEX IF NOT EXISTS idx_ad_playback_logs_played_at
			ON advertisement_playback_logs (played_at DESC);

		CREATE INDEX IF NOT EXISTS idx_ad_playback_logs_ad_id
			ON advertisement_playback_logs (advertisement_id);

		CREATE INDEX IF NOT EXISTS idx_ad_cost_aggregates_date
			ON advertisement_cost_aggregates (cost_date DESC);

		CREATE INDEX IF NOT EXISTS idx_ad_cost_aggregates_ad_id
			ON advertisement_cost_aggregates (advertisement_id);

		INSERT INTO advertisement_calculation (traffic_level, cost_per_second)
		VALUES
			('Peak', 2.00),
			('Moderate', 1.25),
			('Off-Peak', 0.75)
		ON CONFLICT (traffic_level) DO NOTHING;
	`

	if _, err := r.db.Exec(query); err != nil {
		return fmt.Errorf("ensure advertisement calculation schema: %w", err)
	}

	// Migrate: add play_time_slots column if it doesn't exist yet
	migrate := `
		ALTER TABLE advertisements
			ADD COLUMN IF NOT EXISTS play_time_slots TEXT[] NOT NULL DEFAULT '{}';
	`
	if _, err := r.db.Exec(migrate); err != nil {
		log.Printf("warning: could not add play_time_slots column automatically: %v", err)
	}

	return nil
}

func (r *AdvertisementRepository) GetCalculationRates() ([]models.AdvertisementCalculationRate, error) {
	rows, err := r.db.Query(`
		SELECT traffic_level, cost_per_second, updated_at
		FROM advertisement_calculation
		ORDER BY CASE traffic_level
			WHEN 'Peak' THEN 1
			WHEN 'Moderate' THEN 2
			WHEN 'Off-Peak' THEN 3
			ELSE 4
		END
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rates := make([]models.AdvertisementCalculationRate, 0)
	for rows.Next() {
		var row models.AdvertisementCalculationRate
		if err := rows.Scan(&row.TrafficLevel, &row.CostPerSecond, &row.UpdatedAt); err != nil {
			return nil, err
		}
		rates = append(rates, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rates, nil
}

func (r *AdvertisementRepository) UpsertCalculationRate(trafficLevel string, costPerSecond float64) (*models.AdvertisementCalculationRate, error) {
	row := &models.AdvertisementCalculationRate{}
	err := r.db.QueryRow(`
		INSERT INTO advertisement_calculation (traffic_level, cost_per_second, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (traffic_level)
		DO UPDATE SET cost_per_second = EXCLUDED.cost_per_second, updated_at = NOW()
		RETURNING traffic_level, cost_per_second, updated_at
	`, trafficLevel, costPerSecond).Scan(&row.TrafficLevel, &row.CostPerSecond, &row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *AdvertisementRepository) CreatePlaybackLog(item *models.AdvertisementPlaybackLog) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = tx.QueryRow(`
		INSERT INTO advertisement_playback_logs (
			advertisement_id,
			advertisement_name,
			traffic_level,
			duration_seconds,
			played_at
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`,
		item.AdvertisementID,
		item.AdvertisementName,
		item.TrafficLevel,
		item.DurationSeconds,
		item.PlayedAt,
	).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO advertisement_cost_aggregates (
			advertisement_id,
			advertisement_name,
			traffic_level,
			cost_date,
			play_count,
			total_seconds,
			total_cost,
			updated_at
		)
		SELECT
			$1,
			$2,
			$3,
			$4::timestamptz::date,
			1,
			$5,
			ROUND(($5 * c.cost_per_second)::NUMERIC, 2),
			NOW()
		FROM advertisement_calculation c
		WHERE c.traffic_level = $3
		ON CONFLICT (advertisement_id, traffic_level, cost_date)
		DO UPDATE SET
			advertisement_name = EXCLUDED.advertisement_name,
			play_count = advertisement_cost_aggregates.play_count + EXCLUDED.play_count,
			total_seconds = advertisement_cost_aggregates.total_seconds + EXCLUDED.total_seconds,
			total_cost = ROUND((advertisement_cost_aggregates.total_cost + EXCLUDED.total_cost)::NUMERIC, 2),
			updated_at = NOW()
	`,
		item.AdvertisementID,
		item.AdvertisementName,
		item.TrafficLevel,
		item.PlayedAt,
		item.DurationSeconds,
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *AdvertisementRepository) GetPlaybackLogs(startAt, endAt *time.Time, advertisementID, trafficLevel string, limit int) ([]models.AdvertisementPlaybackLog, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := r.db.Query(`
		SELECT
			id,
			advertisement_id,
			advertisement_name,
			traffic_level,
			duration_seconds,
			played_at,
			created_at
		FROM advertisement_playback_logs
		WHERE ($1::timestamptz IS NULL OR played_at >= $1::timestamptz)
			AND ($2::timestamptz IS NULL OR played_at <= $2::timestamptz)
			AND ($3 = '' OR advertisement_id = $3)
			AND ($4 = '' OR traffic_level = $4)
		ORDER BY played_at DESC, id DESC
		LIMIT $5
	`, startAt, endAt, strings.TrimSpace(advertisementID), strings.TrimSpace(trafficLevel), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]models.AdvertisementPlaybackLog, 0)
	for rows.Next() {
		var item models.AdvertisementPlaybackLog
		if err := rows.Scan(
			&item.ID,
			&item.AdvertisementID,
			&item.AdvertisementName,
			&item.TrafficLevel,
			&item.DurationSeconds,
			&item.PlayedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *AdvertisementRepository) GetCostReport(startAt, endAt *time.Time) ([]models.AdvertisementCostReportRow, error) {
	query := `
		SELECT
			a.advertisement_id,
			a.advertisement_name,
			a.traffic_level,
			COALESCE(SUM(a.play_count), 0)::INT AS play_count,
			COALESCE(SUM(a.total_seconds), 0)::INT AS total_seconds,
			CASE
				WHEN COALESCE(SUM(a.total_seconds), 0) = 0 THEN 0
				ELSE ROUND((SUM(a.total_cost) / SUM(a.total_seconds))::NUMERIC, 4)::FLOAT8
			END AS cost_per_second,
			ROUND(COALESCE(SUM(a.total_cost), 0)::NUMERIC, 2)::FLOAT8 AS total_cost
		FROM advertisement_cost_aggregates a
		WHERE ($1::date IS NULL OR a.cost_date >= $1::date)
			AND ($2::date IS NULL OR a.cost_date <= $2::date)
		GROUP BY a.advertisement_id, a.advertisement_name, a.traffic_level
		ORDER BY total_cost DESC, a.advertisement_name ASC
	`

	rows, err := r.db.Query(query, startAt, endAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.AdvertisementCostReportRow, 0)
	for rows.Next() {
		var row models.AdvertisementCostReportRow
		if err := rows.Scan(
			&row.AdvertisementID,
			&row.AdvertisementName,
			&row.TrafficLevel,
			&row.PlayCount,
			&row.TotalSeconds,
			&row.CostPerSecond,
			&row.TotalCost,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *AdvertisementRepository) Create(ad *models.Advertisement) error {
	// Debug logging
	fmt.Printf("=== Advertisement Repository Create ===\n")
	fmt.Printf("Recurrence Interval: %v\n", ad.RecurrenceInterval)
	fmt.Printf("Occurs Every Interval: %v\n", ad.OccursEveryInterval)
	fmt.Printf("Start Date: %v\n", ad.StartDate)
	fmt.Printf("End Date: %v\n", ad.EndDate)
	fmt.Printf("Start Time: %v\n", ad.StartTime)
	fmt.Printf("End Time: %v\n", ad.EndTime)
	fmt.Printf("Occurs Once At: %v\n", ad.OccursOnceAt)
	fmt.Printf("Max Idle Loop Duration: %v\n", ad.MaxIdleLoopDuration)
	fmt.Printf("Play Time Slots: %v\n", ad.PlayTimeSlots)
	fmt.Printf("=======================================\n")

	if ad.PlayTimeSlots == nil {
		ad.PlayTimeSlots = []string{}
	}

	hasSlotsArray, err := r.columnExists("advertisements", "play_time_slots")
	if err != nil {
		return err
	}
	hasLegacySlot, err := r.columnExists("advertisements", "play_time_slot")
	if err != nil {
		return err
	}

	legacySlot := strings.Join(ad.PlayTimeSlots, ",")
	descriptionForStorage := ad.Description
	if !hasSlotsArray && !hasLegacySlot {
		descriptionForStorage = encodePlayTimeSlotsInDescription(ad.Description, ad.PlayTimeSlots)
	}

	if hasSlotsArray {
		query := `
			INSERT INTO advertisements (
				advertisement_name, description, advertisement_category,
				media_duration, media_url, media_type, lounge_group_name,
				priority, schedule_type, frequency, recurrence_interval,
				occurs_once_at, occurs_every_interval, weekly_days,
				monthly_day_of_month, monthly_week, monthly_day,
				start_date, end_date, start_time, end_time, max_idle_loop_duration,
				play_time_slots, status
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
			) RETURNING id, version, created_at, updated_at`

		err = r.db.QueryRow(
			query,
			ad.AdvertisementName,
			descriptionForStorage,
			ad.AdvertisementCategory,
			ad.MediaDuration,
			ad.MediaURL,
			ad.MediaType,
			ad.LoungeGroupName,
			ad.Priority,
			ad.ScheduleType,
			ad.Frequency,
			ad.RecurrenceInterval,
			ad.OccursOnceAt,
			ad.OccursEveryInterval,
			ad.WeeklyDays,
			ad.MonthlyDayOfMonth,
			ad.MonthlyWeek,
			ad.MonthlyDay,
			ad.StartDate,
			ad.EndDate,
			ad.StartTime,
			ad.EndTime,
			ad.MaxIdleLoopDuration,
			pq.Array(ad.PlayTimeSlots),
			ad.Status,
		).Scan(&ad.ID, &ad.Version, &ad.CreatedAt, &ad.UpdatedAt)
	} else if hasLegacySlot {
		query := `
			INSERT INTO advertisements (
				advertisement_name, description, advertisement_category,
				media_duration, media_url, media_type, lounge_group_name,
				priority, schedule_type, frequency, recurrence_interval,
				occurs_once_at, occurs_every_interval, weekly_days,
				monthly_day_of_month, monthly_week, monthly_day,
				start_date, end_date, start_time, end_time, max_idle_loop_duration,
				play_time_slot, status
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
			) RETURNING id, version, created_at, updated_at`

		err = r.db.QueryRow(
			query,
			ad.AdvertisementName,
			descriptionForStorage,
			ad.AdvertisementCategory,
			ad.MediaDuration,
			ad.MediaURL,
			ad.MediaType,
			ad.LoungeGroupName,
			ad.Priority,
			ad.ScheduleType,
			ad.Frequency,
			ad.RecurrenceInterval,
			ad.OccursOnceAt,
			ad.OccursEveryInterval,
			ad.WeeklyDays,
			ad.MonthlyDayOfMonth,
			ad.MonthlyWeek,
			ad.MonthlyDay,
			ad.StartDate,
			ad.EndDate,
			ad.StartTime,
			ad.EndTime,
			ad.MaxIdleLoopDuration,
			legacySlot,
			ad.Status,
		).Scan(&ad.ID, &ad.Version, &ad.CreatedAt, &ad.UpdatedAt)
	} else {
		query := `
			INSERT INTO advertisements (
				advertisement_name, description, advertisement_category,
				media_duration, media_url, media_type, lounge_group_name,
				priority, schedule_type, frequency, recurrence_interval,
				occurs_once_at, occurs_every_interval, weekly_days,
				monthly_day_of_month, monthly_week, monthly_day,
				start_date, end_date, start_time, end_time, max_idle_loop_duration,
				status
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
			) RETURNING id, version, created_at, updated_at`

		err = r.db.QueryRow(
			query,
			ad.AdvertisementName,
			descriptionForStorage,
			ad.AdvertisementCategory,
			ad.MediaDuration,
			ad.MediaURL,
			ad.MediaType,
			ad.LoungeGroupName,
			ad.Priority,
			ad.ScheduleType,
			ad.Frequency,
			ad.RecurrenceInterval,
			ad.OccursOnceAt,
			ad.OccursEveryInterval,
			ad.WeeklyDays,
			ad.MonthlyDayOfMonth,
			ad.MonthlyWeek,
			ad.MonthlyDay,
			ad.StartDate,
			ad.EndDate,
			ad.StartTime,
			ad.EndTime,
			ad.MaxIdleLoopDuration,
			ad.Status,
		).Scan(&ad.ID, &ad.Version, &ad.CreatedAt, &ad.UpdatedAt)
	}

	if err != nil {
		return err
	}

	// Update the group's advertisement count
	if ad.LoungeGroupName != nil && *ad.LoungeGroupName != "" {
		_, _ = r.db.Exec(`UPDATE advertisement_groups SET no_of_advertisements = no_of_advertisements + 1 WHERE group_name = $1`, ad.LoungeGroupName)
	}

	return nil
}

func (r *AdvertisementRepository) GetAll() ([]models.Advertisement, error) {
	return r.getAllInternal("")
}

// getAllInternal allows optional schedule type filtering for reuse
func (r *AdvertisementRepository) getAllInternal(scheduleType string) ([]models.Advertisement, error) {
	hasSlotsArray, err := r.columnExists("advertisements", "play_time_slots")
	if err != nil {
		return nil, err
	}
	hasLegacySlot, err := r.columnExists("advertisements", "play_time_slot")
	if err != nil {
		return nil, err
	}

	base := `
		SELECT 
			id, advertisement_name, description, advertisement_category,
			media_duration, media_url, media_type, lounge_group_name,
			priority, version, schedule_type, frequency, recurrence_interval,
			occurs_once_at, occurs_every_interval, weekly_days,
			monthly_day_of_month, monthly_week, monthly_day,
			start_date, end_date, start_time, end_time, max_idle_loop_duration,`

	var query string
	if hasSlotsArray {
		query = base + ` COALESCE(play_time_slots, '{}') as play_time_slots,
			status, created_at, updated_at
		FROM advertisements
		WHERE ($1 = '' OR schedule_type = $1)
		ORDER BY created_at DESC`
	} else if hasLegacySlot {
		query = base + ` COALESCE(play_time_slot, '') as play_time_slot,
			status, created_at, updated_at
		FROM advertisements
		WHERE ($1 = '' OR schedule_type = $1)
		ORDER BY created_at DESC`
	} else {
		query = base + ` '' as play_time_slot,
			status, created_at, updated_at
		FROM advertisements
		WHERE ($1 = '' OR schedule_type = $1)
		ORDER BY created_at DESC`
	}

	rows, err := r.db.Query(query, scheduleType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var advertisements []models.Advertisement
	for rows.Next() {
		var ad models.Advertisement
		if hasSlotsArray {
			err = rows.Scan(
				&ad.ID,
				&ad.AdvertisementName,
				&ad.Description,
				&ad.AdvertisementCategory,
				&ad.MediaDuration,
				&ad.MediaURL,
				&ad.MediaType,
				&ad.LoungeGroupName,
				&ad.Priority,
				&ad.Version,
				&ad.ScheduleType,
				&ad.Frequency,
				&ad.RecurrenceInterval,
				&ad.OccursOnceAt,
				&ad.OccursEveryInterval,
				&ad.WeeklyDays,
				&ad.MonthlyDayOfMonth,
				&ad.MonthlyWeek,
				&ad.MonthlyDay,
				&ad.StartDate,
				&ad.EndDate,
				&ad.StartTime,
				&ad.EndTime,
				&ad.MaxIdleLoopDuration,
				pq.Array(&ad.PlayTimeSlots),
				&ad.Status,
				&ad.CreatedAt,
				&ad.UpdatedAt,
			)
		} else {
			var legacySlot string
			err = rows.Scan(
				&ad.ID,
				&ad.AdvertisementName,
				&ad.Description,
				&ad.AdvertisementCategory,
				&ad.MediaDuration,
				&ad.MediaURL,
				&ad.MediaType,
				&ad.LoungeGroupName,
				&ad.Priority,
				&ad.Version,
				&ad.ScheduleType,
				&ad.Frequency,
				&ad.RecurrenceInterval,
				&ad.OccursOnceAt,
				&ad.OccursEveryInterval,
				&ad.WeeklyDays,
				&ad.MonthlyDayOfMonth,
				&ad.MonthlyWeek,
				&ad.MonthlyDay,
				&ad.StartDate,
				&ad.EndDate,
				&ad.StartTime,
				&ad.EndTime,
				&ad.MaxIdleLoopDuration,
				&legacySlot,
				&ad.Status,
				&ad.CreatedAt,
				&ad.UpdatedAt,
			)
			ad.PlayTimeSlots = normalizeSlots(legacySlot)
		}

		cleanDesc, slotsFromDesc := decodePlayTimeSlotsFromDescription(ad.Description)
		ad.Description = cleanDesc
		if len(ad.PlayTimeSlots) == 0 && len(slotsFromDesc) > 0 {
			ad.PlayTimeSlots = slotsFromDesc
		}
		if err != nil {
			return nil, err
		}
		advertisements = append(advertisements, ad)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return advertisements, nil
}

// CheckConflicts fetches advertisements filtered by schedule type (optional)
func (r *AdvertisementRepository) CheckConflicts(scheduleType string) ([]models.Advertisement, error) {
	return r.getAllInternal(scheduleType)
}

func (r *AdvertisementRepository) GetByID(id string) (*models.Advertisement, error) {
	hasSlotsArray, err := r.columnExists("advertisements", "play_time_slots")
	if err != nil {
		return nil, err
	}
	hasLegacySlot, err := r.columnExists("advertisements", "play_time_slot")
	if err != nil {
		return nil, err
	}

	base := `
		SELECT 
			id, advertisement_name, description, advertisement_category,
			media_duration, media_url, media_type, lounge_group_name,
			priority, version, schedule_type, frequency, recurrence_interval,
			occurs_once_at, occurs_every_interval, weekly_days,
			monthly_day_of_month, monthly_week, monthly_day,
			start_date, end_date, start_time, end_time, max_idle_loop_duration,`

	var query string
	if hasSlotsArray {
		query = base + ` COALESCE(play_time_slots, '{}') as play_time_slots,
			status, created_at, updated_at
		FROM advertisements
		WHERE id = $1`
	} else if hasLegacySlot {
		query = base + ` COALESCE(play_time_slot, '') as play_time_slot,
			status, created_at, updated_at
		FROM advertisements
		WHERE id = $1`
	} else {
		query = base + ` '' as play_time_slot,
			status, created_at, updated_at
		FROM advertisements
		WHERE id = $1`
	}

	var ad models.Advertisement
	if hasSlotsArray {
		err = r.db.QueryRow(query, id).Scan(
			&ad.ID,
			&ad.AdvertisementName,
			&ad.Description,
			&ad.AdvertisementCategory,
			&ad.MediaDuration,
			&ad.MediaURL,
			&ad.MediaType,
			&ad.LoungeGroupName,
			&ad.Priority,
			&ad.Version,
			&ad.ScheduleType,
			&ad.Frequency,
			&ad.RecurrenceInterval,
			&ad.OccursOnceAt,
			&ad.OccursEveryInterval,
			&ad.WeeklyDays,
			&ad.MonthlyDayOfMonth,
			&ad.MonthlyWeek,
			&ad.MonthlyDay,
			&ad.StartDate,
			&ad.EndDate,
			&ad.StartTime,
			&ad.EndTime,
			&ad.MaxIdleLoopDuration,
			pq.Array(&ad.PlayTimeSlots),
			&ad.Status,
			&ad.CreatedAt,
			&ad.UpdatedAt,
		)
	} else {
		var legacySlot string
		err = r.db.QueryRow(query, id).Scan(
			&ad.ID,
			&ad.AdvertisementName,
			&ad.Description,
			&ad.AdvertisementCategory,
			&ad.MediaDuration,
			&ad.MediaURL,
			&ad.MediaType,
			&ad.LoungeGroupName,
			&ad.Priority,
			&ad.Version,
			&ad.ScheduleType,
			&ad.Frequency,
			&ad.RecurrenceInterval,
			&ad.OccursOnceAt,
			&ad.OccursEveryInterval,
			&ad.WeeklyDays,
			&ad.MonthlyDayOfMonth,
			&ad.MonthlyWeek,
			&ad.MonthlyDay,
			&ad.StartDate,
			&ad.EndDate,
			&ad.StartTime,
			&ad.EndTime,
			&ad.MaxIdleLoopDuration,
			&legacySlot,
			&ad.Status,
			&ad.CreatedAt,
			&ad.UpdatedAt,
		)
		ad.PlayTimeSlots = normalizeSlots(legacySlot)
	}

	cleanDesc, slotsFromDesc := decodePlayTimeSlotsFromDescription(ad.Description)
	ad.Description = cleanDesc
	if len(ad.PlayTimeSlots) == 0 && len(slotsFromDesc) > 0 {
		ad.PlayTimeSlots = slotsFromDesc
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("advertisement not found")
	}
	return &ad, err
}

func (r *AdvertisementRepository) Update(ad *models.Advertisement) error {
	if ad.PlayTimeSlots == nil {
		ad.PlayTimeSlots = []string{}
	}

	hasSlotsArray, err := r.columnExists("advertisements", "play_time_slots")
	if err != nil {
		return err
	}
	hasLegacySlot, err := r.columnExists("advertisements", "play_time_slot")
	if err != nil {
		return err
	}

	legacySlot := strings.Join(ad.PlayTimeSlots, ",")
	descriptionForStorage := ad.Description
	if !hasSlotsArray && !hasLegacySlot {
		descriptionForStorage = encodePlayTimeSlotsInDescription(ad.Description, ad.PlayTimeSlots)
	}

	var query string
	var args []interface{}
	if hasSlotsArray {
		query = `
			UPDATE advertisements SET 
				advertisement_name = $1,
				description = $2,
				advertisement_category = $3,
				media_duration = $4,
				media_url = $5,
				media_type = $6,
				lounge_group_name = $7,
				priority = $8,
				schedule_type = $9,
				frequency = $10,
				recurrence_interval = $11,
				occurs_once_at = $12,
				occurs_every_interval = $13,
				weekly_days = $14,
				monthly_day_of_month = $15,
				monthly_week = $16,
				monthly_day = $17,
				start_date = $18,
				end_date = $19,
				start_time = $20,
				end_time = $21,
				max_idle_loop_duration = $22,
				status = $23,
				play_time_slots = $24,
				version = version + 1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $25`
		args = []interface{}{
			ad.AdvertisementName,
			descriptionForStorage,
			ad.AdvertisementCategory,
			ad.MediaDuration,
			ad.MediaURL,
			ad.MediaType,
			ad.LoungeGroupName,
			ad.Priority,
			ad.ScheduleType,
			ad.Frequency,
			ad.RecurrenceInterval,
			ad.OccursOnceAt,
			ad.OccursEveryInterval,
			ad.WeeklyDays,
			ad.MonthlyDayOfMonth,
			ad.MonthlyWeek,
			ad.MonthlyDay,
			ad.StartDate,
			ad.EndDate,
			ad.StartTime,
			ad.EndTime,
			ad.MaxIdleLoopDuration,
			ad.Status,
			pq.Array(ad.PlayTimeSlots),
			ad.ID,
		}
	} else if hasLegacySlot {
		query = `
			UPDATE advertisements SET 
				advertisement_name = $1,
				description = $2,
				advertisement_category = $3,
				media_duration = $4,
				media_url = $5,
				media_type = $6,
				lounge_group_name = $7,
				priority = $8,
				schedule_type = $9,
				frequency = $10,
				recurrence_interval = $11,
				occurs_once_at = $12,
				occurs_every_interval = $13,
				weekly_days = $14,
				monthly_day_of_month = $15,
				monthly_week = $16,
				monthly_day = $17,
				start_date = $18,
				end_date = $19,
				start_time = $20,
				end_time = $21,
				max_idle_loop_duration = $22,
				status = $23,
				play_time_slot = $24,
				version = version + 1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $25`
		args = []interface{}{
			ad.AdvertisementName,
			descriptionForStorage,
			ad.AdvertisementCategory,
			ad.MediaDuration,
			ad.MediaURL,
			ad.MediaType,
			ad.LoungeGroupName,
			ad.Priority,
			ad.ScheduleType,
			ad.Frequency,
			ad.RecurrenceInterval,
			ad.OccursOnceAt,
			ad.OccursEveryInterval,
			ad.WeeklyDays,
			ad.MonthlyDayOfMonth,
			ad.MonthlyWeek,
			ad.MonthlyDay,
			ad.StartDate,
			ad.EndDate,
			ad.StartTime,
			ad.EndTime,
			ad.MaxIdleLoopDuration,
			ad.Status,
			legacySlot,
			ad.ID,
		}
	} else {
		query = `
			UPDATE advertisements SET 
				advertisement_name = $1,
				description = $2,
				advertisement_category = $3,
				media_duration = $4,
				media_url = $5,
				media_type = $6,
				lounge_group_name = $7,
				priority = $8,
				schedule_type = $9,
				frequency = $10,
				recurrence_interval = $11,
				occurs_once_at = $12,
				occurs_every_interval = $13,
				weekly_days = $14,
				monthly_day_of_month = $15,
				monthly_week = $16,
				monthly_day = $17,
				start_date = $18,
				end_date = $19,
				start_time = $20,
				end_time = $21,
				max_idle_loop_duration = $22,
				status = $23,
				version = version + 1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $24`
		args = []interface{}{
			ad.AdvertisementName,
			descriptionForStorage,
			ad.AdvertisementCategory,
			ad.MediaDuration,
			ad.MediaURL,
			ad.MediaType,
			ad.LoungeGroupName,
			ad.Priority,
			ad.ScheduleType,
			ad.Frequency,
			ad.RecurrenceInterval,
			ad.OccursOnceAt,
			ad.OccursEveryInterval,
			ad.WeeklyDays,
			ad.MonthlyDayOfMonth,
			ad.MonthlyWeek,
			ad.MonthlyDay,
			ad.StartDate,
			ad.EndDate,
			ad.StartTime,
			ad.EndTime,
			ad.MaxIdleLoopDuration,
			ad.Status,
			ad.ID,
		}
	}

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("advertisement not found")
	}
	return nil
}

func (r *AdvertisementRepository) Delete(id string) error {
	// First, get the advertisement to know which group to update
	var groupName sql.NullString
	err := r.db.QueryRow(`SELECT lounge_group_name FROM advertisements WHERE id = $1`, id).Scan(&groupName)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	result, err := r.db.Exec(`DELETE FROM advertisements WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("advertisement not found")
	}

	// Update the group's advertisement count
	if groupName.Valid && groupName.String != "" {
		_, _ = r.db.Exec(`UPDATE advertisement_groups SET no_of_advertisements = GREATEST(no_of_advertisements - 1, 0) WHERE group_name = $1`, groupName.String)
	}

	return nil
}

// Advertisement Group methods
func (r *AdvertisementRepository) CreateGroup(group *models.AdvertisementGroup) error {
	query := `INSERT INTO advertisement_groups (group_name, lounges, no_of_advertisements)
		VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`

	return r.db.QueryRow(query, group.GroupName, group.Lounges, group.NoOfAdvertisements).
		Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
}

func (r *AdvertisementRepository) GetAllGroups() ([]models.AdvertisementGroup, error) {
	// Use a LEFT JOIN to count advertisements per group dynamically
	query := `
		SELECT 
			ag.id, 
			ag.group_name, 
			ag.lounges, 
			COALESCE(COUNT(a.id), 0) as no_of_advertisements,
			ag.created_at, 
			ag.updated_at
		FROM advertisement_groups ag
		LEFT JOIN advertisements a ON ag.group_name = a.lounge_group_name
		GROUP BY ag.id, ag.group_name, ag.lounges, ag.created_at, ag.updated_at
		ORDER BY ag.created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.AdvertisementGroup
	for rows.Next() {
		var group models.AdvertisementGroup
		err := rows.Scan(&group.ID, &group.GroupName, &group.Lounges,
			&group.NoOfAdvertisements, &group.CreatedAt, &group.UpdatedAt)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func (r *AdvertisementRepository) GetGroupByID(id string) (*models.AdvertisementGroup, error) {
	var group models.AdvertisementGroup
	err := r.db.QueryRow(`SELECT id, group_name, lounges, no_of_advertisements, created_at, updated_at
		FROM advertisement_groups WHERE id = $1`, id).
		Scan(&group.ID, &group.GroupName, &group.Lounges,
			&group.NoOfAdvertisements, &group.CreatedAt, &group.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	return &group, err
}

func (r *AdvertisementRepository) UpdateGroup(group *models.AdvertisementGroup) error {
	result, err := r.db.Exec(`UPDATE advertisement_groups SET group_name = $1, lounges = $2, no_of_advertisements = $3 WHERE id = $4`,
		group.GroupName, group.Lounges, group.NoOfAdvertisements, group.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("group not found")
	}
	return nil
}

func (r *AdvertisementRepository) DeleteGroup(id string) error {
	result, err := r.db.Exec(`DELETE FROM advertisement_groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("group not found")
	}
	return nil
}

func (r *AdvertisementRepository) GetScheduledAdsForDateRange(startDate, endDate *time.Time) ([]models.Advertisement, error) {
	query := `
		SELECT 
			id, advertisement_name, description, advertisement_category,
			media_duration, media_url, media_type, lounge_group_name,
			priority, version, schedule_type, frequency, recurrence_interval,
			occurs_once_at, occurs_every_interval, weekly_days,
			monthly_day_of_month, monthly_week, monthly_day,
			start_date, end_date, start_time, end_time, max_idle_loop_duration, status,
			created_at, updated_at
		FROM advertisements
		WHERE status = 'active'
			AND schedule_type IN ('recurring', 'one-time')
			AND start_date IS NOT NULL
			AND ($2 IS NULL OR start_date <= $2)
			AND ($1 IS NULL OR end_date IS NULL OR end_date >= $1)
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []models.Advertisement
	for rows.Next() {
		var ad models.Advertisement
		err := rows.Scan(
			&ad.ID,
			&ad.AdvertisementName,
			&ad.Description,
			&ad.AdvertisementCategory,
			&ad.MediaDuration,
			&ad.MediaURL,
			&ad.MediaType,
			&ad.LoungeGroupName,
			&ad.Priority,
			&ad.Version,
			&ad.ScheduleType,
			&ad.Frequency,
			&ad.RecurrenceInterval,
			&ad.OccursOnceAt,
			&ad.OccursEveryInterval,
			&ad.WeeklyDays,
			&ad.MonthlyDayOfMonth,
			&ad.MonthlyWeek,
			&ad.MonthlyDay,
			&ad.StartDate,
			&ad.EndDate,
			&ad.StartTime,
			&ad.EndTime,
			&ad.MaxIdleLoopDuration,
			&ad.Status,
			&ad.CreatedAt,
			&ad.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		ads = append(ads, ad)
	}
	return ads, nil
}
