package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"fmt"
	"time"
)

type BroadcastMessageRepository struct {
	db *sql.DB
}

func NewBroadcastMessageRepository(db *sql.DB) *BroadcastMessageRepository {
	return &BroadcastMessageRepository{db: db}
}

func (r *BroadcastMessageRepository) EnsureSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS broadcast_messages (
			id UUID PRIMARY KEY,
			message TEXT NOT NULL,
			priority TEXT NOT NULL DEFAULT 'normal',
			display_duration_seconds INTEGER NOT NULL CHECK (display_duration_seconds > 0),
			frequency_seconds INTEGER NOT NULL CHECK (frequency_seconds > 0),
			start_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			end_at TIMESTAMPTZ NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			show_on_lounge_tv BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_broadcast_messages_active_window
			ON broadcast_messages (is_active, start_at, end_at);
	`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("ensure broadcast_messages schema: %w", err)
	}
	return nil
}

func (r *BroadcastMessageRepository) Create(message *models.BroadcastMessage) error {
	query := `
		INSERT INTO broadcast_messages (
			id, message, priority, display_duration_seconds, frequency_seconds,
			start_at, end_at, is_active, show_on_lounge_tv
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		message.ID,
		message.Message,
		message.Priority,
		message.DisplayDurationSeconds,
		message.FrequencySeconds,
		message.StartAt,
		message.EndAt,
		message.IsActive,
		message.ShowOnLoungeTV,
	).Scan(&message.CreatedAt, &message.UpdatedAt)
}

func (r *BroadcastMessageRepository) GetAll() ([]models.BroadcastMessage, error) {
	query := `
		SELECT id, message, priority, display_duration_seconds, frequency_seconds,
			start_at, end_at, is_active, show_on_lounge_tv, created_at, updated_at
		FROM broadcast_messages
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.BroadcastMessage, 0)
	for rows.Next() {
		var item models.BroadcastMessage
		if err := rows.Scan(
			&item.ID,
			&item.Message,
			&item.Priority,
			&item.DisplayDurationSeconds,
			&item.FrequencySeconds,
			&item.StartAt,
			&item.EndAt,
			&item.IsActive,
			&item.ShowOnLoungeTV,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *BroadcastMessageRepository) GetActiveForTV(now time.Time) ([]models.BroadcastMessage, error) {
	query := `
		SELECT id, message, priority, display_duration_seconds, frequency_seconds,
			start_at, end_at, is_active, show_on_lounge_tv, created_at, updated_at
		FROM broadcast_messages
		WHERE is_active = TRUE
			AND show_on_lounge_tv = TRUE
			AND start_at <= $1
			AND (end_at IS NULL OR end_at >= $1)
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Query(query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.BroadcastMessage, 0)
	for rows.Next() {
		var item models.BroadcastMessage
		if err := rows.Scan(
			&item.ID,
			&item.Message,
			&item.Priority,
			&item.DisplayDurationSeconds,
			&item.FrequencySeconds,
			&item.StartAt,
			&item.EndAt,
			&item.IsActive,
			&item.ShowOnLoungeTV,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *BroadcastMessageRepository) Update(message *models.BroadcastMessage) error {
	query := `
		UPDATE broadcast_messages
		SET message = $1,
			priority = $2,
			display_duration_seconds = $3,
			frequency_seconds = $4,
			start_at = $5,
			end_at = $6,
			is_active = $7,
			show_on_lounge_tv = $8,
			updated_at = NOW()
		WHERE id = $9
		RETURNING created_at, updated_at
	`

	return r.db.QueryRow(
		query,
		message.Message,
		message.Priority,
		message.DisplayDurationSeconds,
		message.FrequencySeconds,
		message.StartAt,
		message.EndAt,
		message.IsActive,
		message.ShowOnLoungeTV,
		message.ID,
	).Scan(&message.CreatedAt, &message.UpdatedAt)
}

func (r *BroadcastMessageRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM broadcast_messages WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("broadcast message not found")
	}
	return nil
}
