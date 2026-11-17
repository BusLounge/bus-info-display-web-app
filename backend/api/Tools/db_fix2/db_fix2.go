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

	fmt.Printf("Updating offsets for Route ID: %s\n", routeID)

	// Fetch stops
	rows, err := db.Query("SELECT id, stop_order, stop_name FROM master_route_stops WHERE master_route_id = $1 ORDER BY stop_order", routeID)
	if err != nil {
		log.Fatalf("Failed to query stops: %v", err)
	}
	defer rows.Close()

	var stops []struct {
		id    string
		order int
		name  string
	}
	for rows.Next() {
		var s struct {
			id    string
			order int
			name  string
		}
		if err := rows.Scan(&s.id, &s.order, &s.name); err != nil {
			log.Fatal(err)
		}
		stops = append(stops, s)
	}

	// Update each stop with a progressively increasing offset
	// Let's assume an average of 45 minutes between stops
	for _, s := range stops {
		// Stop order usually starts at 1
		// If stop_order is 1, offset is 0. If 2, offset is 45, etc.
		offsetMinutes := (s.order - 1) * 45
		
		_, err := db.Exec("UPDATE master_route_stops SET arrival_time_offset_minutes = $1 WHERE id = $2", offsetMinutes, s.id)
		if err != nil {
			log.Printf("Failed to update stop %s: %v", s.name, err)
		} else {
			fmt.Printf("Updated stop %s (order %d) with offset: %d minutes\n", s.name, s.order, offsetMinutes)
		}
	}
	fmt.Println("Done updating offsets!")
}
