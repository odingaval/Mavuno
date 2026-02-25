package storage

import (
	//"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// DB is the database connection that the entire application will use. Defined here so every other file in storage can access it.
var DB *sqlx.DB

// InitDB opens a connection to the SQLite database.It takes the path where the database file should be created or opened.

func InitDB(dataSourceName string) error {
	var err error

	// Open the database connection
	// If the file does not exist, SQLite will create it automatically
	DB, err = sqlx.Open("sqlite3", dataSourceName)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Ping the database to verify the connection is working
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	// Enable WAL mode for better performance with concurrent reads and writes
	// WAL means Write Ahead Logging - it prevents the database from locking
	// when multiple operations happen at the same time
	_, err = DB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return fmt.Errorf("error setting WAL mode: %w", err)
	}

	// Enable foreign key support. This ensures ProduceID in listings always points to a real produce record
	_, err = DB.Exec("PRAGMA foreign_keys=ON;")
	if err != nil {
		return fmt.Errorf("error enabling foreign keys: %w", err)
	}

	log.Println("Database connection established successfully")
	return nil
}

// CloseDB safely closes the database connection when the app shuts down.
func CloseDB() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		log.Println("Database connection closed")
	}
}

// GetDB returns the active database connection. Other parts of the application call this to get access to the database.
func GetDB() (*sqlx.DB, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return DB, nil
}

// HealthCheck verifies the database is still responding. This is used by the health check endpoint to confirm the DB is alive.
func HealthCheck() error {
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	return nil
}