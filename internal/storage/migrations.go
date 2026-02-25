package storage

import (
	"fmt"
	"log"
)

// RunMigrations creates all the database tables if they do not already exist.
// This runs automatically every time the application starts.
// Think of it like a construction worker who checks if all rooms exist
// and builds any that are missing.
func RunMigrations() error {
	log.Println("Running database migrations...")

	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "create_farmers_table",
			sql: `
			CREATE TABLE IF NOT EXISTS farmers (
				id          TEXT PRIMARY KEY,
				full_name   TEXT NOT NULL,
				phone       TEXT NOT NULL,
				location    TEXT NOT NULL,
				version     INTEGER NOT NULL DEFAULT 1,
				created_at  DATETIME NOT NULL,
				updated_at  DATETIME NOT NULL,
				deleted     BOOLEAN NOT NULL DEFAULT 0
			);`,
		},
		{
			name: "create_produce_table",
			sql: `
			CREATE TABLE IF NOT EXISTS produce (
				id                  TEXT PRIMARY KEY,
				farmer_id           TEXT NOT NULL DEFAULT '',
				category            TEXT NOT NULL,
				produce_name        TEXT NOT NULL,
				quantity            REAL NOT NULL DEFAULT 0,
				quantity_sold       REAL NOT NULL DEFAULT 0,
				quantity_rejected   REAL NOT NULL DEFAULT 0,
				quantity_remaining  REAL NOT NULL DEFAULT 0,
				price_per_unit      REAL NOT NULL DEFAULT 0,
				total_received      REAL NOT NULL DEFAULT 0,
				unit                TEXT NOT NULL,
				location            TEXT NOT NULL DEFAULT '',
				notes               TEXT NOT NULL DEFAULT '',
				version             INTEGER NOT NULL DEFAULT 1,
				created_at          DATETIME NOT NULL,
				updated_at          DATETIME NOT NULL,
				deleted             BOOLEAN NOT NULL DEFAULT 0
			);`,
		},
		{
			name: "create_listings_table",
			sql: `
			CREATE TABLE IF NOT EXISTS listings (
				id                TEXT PRIMARY KEY,
				produce_id        TEXT NOT NULL DEFAULT '',
				produce_name      TEXT NOT NULL DEFAULT '',
				farmer_id         TEXT NOT NULL DEFAULT '',
				quantity_listed   REAL NOT NULL DEFAULT 0,
				asking_price      REAL NOT NULL DEFAULT 0,
				location          TEXT NOT NULL DEFAULT '',
				contact           TEXT NOT NULL DEFAULT '',
				status            TEXT NOT NULL DEFAULT 'available',
				buyer_name        TEXT NOT NULL DEFAULT '',
				buyer_contact     TEXT NOT NULL DEFAULT '',
				buyer_location    TEXT NOT NULL DEFAULT '',
				notes             TEXT NOT NULL DEFAULT '',
				version           INTEGER NOT NULL DEFAULT 1,
				created_at        DATETIME NOT NULL,
				updated_at        DATETIME NOT NULL,
				deleted           BOOLEAN NOT NULL DEFAULT 0
			);`,
		},
		{
			name: "create_sync_queue_table",
			sql: `
			CREATE TABLE IF NOT EXISTS sync_queue (
				id            TEXT PRIMARY KEY,
				entity_type   TEXT NOT NULL,
				operation     TEXT NOT NULL,
				payload       TEXT NOT NULL,
				status        TEXT NOT NULL DEFAULT 'pending',
				retry_count   INTEGER NOT NULL DEFAULT 0,
				last_attempt  DATETIME,
				created_at    DATETIME NOT NULL
			);`,
		},
	}

	// Run each migration one by one
	for _, migration := range migrations {
		log.Printf("Running migration: %s", migration.name)

		_, err := DB.Exec(migration.sql)
		if err != nil {
			return fmt.Errorf("error running migration %s: %w", migration.name, err)
		}

		log.Printf("Migration complete: %s", migration.name)
	}

	log.Println("All migrations completed successfully")
	return nil
}
