package main

import (
	"fmt"
	"strings"
	"time"
)

type Memory struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Category  string    `json:"category"` // facts, preferences, events, commands
	Priority  int       `json:"priority"` // 1-5, higher = more important
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  time.Time `json:"lastUsed"`
	Tags      []string  `json:"tags"`
}

type MemoryStore struct{}

var memoryStore *MemoryStore

func initMemoryStore() error {
	memoryStore = &MemoryStore{}
	return nil
}

func (ms *MemoryStore) AddMemory(content, category string, priority int, tags []string) error {
	if priority < 1 {
		priority = 1
	}
	if priority > 5 {
		priority = 5
	}

	id := generateID()
	now := time.Now()
	tagsStr := tagsToJSON(tags)

	_, err := db.Exec(
		"INSERT INTO memories (id, content, category, priority, tags, created_at, last_used) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, content, category, priority, tagsStr, now, now,
	)
	return err
}

func (ms *MemoryStore) GetMemories(category string, limit int) []Memory {
	query := "SELECT id, content, category, priority, tags, created_at, last_used FROM memories"
	args := []interface{}{}

	if category != "" {
		query += " WHERE category = ?"
		args = append(args, category)
	}

	query += " ORDER BY priority DESC, created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return []Memory{}
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		var tagsStr string
		if err := rows.Scan(&m.ID, &m.Content, &m.Category, &m.Priority, &tagsStr, &m.CreatedAt, &m.LastUsed); err != nil {
			continue
		}
		m.Tags = jsonToTags(tagsStr)
		memories = append(memories, m)
	}

	if memories == nil {
		memories = []Memory{}
	}
	return memories
}

func (ms *MemoryStore) SearchMemories(query string, limit int) []Memory {
	query = strings.ToLower(query)
	searchPattern := "%" + query + "%"

	// Search in content and tags
	sqlQuery := `
		SELECT id, content, category, priority, tags, created_at, last_used 
		FROM memories 
		WHERE LOWER(content) LIKE ? OR LOWER(tags) LIKE ?
		ORDER BY priority DESC, created_at DESC
	`
	args := []interface{}{searchPattern, searchPattern}

	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return []Memory{}
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		var tagsStr string
		if err := rows.Scan(&m.ID, &m.Content, &m.Category, &m.Priority, &tagsStr, &m.CreatedAt, &m.LastUsed); err != nil {
			continue
		}
		m.Tags = jsonToTags(tagsStr)
		memories = append(memories, m)
	}

	if memories == nil {
		memories = []Memory{}
	}
	return memories
}

func (ms *MemoryStore) DeleteMemory(id string) error {
	_, err := db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}

func (ms *MemoryStore) ClearMemories(category string) error {
	if category == "" {
		_, err := db.Exec("DELETE FROM memories")
		return err
	}
	_, err := db.Exec("DELETE FROM memories WHERE category = ?", category)
	return err
}

func (ms *MemoryStore) GetMemoryStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total": 0,
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM memories").Scan(&total); err == nil {
		stats["total"] = total
	}

	categoryCount := make(map[string]int)
	rows, err := db.Query("SELECT category, COUNT(*) FROM memories GROUP BY category")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat string
			var count int
			if rows.Scan(&cat, &count) == nil {
				categoryCount[cat] = count
			}
		}
	}
	stats["byCategory"] = categoryCount

	return stats
}

func (ms *MemoryStore) UpdateLastUsed(id string) {
	db.Exec("UPDATE memories SET last_used = ? WHERE id = ?", time.Now(), id)
}
