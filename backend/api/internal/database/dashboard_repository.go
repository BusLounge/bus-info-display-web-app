package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"fmt"
	"time"
)

type DashboardRepository struct {
	db *sql.DB
}

func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// GetDashboardStats retrieves the main statistics
func (r *DashboardRepository) GetDashboardStats() (*models.DashboardStats, error) {
	today := time.Now().Format("2006-01-02")

	stats := &models.DashboardStats{}

	// Count today's scheduled trips (based on created_at date)
	err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM scheduled_trips
		WHERE DATE(created_at) = $1
	`, today).Scan(&stats.TodayDepartures)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error counting scheduled trips: %w", err)
	}

	// Count today's active trips (based on created_at date in active_trips table)
	err = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM active_trips
		WHERE DATE(created_at) = $1
	`, today).Scan(&stats.TodayArrivals)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error counting active trips: %w", err)
	}

	// Count delayed arrivals (trips with eta_delayed status or significant delays)
	err = r.db.QueryRow(`
		SELECT COUNT(DISTINCT at.id)
		FROM active_trips at
		JOIN scheduled_trips st ON at.scheduled_trip_id = st.id
		WHERE DATE(st.departure_datetime) = $1
		AND at.status = 'delayed'
	`, today).Scan(&stats.DelayedArrivals)
	if err != nil && err != sql.ErrNoRows {
		stats.DelayedArrivals = 0
	}

	// Count delayed departures
	stats.DelayedDepartures = stats.DelayedArrivals

	return stats, nil
}

// GetRoutesPerLounges retrieves route statistics grouped by lounges
func (r *DashboardRepository) GetRoutesPerLounges() ([]models.RouteStats, error) {
	// Query based on the actual database schema with lounge_routes table
	query := `
		SELECT 
			CONCAT(mr.origin_city, '-', mr.destination_city) as route,
			COUNT(DISTINCT lr.lounge_id) as lounges
		FROM master_routes mr
		LEFT JOIN lounge_routes lr ON mr.id = lr.master_route_id
		GROUP BY mr.origin_city, mr.destination_city
		HAVING COUNT(DISTINCT lr.lounge_id) > 0
		ORDER BY lounges DESC
		LIMIT 10
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching route stats: %w", err)
	}
	defer rows.Close()

	var stats []models.RouteStats
	for rows.Next() {
		var stat models.RouteStats
		if err := rows.Scan(&stat.Route, &stat.Lounges); err != nil {
			return nil, fmt.Errorf("error scanning route stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetAdvertisementCategories retrieves advertisement statistics by schedule type
func (r *DashboardRepository) GetAdvertisementCategories() ([]models.AdvertisementCategoryStats, error) {
	query := `
		SELECT 
			COALESCE(schedule_type, 'Unknown') as schedule_type,
			COUNT(*) as count
		FROM advertisements
		GROUP BY schedule_type
		ORDER BY count DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching advertisement categories: %w", err)
	}
	defer rows.Close()

	var stats []models.AdvertisementCategoryStats
	for rows.Next() {
		var stat models.AdvertisementCategoryStats
		if err := rows.Scan(&stat.Category, &stat.Count); err != nil {
			return nil, fmt.Errorf("error scanning advertisement category: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetAdvertisementStatus retrieves advertisement statistics by status
func (r *DashboardRepository) GetAdvertisementStatus() ([]models.AdvertisementStatusStats, error) {
	query := `
		SELECT 
			COALESCE(status, 'Unknown') as status,
			COUNT(*) as count
		FROM advertisements
		GROUP BY status
		ORDER BY count DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching advertisement status: %w", err)
	}
	defer rows.Close()

	var stats []models.AdvertisementStatusStats
	var total int = 0
	var tempStats []models.AdvertisementStatusStats

	for rows.Next() {
		var stat models.AdvertisementStatusStats
		if err := rows.Scan(&stat.Label, &stat.Value); err != nil {
			return nil, fmt.Errorf("error scanning advertisement status: %w", err)
		}
		total += stat.Value
		tempStats = append(tempStats, stat)
	}

	// Calculate percentages and assign colors
	colorMap := map[string]string{
		"active":    "#059669",
		"scheduled": "#2563eb",
		"expired":   "#dc2626",
		"inactive":  "#6b7280",
	}

	for _, stat := range tempStats {
		if total > 0 {
			stat.Percentage = float64(stat.Value) / float64(total) * 100
		}
		if color, ok := colorMap[stat.Label]; ok {
			stat.Color = color
		} else {
			stat.Color = "#6b7280"
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetMediaCategories retrieves advertisement statistics by media category
func (r *DashboardRepository) GetMediaCategories() ([]models.MediaCategoryStats, error) {
	query := `
		SELECT 
			COALESCE(advertisement_category, 'Unknown') as advertisement_category,
			COUNT(*) as count
		FROM advertisements
		GROUP BY advertisement_category
		ORDER BY count DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching media categories: %w", err)
	}
	defer rows.Close()

	var stats []models.MediaCategoryStats
	var total int = 0
	var tempStats []models.MediaCategoryStats

	for rows.Next() {
		var stat models.MediaCategoryStats
		if err := rows.Scan(&stat.Label, &stat.Value); err != nil {
			return nil, fmt.Errorf("error scanning media category: %w", err)
		}
		total += stat.Value
		tempStats = append(tempStats, stat)
	}

	// Calculate percentages and assign colors
	colorMap := map[string]string{
		"event":             "#2563eb",
		"commercial":        "#7c3aed",
		"seasonal":          "#059669",
		"emergency":         "#dc2626",
		"internal branding": "#f59e0b",
	}

	for _, stat := range tempStats {
		if total > 0 {
			stat.Percentage = float64(stat.Value) / float64(total) * 100
		}
		if color, ok := colorMap[stat.Label]; ok {
			stat.Color = color
		} else {
			stat.Color = "#6b7280"
		}
		stats = append(stats, stat)
	}

	return stats, nil
}
