package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

var (
	cachedClient  *openai.Client
	cachedApiKey  string
	cachedApiUrl  string
	clientMutex   sync.RWMutex
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply          string          `json:"reply"`
	HasPendingCmd  bool            `json:"hasPendingCmd"`
	PendingCommand *PendingCommand `json:"pendingCommand,omitempty"`
}

var conversationHistory []openai.ChatCompletionMessage

var availableTools = []openai.Tool{
	{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "run_powershell_command",
			Description: "Propose a powershell command to execute on the user's computer. The user must approve it first.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {
						"type": "string",
						"description": "The exact powershell command to run."
					},
					"description": {
						"type": "string",
						"description": "A short summary explaining what this command does safely."
					}
				},
				"required": ["command", "description"]
			}`),
		},
	},
}

func getOpenAIClient() *openai.Client {
	settingsMutex.RLock()
	apiKey := currentSettings.ApiKey
	apiUrl := currentSettings.ApiUrl
	settingsMutex.RUnlock()

	clientMutex.RLock()
	if cachedClient != nil && cachedApiKey == apiKey && cachedApiUrl == apiUrl {
		client := cachedClient
		clientMutex.RUnlock()
		return client
	}
	clientMutex.RUnlock()

	clientMutex.Lock()
	defer clientMutex.Unlock()

	// Double check pattern
	if cachedClient != nil && cachedApiKey == apiKey && cachedApiUrl == apiUrl {
		return cachedClient
	}

	config := openai.DefaultConfig(apiKey)
	if apiUrl != "" {
		if before, ok := strings.CutSuffix(apiUrl, "/chat/completions"); ok {
			config.BaseURL = before
		} else {
			config.BaseURL = apiUrl
		}
	}

	cachedClient = openai.NewClientWithConfig(config)
	cachedApiKey = apiKey
	cachedApiUrl = apiUrl

	return cachedClient
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(conversationHistory) == 0 {
		conversationHistory = append(conversationHistory, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are an AI assistant controlling a Windows computer. 
You can help the user by suggesting powershell commands.
When you want to run a command, use the tool "run_powershell_command".
The user will review your command and explicitly approve or reject it.
Wait for the tool result before proceeding.`,
		})
	}

	conversationHistory = append(conversationHistory, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Message,
	})

	client := getOpenAIClient()
	settingsMutex.RLock()
	model := currentSettings.Model
	settingsMutex.RUnlock()

	if currentSettings.ApiKey == "" {
		http.Error(w, "API Key is missing. Silakan isi konfigurasi API Key terlebih dahulu di menu ⚙️ Settings.", http.StatusBadRequest)
		return
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: conversationHistory,
			Tools:    availableTools,
		},
	)

	if err != nil {
		log.Printf("Chat completion error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	choice := resp.Choices[0]
	conversationHistory = append(conversationHistory, choice.Message)

	res := ChatResponse{
		Reply: choice.Message.Content,
	}

	if len(choice.Message.ToolCalls) > 0 {
		// Process all tool calls, but we only expect one run_powershell_command per request
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Function.Name == "run_powershell_command" {
				var args map[string]string
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

				cmdID := toolCall.ID
				pendingCmd := &PendingCommand{
					ID:          cmdID,
					Command:     args["command"],
					Description: args["description"],
				}

				pendingCmdsMutex.Lock()
				pendingCommands[cmdID] = pendingCmd
				pendingCmdsMutex.Unlock()

				res.HasPendingCmd = true
				res.PendingCommand = pendingCmd
				if res.Reply == "" {
					res.Reply = fmt.Sprintf("I need to run a command: %s", args["description"])
				}
				// Break after processing the first tool call (UI handles one at a time)
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func handleApproveCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pendingCmdsMutex.RLock()
	cmd, exists := pendingCommands[req.ID]
	pendingCmdsMutex.RUnlock()

	if !exists {
		http.Error(w, "Command not found or already executed", http.StatusNotFound)
		return
	}

	// Remove from pending
	pendingCmdsMutex.Lock()
	delete(pendingCommands, req.ID)
	pendingCmdsMutex.Unlock()

	// Execute it
	execCmd := exec.Command("pwsh", "-Command", cmd.Command)
	output, err := execCmd.CombinedOutput()
	resultStr := string(output)
	
	// Clean ANSI escape codes (PowerShell colors)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	resultStr = ansiRegex.ReplaceAllString(resultStr, "")

	if err != nil {
		resultStr += fmt.Sprintf("\nError: %v", err)
	}

	// Send outcome back to conversation
	conversationHistory = append(conversationHistory, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    resultStr,
		Name:       "run_powershell_command",
		ToolCallID: req.ID,
	})

	// Prompt the AI again with the result
	client := getOpenAIClient()
	settingsMutex.RLock()
	model := currentSettings.Model
	settingsMutex.RUnlock()

	resp, chatErr := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: conversationHistory,
			Tools:    availableTools,
		},
	)

	reply := "Execution finished. AI didn't respond."
	if chatErr != nil {
		reply = fmt.Sprintf("Execution finished, but AI encountered an error: %v", chatErr)
	} else if len(resp.Choices) > 0 {
		reply = resp.Choices[0].Message.Content
		conversationHistory = append(conversationHistory, resp.Choices[0].Message)
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"output": resultStr,
		"reply":  reply,
	})
}

func handleRejectCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pendingCmdsMutex.Lock()
	delete(pendingCommands, req.ID)
	pendingCmdsMutex.Unlock()

	conversationHistory = append(conversationHistory, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    "User rejected the execution of this command.",
		Name:       "run_powershell_command",
		ToolCallID: req.ID,
	})

	json.NewEncoder(w).Encode(map[string]string{
		"status": "rejected",
	})
}
