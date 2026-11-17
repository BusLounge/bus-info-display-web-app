package models

import "time"

// Lounge represents a lounge entity in the database
type Lounge struct {
	ID                    string    `json:"id" db:"id"`
	LoungeOwnerID         *string   `json:"loungeOwnerId,omitempty" db:"lounge_owner_id"`
	LoungeName            string    `json:"loungeName" db:"lounge_name"`
	Description           *string   `json:"description,omitempty" db:"description"`
	Address               *string   `json:"address,omitempty" db:"address"`
	District              *string   `json:"district,omitempty" db:"district"`
	State                 *string   `json:"state,omitempty" db:"state"`
	Country               *string   `json:"country,omitempty" db:"country"`
	PostalCode            *string   `json:"postalCode,omitempty" db:"postal_code"`
	Latitude              *float64  `json:"latitude,omitempty" db:"latitude"`
	Longitude             *float64  `json:"longitude,omitempty" db:"longitude"`
	ContactPhone          *string   `json:"contactPhone,omitempty" db:"contact_phone"`
	Price1Hour            *float64  `json:"price1Hour,omitempty" db:"price_1_hour"`
	Price2Hours           *float64  `json:"price2Hours,omitempty" db:"price_2_hours"`
	Price3Hours           *float64  `json:"price3Hours,omitempty" db:"price_3_hours"`
	PriceUntilBus         *float64  `json:"priceUntilBus,omitempty" db:"price_until_bus"`
	Amenities             *string   `json:"amenities,omitempty" db:"amenities"`
	Images                *string   `json:"images,omitempty" db:"images"`
	Status                string    `json:"status" db:"status"`
	IsOperational         bool      `json:"isOperational" db:"is_operational"`
	AverageRating         *float64  `json:"averageRating,omitempty" db:"average_rating"`
	Capacity              *int      `json:"capacity,omitempty" db:"capacity"`
	OwnerID               *string   `json:"ownerId,omitempty" db:"owner_id"`
	MarketplaceCategoryID *string   `json:"marketplaceCategoryId,omitempty" db:"marketplace_category_id"`
	VerificationNote      *string   `json:"verificationNote,omitempty" db:"verification_note"`
	TotalStaff            *int      `json:"totalStaff,omitempty" db:"total_staff"`
	CreatedAt             time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time `json:"updatedAt" db:"updated_at"`
}

type LoungeRouteSegmentValidation struct {
	LoungeRouteID  string  `json:"loungeRouteId"`
	MasterRouteID  string  `json:"masterRouteId"`
	RouteNumber    *string `json:"routeNumber,omitempty"`
	StopBeforeID   string  `json:"stopBeforeId"`
	StopBeforeName *string `json:"stopBeforeName,omitempty"`
	StopBeforeOrder *int   `json:"stopBeforeOrder,omitempty"`
	StopAfterID    string  `json:"stopAfterId"`
	StopAfterName  *string `json:"stopAfterName,omitempty"`
	StopAfterOrder *int    `json:"stopAfterOrder,omitempty"`
	IsValid        bool    `json:"isValid"`
	Reason         string  `json:"reason"`
}

type LoungeRouteValidationResponse struct {
	LoungeID string                        `json:"loungeId"`
	IsValid  bool                          `json:"isValid"`
	Segments []LoungeRouteSegmentValidation `json:"segments"`
}
