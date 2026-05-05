package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

const dbFile = "dilmeter.db"

// InitDB initializes the SQLite database, runs migrations, and handles JSON import.
func InitDB() {
	var err error
	db, err = sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatal(err)
	}

	// Optimize SQLite performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;"); err != nil {
		log.Println("Warning: Failed to set PRAGMA:", err)
	}

	if err := createSchema(); err != nil {
		log.Fatal("Failed to create schema:", err)
	}

	if err := migrateFromJSON(); err != nil {
		log.Println("Warning: JSON Migration failed (skipping):", err)
	}

}

func createSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY,
			name TEXT,
			race_id INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS system_metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// migrateFromJSON checks for legacy JSON files and imports them if the DB is empty.
func migrateFromJSON() error {
	// Check if migration is needed (simplest check: players table is empty)
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM players").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Already migrated or populated
	}

	// Use a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Migrate PlayerCache
	if err := migratePlayers(tx); err != nil {
		// Only log, don't fail entire init? No, we should fail migration but let app start blank?
		// Plan said: "If any part fails, we rollback".
		return fmt.Errorf("migrating players: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Println("[INFO] storage: Migration complete. Legacy files backed up.")
	return nil
}

func migratePlayers(tx *sql.Tx) error {
	path := "player_cache.json" // Assuming root
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = "build/player_cache.json" // Check build dir
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil // No file
		}
	}

	log.Println("[INFO] storage: Migrating player_cache.json...")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Filter NPCs logic duplicated here or just rely on the map
	var players map[uint64]*PlayerInfo
	if err := json.Unmarshal(data, &players); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO players 
		(id, name, race_id) 
		VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	for id, p := range players {
		if strings.HasPrefix(p.Name, "_") {
			continue // Skip NPCs
		}

		_, err := stmt.Exec(id, p.Name, p.RaceId)
		if err != nil {
			return err
		}
		count++
	}

	// Rename backup
	os.Rename(path, path+".bak")
	log.Printf("[INFO] storage: Migrated %d players.\n", count)
	return nil
}
