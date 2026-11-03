package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres.pttatcukzpceljcrwehk:KQ95tJUYdFX251VR@aws-1-us-east-1.pooler.supabase.com:6543/postgres"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Find the route
	var routeID string
	err = db.QueryRow("SELECT id FROM master_routes WHERE route_number = '21/Col-Kalmunai'").Scan(&routeID)
	if err != nil {
		log.Fatalf("Failed to find route: %v", err)
	}

	fmt.Printf("Route ID: %s\n", routeID)

	// Fetch stops
	rows, err := db.Query("SELECT id, stop_order, stop_name, arrival_time_offset_minutes FROM master_route_stops WHERE master_route_id = $1 ORDER BY stop_order", routeID)
	if err != nil {
		log.Fatalf("Failed to query stops: %v", err)
	}
	defer rows.Close()

	fmt.Println("Stops on route:")
	for rows.Next() {
		var id string
		var order int
		var name string
		var offset sql.NullInt64
		if err := rows.Scan(&id, &order, &name, &offset); err != nil {
			log.Fatal(err)
		}
		offsetVal := "NULL"
		if offset.Valid {
			offsetVal = fmt.Sprintf("%d", offset.Int64)
		}
		fmt.Printf("Order: %d, Stop: %s, Offset: %s (ID: %s)\n", order, name, offsetVal, id)
	}

	// Fetch lounges on this route
	lRows, err := db.Query(`
		SELECT l.lounge_name, l.id, lr.stop_before_id, lr.stop_after_id 
		FROM lounges l 
		JOIN lounge_routes lr ON l.id = lr.lounge_id 
		WHERE lr.master_route_id = $1
	`, routeID)
	if err != nil {
		log.Fatalf("Failed to query lounges: %v", err)
	}
	defer lRows.Close()

	fmt.Println("\nLounges on route:")
	for lRows.Next() {
		var name, id string
		var sbID, saID sql.NullString
		if err := lRows.Scan(&name, &id, &sbID, &saID); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Lounge: %s (ID: %s)\n  Stop Before ID: %v\n  Stop After ID: %v\n", name, id, sbID.String, saID.String)
	}
}
