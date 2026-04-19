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
)

type Settings struct {
	ApiKey string `json:"apiKey"`
	Model  string `json:"model"`
	ApiUrl string `json:"apiUrl"` // e.g., https://dashscope.aliyuncs.com/compatible-mode/v1
}

var (
	currentSettings  Settings
	settingsMutex    sync.RWMutex
	settingsFilePath = filepath.Join("data", "settings.json")
	pendingCommands  = make(map[string]*PendingCommand)
	pendingCmdsMutex sync.RWMutex
)

type PendingCommand struct {
	ID          string   `json:"id"`
	Command     string   `json:"command"`
	Arguments   []string `json:"arguments"`
	Description string   `json:"description"`
}

func main() {
	loadSettings()

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
	mux.HandleFunc("/api/command/approve", handleApproveCommand)
	mux.HandleFunc("/api/command/reject", handleRejectCommand)

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
			log.Fatalf("Failed to run npm run build: %v", err)
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
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loadSettings() {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	// Ensure data directory exists
	dataDir := filepath.Dir(settingsFilePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Warning: Could not create data directory: %v", err)
	}

	data, err := os.ReadFile(settingsFilePath)
	if err != nil {
		// File doesn't exist or can't be read - use defaults
		if os.IsNotExist(err) {
			log.Println("No settings file found, using defaults")
		} else {
			log.Printf("Warning: Could not read settings file: %v", err)
		}
		// Apply defaults
		if currentSettings.Model == "" {
			currentSettings.Model = "glm-4.7-flash"
		}
		if currentSettings.ApiUrl == "" {
			currentSettings.ApiUrl = "https://open.bigmodel.cn/api/paas/v4/"
		}
		// Save defaults to file (we already hold the lock)
		defaultData, marshalErr := json.MarshalIndent(currentSettings, "", "  ")
		if marshalErr == nil {
			os.WriteFile(settingsFilePath, defaultData, 0600)
		}
		return
	}

	if err := json.Unmarshal(data, &currentSettings); err != nil {
		log.Printf("Warning: Could not parse settings file: %v", err)
	}

	if currentSettings.Model == "" {
		currentSettings.Model = "glm-4.7-flash"
	}
	if currentSettings.ApiUrl == "" {
		currentSettings.ApiUrl = "https://open.bigmodel.cn/api/paas/v4/"
	}
}

func saveSettings() error {
	settingsMutex.RLock()
	data, err := json.MarshalIndent(currentSettings, "", "  ")
	settingsMutex.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFilePath, data, 0600)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		settingsMutex.RLock()
		json.NewEncoder(w).Encode(currentSettings)
		settingsMutex.RUnlock()
		return
	}

	if r.Method == "POST" {
		var newSettings Settings
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		settingsMutex.Lock()
		currentSettings = newSettings
		settingsMutex.Unlock()

		saveSettings()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
