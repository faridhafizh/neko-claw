package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Settings struct {
	ApiKey string `json:"apiKey"`
	Model  string `json:"model"`
	ApiUrl string `json:"apiUrl"` // e.g., https://dashscope.aliyuncs.com/compatible-mode/v1
}

var (
	settingsMutex    sync.RWMutex
	pendingCommands  = make(map[string]*PendingCommand)
	pendingCmdsMutex sync.RWMutex
)

type PendingCommand struct {
	ID          string   `json:"id"`
	SessionID   string   `json:"sessionId"`
	Command     string   `json:"command"`
	Arguments   []string `json:"arguments"`
	Description string   `json:"description"`
}

func main() {
	// Initialize database first
	if err := initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize stores (they now use the database)
	if err := initMemoryStore(); err != nil {
		log.Printf("Warning: Could not initialize memory store: %v", err)
	}
	if err := initSoulStore(); err != nil {
		log.Printf("Warning: Could not initialize soul store: %v", err)
	}
	if err := initChatHistoryStore(); err != nil {
		log.Printf("Warning: Could not initialize chat history store: %v", err)
	}

	// Load settings defaults if not set
	initSettingsDefaults()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Make data directory if not exists
	os.MkdirAll("data", 0755)

	mux := http.NewServeMux()

	// API Handlers
	mux.HandleFunc("/api/settings", handleSettings)
	mux.HandleFunc("/api/chat", handleChat)
	mux.HandleFunc("/api/chat/stream", handleChatStream)
	mux.HandleFunc("/api/command/approve", handleApproveCommand)
	mux.HandleFunc("/api/command/reject", handleRejectCommand)

	// Memory handlers
	mux.HandleFunc("/api/memory", handleMemory)
	mux.HandleFunc("/api/memory/search", handleMemorySearch)
	mux.HandleFunc("/api/memory/stats", handleMemoryStats)

	// Soul handlers
	mux.HandleFunc("/api/souls", handleSouls)
	mux.HandleFunc("/api/souls/active", handleActiveSoul)

	// Additional System endpoint
	mux.HandleFunc("/api/system/info", handleSystemInfo)

	// Filesystem endpoints
	mux.HandleFunc("/api/files", handleFilesystem)
	mux.HandleFunc("/api/files/read", handleFileRead)
	mux.HandleFunc("/api/files/write", handleFileWrite)
	mux.HandleFunc("/api/files/mkdir", handleFileMkdir)

	// Chat history handlers
	mux.HandleFunc("/api/chats", handleChatSessions)
	mux.HandleFunc("/api/chats/{id}", handleChatSession)

	// Static file serving of the UI
	uiPath := filepath.Join("..", "ui", "out")

	// Auto-build UI if not exists
	if _, err := os.Stat(uiPath); os.IsNotExist(err) {
		fmt.Println("UI 'out' directory not found. Building Next.js UI...")
		uiDir := filepath.Join("..", "ui")

		fmt.Println("Running 'npm install'...")
		installCmd := exec.Command("npm", "install")
		installCmd.Dir = uiDir
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			log.Fatalf("Failed to run npm install: %v", err)
		}

		fmt.Println("Running 'npm run build'...")
		buildCmd := exec.Command("npm", "run", "build")
		buildCmd.Dir = uiDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			log.Printf("Warning: Failed to run npm run build: %v", err)
			log.Println("Please build the UI manually or check for errors.")
		}
		fmt.Println("UI built successfully!")
	}

	fs := http.FileServer(http.Dir(uiPath))
	mux.Handle("/", fs)

	// Add CORS middleware
	handler := corsMiddleware(mux)

	fmt.Printf("Agent server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ── Settings (backed by SQLite) ────────────────────────────────────────────

func initSettingsDefaults() {
	if dbGetSetting("model") == "" {
		dbSetSetting("model", "glm-4.7-flash")
	}
	if dbGetSetting("apiUrl") == "" {
		dbSetSetting("apiUrl", "https://open.bigmodel.cn/api/paas/v4/")
	}
}

func getSettings() Settings {
	return Settings{
		ApiKey: dbGetSetting("apiKey"),
		Model:  dbGetSetting("model"),
		ApiUrl: dbGetSetting("apiUrl"),
	}
}

func setSettings(s Settings) {
	dbSetSetting("apiKey", s.ApiKey)
	dbSetSetting("model", s.Model)
	dbSetSetting("apiUrl", s.ApiUrl)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s := getSettings()
		json.NewEncoder(w).Encode(s)
		return
	}

	if r.Method == "POST" {
		var newSettings Settings
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		setSettings(newSettings)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Memory API Handlers
func handleMemory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		category := r.URL.Query().Get("category")
		limit := 0
		memories := memoryStore.GetMemories(category, limit)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"memories": memories,
			"count":    len(memories),
		})

	case "POST":
		var req struct {
			Content  string   `json:"content"`
			Category string   `json:"category"`
			Priority int      `json:"priority"`
			Tags     []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := memoryStore.AddMemory(req.Content, req.Category, req.Priority, req.Tags); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		if err := memoryStore.DeleteMemory(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	limit := 10

	memories := memoryStore.SearchMemories(query, limit)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"memories": memories,
		"count":    len(memories),
	})
}

func handleMemoryStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := memoryStore.GetMemoryStats()
	json.NewEncoder(w).Encode(stats)
}

// Soul API Handlers
func handleSouls(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	souls := soulStore.GetAllSouls()
	activeSoul := soulStore.GetActiveSoul()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"souls":         souls,
		"activeSoul":    soulStore.GetActiveSoulID(),
		"activeProfile": activeSoul,
	})
}

func handleActiveSoul(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		activeSoul := soulStore.GetActiveSoul()
		json.NewEncoder(w).Encode(activeSoul)
		return
	}

	if r.Method == "POST" {
		var req struct {
			SoulID string `json:"soulId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := soulStore.SetActiveSoul(req.SoulID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Chat History API Handlers
func handleChatSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		includeArchived := r.URL.Query().Get("archived") == "true"
		sessions := chatHistoryStore.GetSessions(includeArchived)

		// Return summary (without full messages for list view)
		type SessionSummary struct {
			ID           string    `json:"id"`
			Title        string    `json:"title"`
			CreatedAt    time.Time `json:"createdAt"`
			UpdatedAt    time.Time `json:"updatedAt"`
			Archived     bool      `json:"archived"`
			MessageCount int       `json:"messageCount"`
			LastMessage  string    `json:"lastMessage"`
		}

		summaries := make([]SessionSummary, len(sessions))
		for i, s := range sessions {
			summaries[i] = SessionSummary{
				ID:           s.ID,
				Title:        s.Title,
				CreatedAt:    s.CreatedAt,
				UpdatedAt:    s.UpdatedAt,
				Archived:     s.Archived,
				MessageCount: len(s.Messages),
				LastMessage:  s.LastMessagePreview(),
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": summaries,
		})

	case "POST":
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			req.Title = "New Chat"
		}

		session := chatHistoryStore.CreateSession(req.Title)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(session)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleChatSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	switch r.Method {
	case "GET":
		session := chatHistoryStore.GetSession(id)
		if session == nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(session)

	case "PATCH":
		var req struct {
			Action string `json:"action"` // "archive", "unarchive", "delete", "rename"
			Title  string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var err error
		switch req.Action {
		case "archive":
			err = chatHistoryStore.ArchiveSession(id)
		case "unarchive":
			err = chatHistoryStore.UnarchiveSession(id)
		case "delete":
			err = chatHistoryStore.DeleteSession(id)
		case "rename":
			err = chatHistoryStore.UpdateSessionTitle(id, req.Title)
		default:
			http.Error(w, "Invalid action", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// System API Handler
func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info := getSystemInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
