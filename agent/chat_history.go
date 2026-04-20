package main

import (
	"fmt"
	"time"
)

type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type ChatSession struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Messages  []ChatMessage `json:"messages"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Archived  bool          `json:"archived"`
}

// LastMessagePreview returns a truncated preview of the last non-system message.
func (s *ChatSession) LastMessagePreview() string {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		m := s.Messages[i]
		if m.Role == "user" || m.Role == "assistant" {
			preview := m.Content
			if len(preview) > 60 {
				preview = preview[:60] + "…"
			}
			return preview
		}
	}
	return ""
}

type ChatHistoryStore struct{}

var chatHistoryStore *ChatHistoryStore

func initChatHistoryStore() error {
	chatHistoryStore = &ChatHistoryStore{}
	return nil
}

func (s *ChatHistoryStore) CreateSession(title string) *ChatSession {
	id := generateID()
	now := time.Now()

	_, err := db.Exec(
		"INSERT INTO chat_sessions (id, title, archived, created_at, updated_at) VALUES (?, ?, 0, ?, ?)",
		id, title, now, now,
	)
	if err != nil {
		fmt.Printf("Error creating session: %v\n", err)
		return nil
	}

	return &ChatSession{
		ID:        id,
		Title:     title,
		Messages:  []ChatMessage{},
		CreatedAt: now,
		UpdatedAt: now,
		Archived:  false,
	}
}

func (s *ChatHistoryStore) GetSession(id string) *ChatSession {
	var session ChatSession
	var archived int
	err := db.QueryRow(
		"SELECT id, title, archived, created_at, updated_at FROM chat_sessions WHERE id = ?", id,
	).Scan(&session.ID, &session.Title, &archived, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil
	}
	session.Archived = archived != 0

	// Load messages
	rows, err := db.Query(
		"SELECT role, content, created_at FROM chat_messages WHERE session_id = ? ORDER BY id ASC", id,
	)
	if err != nil {
		return &session
	}
	defer rows.Close()

	session.Messages = []ChatMessage{}
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.Role, &msg.Content, &msg.CreatedAt); err == nil {
			session.Messages = append(session.Messages, msg)
		}
	}

	return &session
}

func (s *ChatHistoryStore) GetActiveSession() *ChatSession {
	var id string
	err := db.QueryRow(
		"SELECT id FROM chat_sessions WHERE archived = 0 ORDER BY updated_at DESC LIMIT 1",
	).Scan(&id)
	if err != nil {
		return nil
	}
	return s.GetSession(id)
}

func (s *ChatHistoryStore) AddMessage(sessionID string, message ChatMessage) error {
	now := time.Now()
	if message.CreatedAt.IsZero() {
		message.CreatedAt = now
	}

	_, err := db.Exec(
		"INSERT INTO chat_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)",
		sessionID, message.Role, message.Content, message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add message: %w", err)
	}

	// Update session's updated_at
	db.Exec("UPDATE chat_sessions SET updated_at = ? WHERE id = ?", now, sessionID)

	// Auto-update title from first user message
	var currentTitle string
	db.QueryRow("SELECT title FROM chat_sessions WHERE id = ?", sessionID).Scan(&currentTitle)
	if currentTitle == "New Chat" && message.Role == "user" {
		title := message.Content
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		db.Exec("UPDATE chat_sessions SET title = ? WHERE id = ?", title, sessionID)
	}

	return nil
}

func (s *ChatHistoryStore) UpdateSessionTitle(id string, title string) error {
	result, err := db.Exec(
		"UPDATE chat_sessions SET title = ?, updated_at = ? WHERE id = ?",
		title, time.Now(), id,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

func (s *ChatHistoryStore) ArchiveSession(id string) error {
	result, err := db.Exec(
		"UPDATE chat_sessions SET archived = 1, updated_at = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

func (s *ChatHistoryStore) UnarchiveSession(id string) error {
	result, err := db.Exec(
		"UPDATE chat_sessions SET archived = 0, updated_at = ? WHERE id = ?",
		time.Now(), id,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

func (s *ChatHistoryStore) DeleteSession(id string) error {
	// Delete messages first (foreign key cascade should handle this, but be explicit)
	db.Exec("DELETE FROM chat_messages WHERE session_id = ?", id)

	result, err := db.Exec("DELETE FROM chat_sessions WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

func (s *ChatHistoryStore) GetSessions(includeArchived bool) []*ChatSession {
	query := "SELECT id, title, archived, created_at, updated_at FROM chat_sessions"
	if !includeArchived {
		query += " WHERE archived = 0"
	}
	query += " ORDER BY updated_at DESC"

	rows, err := db.Query(query)
	if err != nil {
		return []*ChatSession{}
	}
	defer rows.Close()

	var sessions []*ChatSession
	for rows.Next() {
		var session ChatSession
		var archived int
		if err := rows.Scan(&session.ID, &session.Title, &archived, &session.CreatedAt, &session.UpdatedAt); err != nil {
			continue
		}
		session.Archived = archived != 0

		// Load messages for message count and last message preview
		msgRows, err := db.Query(
			"SELECT role, content, created_at FROM chat_messages WHERE session_id = ? ORDER BY id ASC", session.ID,
		)
		if err == nil {
			session.Messages = []ChatMessage{}
			for msgRows.Next() {
				var msg ChatMessage
				if err := msgRows.Scan(&msg.Role, &msg.Content, &msg.CreatedAt); err == nil {
					session.Messages = append(session.Messages, msg)
				}
			}
			msgRows.Close()
		}

		sessions = append(sessions, &session)
	}

	if sessions == nil {
		sessions = []*ChatSession{}
	}
	return sessions
}
