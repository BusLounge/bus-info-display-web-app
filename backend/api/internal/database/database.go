package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(databaseURL string) (*Database, error) {
	// Add parameter to disable prepared statements caching
	if databaseURL != "" && !strings.Contains(databaseURL, "default_query_exec_mode") {
		separator := "?"
		if strings.Contains(databaseURL, "?") {
			separator = "&"
		}
		databaseURL = databaseURL + separator + "default_query_exec_mode=simple_protocol"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("Successfully connected to database")
	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	return d.DB.Close()
}
