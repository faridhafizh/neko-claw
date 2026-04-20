package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileEntry struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	IsDir        bool      `json:"isDir"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
}

// Security: Get Base Directory (defaults to user home, or can be configured)
func getBaseDir() string {
	baseDir := dbGetConfig("fs_base_dir")
	if baseDir == "" {
		baseDir, _ = os.UserHomeDir()
		if baseDir == "" {
			baseDir, _ = os.Getwd()
		}
	}
	// Always return absolute path
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return "."
	}
	return abs
}

// Security: Resolve and validate path to prevent path traversal
func resolveSafePath(reqPath string) (string, error) {
	baseDir := getBaseDir()
	
	// Default to base dir if empty
	if reqPath == "" || reqPath == "/" {
		return baseDir, nil
	}

	// Clean the requested path
	cleanReq := filepath.Clean(reqPath)
	
	// Join with base if it's not absolute or if it attempts traversal
	var fullPath string
	if filepath.IsAbs(cleanReq) {
		fullPath = cleanReq
	} else {
		fullPath = filepath.Join(baseDir, cleanReq)
	}

	// Double check absolute resolution
	finalAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	// Ensure it is still within baseDir (or is exactly baseDir)
	if !strings.HasPrefix(finalAbs, baseDir) {
		return "", fmt.Errorf("access denied: path outside base directory")
	}

	return finalAbs, nil
}

// Handler: Limit to 1MB reads to prevent memory bloat in UI
const maxFileSize = 1 * 1024 * 1024

func handleFilesystem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List directory
		reqPath := r.URL.Query().Get("path")
		safePath, err := resolveSafePath(reqPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		info, err := os.Stat(safePath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "Directory not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		if !info.IsDir() {
			http.Error(w, "Path is not a directory", http.StatusBadRequest)
			return
		}

		entries, err := os.ReadDir(safePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result := make([]FileEntry, 0)
		for _, e := range entries {
			i, err := e.Info()
			if err != nil {
				continue
			}
			result = append(result, FileEntry{
				Name:         e.Name(),
				Path:         filepath.Join(reqPath, e.Name()), // Relative to requested path
				IsDir:        e.IsDir(),
				Size:         i.Size(),
				LastModified: i.ModTime(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"baseDir": getBaseDir(),
			"current": reqPath,
			"files":   result,
		})

	case "DELETE":
		reqPath := r.URL.Query().Get("path")
		if reqPath == "" {
			http.Error(w, "Path required", http.StatusBadRequest)
			return
		}
		
		safePath, err := resolveSafePath(reqPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		if err := os.RemoveAll(safePath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFileRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		http.Error(w, "Path required", http.StatusBadRequest)
		return
	}

	safePath, err := resolveSafePath(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	info, err := os.Stat(safePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if info.IsDir() {
		http.Error(w, "Cannot read directory as file", http.StatusBadRequest)
		return
	}

	if info.Size() > maxFileSize {
		http.Error(w, "File too large to open in browser editor (>1MB)", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"path":    reqPath,
		"content": string(content),
	})
}

func handleFileWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	safePath, err := resolveSafePath(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Check if trying to edit a directory
	if info, err := os.Stat(safePath); err == nil && info.IsDir() {
		http.Error(w, "Cannot overwrite a directory with a file", http.StatusBadRequest)
		return
	}

	err = os.WriteFile(safePath, []byte(req.Content), 0644)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func handleFileMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	safePath, err := resolveSafePath(req.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	err = os.MkdirAll(safePath, 0755)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}
