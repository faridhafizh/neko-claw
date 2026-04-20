package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDatabase() error {
	dataDir := filepath.Join("data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "neko.db")
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: Could not enable WAL mode: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		log.Printf("Warning: Could not enable foreign keys: %v", err)
	}

	// Create schema
	if err := createSchema(); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Migrate from JSON if needed
	if err := migrateFromJSON(); err != nil {
		log.Printf("Warning: JSON migration had issues: %v", err)
	}

	return nil
}

func createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS chat_sessions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT 'New Chat',
		archived INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id);

	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'facts',
		priority INTEGER DEFAULT 3,
		tags TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category);
	CREATE INDEX IF NOT EXISTS idx_memories_priority ON memories(priority DESC);

	CREATE TABLE IF NOT EXISTS active_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	_, err := db.Exec(schema)
	return err
}

func migrateFromJSON() error {
	// Check if migration was already done
	var migrated string
	err := db.QueryRow("SELECT value FROM active_config WHERE key = 'json_migrated'").Scan(&migrated)
	if err == nil && migrated == "true" {
		return nil // Already migrated
	}

	log.Println("Starting JSON → SQLite migration...")

	// Migrate settings
	if err := migrateSettings(); err != nil {
		log.Printf("Settings migration: %v", err)
	}

	// Migrate chat history
	if err := migrateChatHistory(); err != nil {
		log.Printf("Chat history migration: %v", err)
	}

	// Migrate memories
	if err := migrateMemories(); err != nil {
		log.Printf("Memories migration: %v", err)
	}

	// Migrate souls (active soul)
	if err := migrateSouls(); err != nil {
		log.Printf("Souls migration: %v", err)
	}

	// Mark migration as done
	db.Exec("INSERT OR REPLACE INTO active_config (key, value) VALUES ('json_migrated', 'true')")
	log.Println("JSON → SQLite migration complete!")
	return nil
}

func migrateSettings() error {
	data, err := os.ReadFile(filepath.Join("data", "settings.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var settings struct {
		ApiKey string `json:"apiKey"`
		Model  string `json:"model"`
		ApiUrl string `json:"apiUrl"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	stmt.Exec("apiKey", settings.ApiKey)
	stmt.Exec("model", settings.Model)
	stmt.Exec("apiUrl", settings.ApiUrl)

	return tx.Commit()
}

func migrateChatHistory() error {
	data, err := os.ReadFile(filepath.Join("data", "chat_history.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var store struct {
		Sessions []struct {
			ID        string    `json:"id"`
			Title     string    `json:"title"`
			Archived  bool      `json:"archived"`
			CreatedAt time.Time `json:"createdAt"`
			UpdatedAt time.Time `json:"updatedAt"`
			Messages  []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	sessionStmt, err := tx.Prepare("INSERT OR IGNORE INTO chat_sessions (id, title, archived, created_at, updated_at) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer sessionStmt.Close()

	msgStmt, err := tx.Prepare("INSERT INTO chat_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer msgStmt.Close()

	for _, s := range store.Sessions {
		archived := 0
		if s.Archived {
			archived = 1
		}
		sessionStmt.Exec(s.ID, s.Title, archived, s.CreatedAt, s.UpdatedAt)

		// Distribute message timestamps evenly between createdAt and updatedAt
		msgCount := len(s.Messages)
		for i, m := range s.Messages {
			var msgTime time.Time
			if msgCount <= 1 {
				msgTime = s.CreatedAt
			} else {
				// Interpolate timestamps
				fraction := float64(i) / float64(msgCount-1)
				diff := s.UpdatedAt.Sub(s.CreatedAt)
				msgTime = s.CreatedAt.Add(time.Duration(float64(diff) * fraction))
			}
			msgStmt.Exec(s.ID, m.Role, m.Content, msgTime)
		}
	}

	return tx.Commit()
}

func migrateMemories() error {
	data, err := os.ReadFile(filepath.Join("data", "memories.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var store struct {
		Memories []struct {
			ID        string    `json:"id"`
			Content   string    `json:"content"`
			Category  string    `json:"category"`
			Priority  int       `json:"priority"`
			Tags      []string  `json:"tags"`
			CreatedAt time.Time `json:"createdAt"`
			LastUsed  time.Time `json:"lastUsed"`
		} `json:"memories"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO memories (id, content, category, priority, tags, created_at, last_used) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range store.Memories {
		tagsJSON, _ := json.Marshal(m.Tags)
		stmt.Exec(m.ID, m.Content, m.Category, m.Priority, string(tagsJSON), m.CreatedAt, m.LastUsed)
	}

	return tx.Commit()
}

func migrateSouls() error {
	data, err := os.ReadFile(filepath.Join("data", "souls.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var store struct {
		ActiveSoul string `json:"activeSoul"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return err
	}

	if store.ActiveSoul != "" {
		db.Exec("INSERT OR REPLACE INTO active_config (key, value) VALUES ('active_soul', ?)", store.ActiveSoul)
	}

	return nil
}

// ── Settings DB helpers ────────────────────────────────────────────────────

func dbGetSetting(key string) string {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

func dbSetSetting(key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// ── Active Config helpers ──────────────────────────────────────────────────

func dbGetConfig(key string) string {
	var value string
	err := db.QueryRow("SELECT value FROM active_config WHERE key = ?", key).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}

func dbSetConfig(key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO active_config (key, value) VALUES (?, ?)", key, value)
	return err
}

// ── Helpers ────────────────────────────────────────────────────────────────

func tagsToJSON(tags []string) string {
	if tags == nil {
		return "[]"
	}
	data, _ := json.Marshal(tags)
	return string(data)
}

func jsonToTags(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(s), &tags); err != nil {
		return []string{}
	}
	return tags
}
