package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"fmt"
	"sort"

	"github.com/lib/pq"
)

type LoungeAdRepository struct {
	db *sql.DB
}

func NewLoungeAdRepository(db *sql.DB) *LoungeAdRepository {
	return &LoungeAdRepository{db: db}
}

func (r *LoungeAdRepository) EnsureSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS lounge_ads (
			id UUID PRIMARY KEY,
			lounge_id UUID NULL REFERENCES lounges(id) ON DELETE CASCADE,
			advertisement_name TEXT NOT NULL,
			media_url TEXT NOT NULL,
			media_type TEXT NOT NULL,
			duration_seconds INTEGER NOT NULL CHECK (duration_seconds > 0),
			priority TEXT NOT NULL DEFAULT 'normal',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			is_default_for_all BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_lounge_ads_lounge_active
			ON lounge_ads (lounge_id, is_active);

		CREATE INDEX IF NOT EXISTS idx_lounge_ads_default_active
			ON lounge_ads (is_default_for_all, is_active);
	`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("ensure lounge_ads schema: %w", err)
	}
	return nil
}

func (r *LoungeAdRepository) Create(item *models.LoungeAd) error {
	query := `
		INSERT INTO lounge_ads (
			id, lounge_id, advertisement_name, media_url, media_type,
			duration_seconds, priority, is_active, is_default_for_all
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		item.ID,
		item.LoungeID,
		item.AdvertisementName,
		item.MediaURL,
		item.MediaType,
		item.DurationSeconds,
		item.Priority,
		item.IsActive,
		item.IsDefaultForAll,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *LoungeAdRepository) GetAll() ([]models.LoungeAd, error) {
	query := `
		SELECT id, lounge_id, advertisement_name, media_url, media_type,
			duration_seconds, priority, is_active, is_default_for_all,
			created_at, updated_at
		FROM lounge_ads
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.LoungeAd, 0)
	for rows.Next() {
		var item models.LoungeAd
		if err := rows.Scan(
			&item.ID,
			&item.LoungeID,
			&item.AdvertisementName,
			&item.MediaURL,
			&item.MediaType,
			&item.DurationSeconds,
			&item.Priority,
			&item.IsActive,
			&item.IsDefaultForAll,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *LoungeAdRepository) GetForLounge(loungeID string) ([]models.LoungeAd, error) {
	loungeSpecificQuery := `
		SELECT id, lounge_id, advertisement_name, media_url, media_type,
			duration_seconds, priority, is_active, is_default_for_all,
			created_at, updated_at
		FROM lounge_ads
		WHERE is_active = TRUE
			AND lounge_id = $1
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Query(loungeSpecificQuery, loungeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.LoungeAd, 0)
	for rows.Next() {
		var item models.LoungeAd
		if err := rows.Scan(
			&item.ID,
			&item.LoungeID,
			&item.AdvertisementName,
			&item.MediaURL,
			&item.MediaType,
			&item.DurationSeconds,
			&item.Priority,
			&item.IsActive,
			&item.IsDefaultForAll,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *LoungeAdRepository) Update(item *models.LoungeAd) error {
	query := `
		UPDATE lounge_ads
		SET lounge_id = $1,
			advertisement_name = $2,
			media_url = $3,
			media_type = $4,
			duration_seconds = $5,
			priority = $6,
			is_active = $7,
			is_default_for_all = $8,
			updated_at = NOW()
		WHERE id = $9
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		item.LoungeID,
		item.AdvertisementName,
		item.MediaURL,
		item.MediaType,
		item.DurationSeconds,
		item.Priority,
		item.IsActive,
		item.IsDefaultForAll,
		item.ID,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *LoungeAdRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM lounge_ads WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("lounge ad not found")
	}
	return nil
}

func (r *LoungeAdRepository) GetBookedSecondsForLounge(loungeID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(media_duration), 0)
		FROM advertisements
		WHERE status = 'active'
			AND (
				LOWER(COALESCE(lounge_group_name, '')) = LOWER((SELECT lounge_name FROM lounges WHERE id = $1))
				OR lounge_group_name IS NULL
			)
	`
	var booked int
	if err := r.db.QueryRow(query, loungeID).Scan(&booked); err != nil {
		return 0, err
	}
	if booked < 0 {
		booked = 0
	}
	if booked > 24 {
		booked = 24
	}
	return booked, nil
}

func (r *LoungeAdRepository) GetBookedSecondsBreakdownForLounge(loungeID string) (int, map[string]int, []string, error) {
	query := `
		WITH target_lounge AS (
			SELECT lounge_name
			FROM lounges
			WHERE id = $1
		),
		applicable_ads AS (
			SELECT
				COALESCE(NULLIF(LOWER(TRIM(a.schedule_type)), ''), 'unspecified') AS schedule_type,
				GREATEST(COALESCE(a.media_duration, 0), 0) AS media_duration,
				NULLIF(TRIM(a.lounge_group_name), '') AS lounge_group_name
			FROM advertisements a
			JOIN target_lounge tl ON TRUE
			WHERE a.status = 'active'
				AND (
					a.lounge_group_name IS NULL
					OR TRIM(a.lounge_group_name) = ''
					OR LOWER(TRIM(a.lounge_group_name)) = LOWER(tl.lounge_name)
					OR EXISTS (
						SELECT 1
						FROM advertisement_groups ag
						WHERE LOWER(TRIM(ag.group_name)) = LOWER(TRIM(a.lounge_group_name))
							AND (
								LOWER(COALESCE(ag.lounges, '')) LIKE '%' || LOWER(tl.lounge_name) || '%'
							)
					)
				)
		)
		SELECT
			schedule_type,
			SUM(media_duration) AS booked_seconds,
			ARRAY_REMOVE(ARRAY_AGG(DISTINCT lounge_group_name), NULL) AS lounge_groups
		FROM applicable_ads
		GROUP BY schedule_type
		ORDER BY schedule_type;
	`

	rows, err := r.db.Query(query, loungeID)
	if err != nil {
		return 0, nil, nil, err
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	groupSet := make(map[string]struct{})
	total := 0

	for rows.Next() {
		var scheduleType string
		var booked int
		var groups pq.StringArray

		if err := rows.Scan(&scheduleType, &booked, &groups); err != nil {
			return 0, nil, nil, err
		}

		if booked < 0 {
			booked = 0
		}

		breakdown[scheduleType] += booked
		total += booked

		for _, group := range groups {
			if group != "" {
				groupSet[group] = struct{}{}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return 0, nil, nil, err
	}

	groups := make([]string, 0, len(groupSet))
	for group := range groupSet {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	if total < 0 {
		total = 0
	}
	if total > 24 {
		total = 24
	}

	for k, v := range breakdown {
		if v < 0 {
			breakdown[k] = 0
		}
		if breakdown[k] > 24 {
			breakdown[k] = 24
		}
	}

	return total, breakdown, groups, nil
}
