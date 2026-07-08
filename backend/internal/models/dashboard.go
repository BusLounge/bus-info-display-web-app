package models

// DashboardStats represents the top-level statistics
type DashboardStats struct {
	TodayDepartures   int `json:"todayDepartures"`
	TodayArrivals     int `json:"todayArrivals"`
	DelayedArrivals   int `json:"delayedArrivals"`
	DelayedDepartures int `json:"delayedDepartures"`
}

// RouteStats represents statistics by route
type RouteStats struct {
	Route   string `json:"route" db:"route"`
	Count   int    `json:"count" db:"count"`
	Lounges int    `json:"lounges" db:"lounges"`
}

// LoungeStats represents statistics by lounge
type LoungeStats struct {
	Lounge string `json:"lounge" db:"lounge"`
	Trips  int    `json:"trips" db:"trips"`
}

// AdvertisementStatusStats represents advertisement status distribution
type AdvertisementStatusStats struct {
	Label      string  `json:"label" db:"status"`
	Value      int     `json:"value" db:"count"`
	Percentage float64 `json:"percentage"`
	Color      string  `json:"color"`
}

// AdvertisementCategoryStats represents advertisement category distribution
type AdvertisementCategoryStats struct {
	Category string `json:"category" db:"schedule_type"`
	Count    int    `json:"count" db:"count"`
}

// MediaCategoryStats represents media category distribution
type MediaCategoryStats struct {
	Label      string  `json:"label" db:"advertisement_category"`
	Value      int     `json:"value" db:"count"`
	Percentage float64 `json:"percentage"`
	Color      string  `json:"color"`
}

// DashboardResponse represents the complete dashboard data
type DashboardResponse struct {
	Stats                   DashboardStats               `json:"stats"`
	RoutesPerLounges        []RouteStats                 `json:"routesPerLounges"`
	AdvertisementCategories []AdvertisementCategoryStats `json:"advertisementCategories"`
	AdvertisementStatus     []AdvertisementStatusStats   `json:"advertisementStatus"`
	MediaCategories         []MediaCategoryStats         `json:"mediaCategories"`
}
