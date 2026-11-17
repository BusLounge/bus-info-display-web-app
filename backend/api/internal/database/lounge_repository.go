package database

import (
	"bus-schedule-lounge/internal/models"
	"database/sql"
	"fmt"
)

type LoungeRepository struct {
	db *sql.DB
}

func NewLoungeRepository(db *sql.DB) *LoungeRepository {
	return &LoungeRepository{db: db}
}

// GetAllLounges retrieves all lounges from the database
func (r *LoungeRepository) GetAllLounges() ([]models.Lounge, error) {
	query := `
		SELECT 
			l.id,
			l.lounge_owner_id,
			l.lounge_name,
			l.description,
			l.address,
			lo.district,
			l.state,
			l.country,
			l.postal_code,
			l.latitude,
			l.longitude,
			l.contact_phone,
			l.price_1_hour,
			l.price_2_hours,
			l.price_3_hours,
			l.price_until_bus,
			l.amenities,
			l.images,
			l.status,
			l.is_operational,
			l.average_rating,
			l.capacity,
			l.owner_id,
			l.marketplace_category_id,
			l.verification_note,
			l.total_staff,
			l.created_at,
			l.updated_at
		FROM lounges l
		LEFT JOIN lounge_owners lo ON l.lounge_owner_id = lo.id
		ORDER BY l.lounge_name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error fetching lounges: %w", err)
	}
	defer rows.Close()

	var lounges []models.Lounge
	for rows.Next() {
		var lounge models.Lounge
		err := rows.Scan(
			&lounge.ID,
			&lounge.LoungeOwnerID,
			&lounge.LoungeName,
			&lounge.Description,
			&lounge.Address,
			&lounge.District,
			&lounge.State,
			&lounge.Country,
			&lounge.PostalCode,
			&lounge.Latitude,
			&lounge.Longitude,
			&lounge.ContactPhone,
			&lounge.Price1Hour,
			&lounge.Price2Hours,
			&lounge.Price3Hours,
			&lounge.PriceUntilBus,
			&lounge.Amenities,
			&lounge.Images,
			&lounge.Status,
			&lounge.IsOperational,
			&lounge.AverageRating,
			&lounge.Capacity,
			&lounge.OwnerID,
			&lounge.MarketplaceCategoryID,
			&lounge.VerificationNote,
			&lounge.TotalStaff,
			&lounge.CreatedAt,
			&lounge.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning lounge: %w", err)
		}
		lounges = append(lounges, lounge)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating lounges: %w", err)
	}

	return lounges, nil
}

// GetLoungeByID retrieves a single lounge by ID
func (r *LoungeRepository) GetLoungeByID(id string) (*models.Lounge, error) {
	query := `
		SELECT 
			l.id,
			l.lounge_owner_id,
			l.lounge_name,
			l.description,
			l.address,
			lo.district,
			l.state,
			l.country,
			l.postal_code,
			l.latitude,
			l.longitude,
			l.contact_phone,
			l.price_1_hour,
			l.price_2_hours,
			l.price_3_hours,
			l.price_until_bus,
			l.amenities,
			l.images,
			l.status,
			l.is_operational,
			l.average_rating,
			l.capacity,
			l.owner_id,
			l.marketplace_category_id,
			l.verification_note,
			l.total_staff,
			l.created_at,
			l.updated_at
		FROM lounges l
		LEFT JOIN lounge_owners lo ON l.lounge_owner_id = lo.id
		WHERE l.id = $1`

	var lounge models.Lounge
	err := r.db.QueryRow(query, id).Scan(
		&lounge.ID,
		&lounge.LoungeOwnerID,
		&lounge.LoungeName,
		&lounge.Description,
		&lounge.Address,
		&lounge.District,
		&lounge.State,
		&lounge.Country,
		&lounge.PostalCode,
		&lounge.Latitude,
		&lounge.Longitude,
		&lounge.ContactPhone,
		&lounge.Price1Hour,
		&lounge.Price2Hours,
		&lounge.Price3Hours,
		&lounge.PriceUntilBus,
		&lounge.Amenities,
		&lounge.Images,
		&lounge.Status,
		&lounge.IsOperational,
		&lounge.AverageRating,
		&lounge.Capacity,
		&lounge.OwnerID,
		&lounge.MarketplaceCategoryID,
		&lounge.VerificationNote,
		&lounge.TotalStaff,
		&lounge.CreatedAt,
		&lounge.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lounge not found")
	}
	if err != nil {
		return nil, fmt.Errorf("error fetching lounge: %w", err)
	}

	return &lounge, nil
}

func (r *LoungeRepository) ValidateLoungeRouteSegments(loungeID string) (*models.LoungeRouteValidationResponse, error) {
	query := `
		SELECT
			lr.id,
			lr.master_route_id,
			mr.route_number,
			lr.stop_before_id,
			sb.stop_name,
			sb.stop_order,
			lr.stop_after_id,
			sa.stop_name,
			sa.stop_order,
			CASE
				WHEN sb.id IS NULL THEN false
				WHEN sa.id IS NULL THEN false
				WHEN sb.master_route_id <> lr.master_route_id THEN false
				WHEN sa.master_route_id <> lr.master_route_id THEN false
				WHEN sb.stop_order >= sa.stop_order THEN false
				ELSE true
			END AS is_valid,
			CASE
				WHEN sb.id IS NULL THEN 'stop_before_id does not exist'
				WHEN sa.id IS NULL THEN 'stop_after_id does not exist'
				WHEN sb.master_route_id <> lr.master_route_id THEN 'stop_before_id is not in the selected master route'
				WHEN sa.master_route_id <> lr.master_route_id THEN 'stop_after_id is not in the selected master route'
				WHEN sb.stop_order >= sa.stop_order THEN 'stop_before must come before stop_after in stop order'
				ELSE 'valid'
			END AS reason
		FROM lounge_routes lr
		LEFT JOIN master_routes mr ON mr.id = lr.master_route_id
		LEFT JOIN master_route_stops sb ON sb.id = lr.stop_before_id
		LEFT JOIN master_route_stops sa ON sa.id = lr.stop_after_id
		WHERE lr.lounge_id = $1
		ORDER BY mr.route_number, lr.id
	`

	rows, err := r.db.Query(query, loungeID)
	if err != nil {
		return nil, fmt.Errorf("error validating lounge route segments: %w", err)
	}
	defer rows.Close()

	response := &models.LoungeRouteValidationResponse{
		LoungeID: loungeID,
		IsValid:  true,
		Segments: []models.LoungeRouteSegmentValidation{},
	}

	for rows.Next() {
		segment := models.LoungeRouteSegmentValidation{}
		if err := rows.Scan(
			&segment.LoungeRouteID,
			&segment.MasterRouteID,
			&segment.RouteNumber,
			&segment.StopBeforeID,
			&segment.StopBeforeName,
			&segment.StopBeforeOrder,
			&segment.StopAfterID,
			&segment.StopAfterName,
			&segment.StopAfterOrder,
			&segment.IsValid,
			&segment.Reason,
		); err != nil {
			return nil, fmt.Errorf("error scanning lounge route segment validation: %w", err)
		}

		if !segment.IsValid {
			response.IsValid = false
		}

		response.Segments = append(response.Segments, segment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating lounge route segment validation rows: %w", err)
	}

	if len(response.Segments) == 0 {
		response.IsValid = false
	}

	return response, nil
}
