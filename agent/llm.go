package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"sessionId,omitempty"`
}

type ChatResponse struct {
	Reply          string          `json:"reply"`
	SessionID      string          `json:"sessionId"`
	HasPendingCmd  bool            `json:"hasPendingCmd"`
	PendingCommand *PendingCommand `json:"pendingCommand,omitempty"`
	ActiveSoul     string          `json:"activeSoul"`
	SoulEmoji      string          `json:"soulEmoji"`
	MemoryCount    int             `json:"memoryCount"`
}

// powershellTool is the OpenAI function tool definition for proposing PowerShell commands.
// Defined once at package level to avoid duplication between handleChat and handleApproveCommand.
var powershellTool = openai.Tool{
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
}

var playwrightTool = openai.Tool{
	Type: openai.ToolTypeFunction,
	Function: &openai.FunctionDefinition{
		Name:        "web_search_and_read",
		Description: "Open a web browser, navigate to a URL, and extract the text content. Use this for web search (e.g. duckduckgo html search) and scraping. Provide a fully qualified URL like 'https://html.duckduckgo.com/html/?q=my+query'.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "The full URL to navigate to."
				}
			},
			"required": ["url"]
		}`),
	},
}

var availableTools = []openai.Tool{powershellTool, playwrightTool}

func getOpenAIClient() *openai.Client {
	s := getSettings()

	config := openai.DefaultConfig(s.ApiKey)
	if s.ApiUrl != "" {
		apiUrl := s.ApiUrl
		if before, ok := strings.CutSuffix(apiUrl, "/chat/completions"); ok {
			apiUrl = before
		}
		config.BaseURL = apiUrl
	}
	return openai.NewClientWithConfig(config)
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

	// Get or create session
	var session *ChatSession
	if req.SessionID != "" {
		session = chatHistoryStore.GetSession(req.SessionID)
	}
	if session == nil {
		session = chatHistoryStore.CreateSession("New Chat")
	}

	// Search for relevant memories based on user message
	relevantMemories := memoryStore.SearchMemories(req.Message, 5)

	// Build memory context string
	memoryContext := ""
	if len(relevantMemories) > 0 {
		memoryContext = "\n\nRelevant memories from past interactions:\n"
		for _, mem := range relevantMemories {
			memoryContext += fmt.Sprintf("- [%s] %s\n", mem.Category, mem.Content)
			// Update last used timestamp
			memoryStore.UpdateLastUsed(mem.ID)
		}
	}

	// Get system info context
	sysInfo := getSystemInfo()
	sysContext := fmt.Sprintf("\n\n[System Context]\nOS: %s\nUser: %s\nCWD: %s\nCPU: %d%%\nRAM: %d/%d MB (%d%%)", 
		sysInfo.OS, sysInfo.Username, sysInfo.CurrentDir, sysInfo.CPUUsage, sysInfo.RAMUsed, sysInfo.RAMTotal, sysInfo.RAMUsage)

	// Build system prompt with soul, memory, and system context
	activeSoul := soulStore.GetActiveSoul()
	systemPrompt := activeSoul.SystemPrompt + memoryContext + sysContext

	// Build conversation from session messages
	var messages []openai.ChatCompletionMessage
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	for _, msg := range session.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Message,
	})

	s := getSettings()

	if s.ApiKey == "" {
		http.Error(w, "API Key is missing. Silakan isi konfigurasi API Key terlebih dahulu di menu ⚙️ Settings.", http.StatusBadRequest)
		return
	}

	client := getOpenAIClient()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    s.Model,
			Messages: messages,
			Tools:    availableTools,
		},
	)

	if err != nil {
		log.Printf("Chat completion error: %v", err)

		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Rate limit") {
			http.Error(w, "Rate limit reached. Please wait a moment before sending another message.", http.StatusTooManyRequests)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	choice := resp.Choices[0]

	// Save messages to session
	chatHistoryStore.AddMessage(session.ID, ChatMessage{Role: "user", Content: req.Message})
	chatHistoryStore.AddMessage(session.ID, ChatMessage{Role: "assistant", Content: choice.Message.Content})

	// Get memory stats
	memStats := memoryStore.GetMemoryStats()
	memoryCount := memStats["total"].(int)

	res := ChatResponse{
		Reply:       choice.Message.Content,
		SessionID:   session.ID,
		ActiveSoul:  activeSoul.Name,
		SoulEmoji:   activeSoul.Emoji,
		MemoryCount: memoryCount,
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
					SessionID:   session.ID,
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
			} else if toolCall.Function.Name == "web_search_and_read" {
				var args map[string]string
				json.Unmarshal([]byte(toolCall.Function.Arguments), &args)

				url := args["url"]
				if res.Reply == "" {
					res.Reply = fmt.Sprintf("I am searching the web at %s...", url)
				}

				// Execute inline and don't need approval for search
				// Actually wait, handleChat is non-streaming and doesn't handle multiple iterations easily.
				// For now, let's just do a single loop if it's a web search.
				content, err := searchWebAndRead(url)
				if err != nil {
					content = fmt.Sprintf("Error searching web: %v", err)
				}

				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    content,
					ToolCallID: toolCall.ID,
				})

				// Call API again
				resp2, err2 := client.CreateChatCompletion(
					ctx,
					openai.ChatCompletionRequest{
						Model:    s.Model,
						Messages: messages,
						Tools:    availableTools,
					},
				)

				if err2 == nil && len(resp2.Choices) > 0 {
					choice = resp2.Choices[0]
					res.Reply = choice.Message.Content
					chatHistoryStore.AddMessage(session.ID, ChatMessage{Role: "assistant", Content: res.Reply})
				}

				// Break after processing tool
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

	// Save command execution result to session
	if cmd.SessionID != "" {
		chatHistoryStore.AddMessage(cmd.SessionID, ChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("Command executed: %s\nOutput:\n%s", cmd.Command, resultStr),
		})
	}

	// Build messages from session for follow-up
	var messages []openai.ChatCompletionMessage
	if cmd.SessionID != "" {
		session := chatHistoryStore.GetSession(cmd.SessionID)
		if session != nil {
			activeSoul := soulStore.GetActiveSoul()
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: activeSoul.SystemPrompt,
			})
			for _, msg := range session.Messages {
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	if len(messages) == 0 {
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"output": resultStr,
			"reply":  "Execution finished.",
		})
		return
	}

	// Prompt the AI again with the result
	client := getOpenAIClient()
	s := getSettings()

	resp, chatErr := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    s.Model,
			Messages: messages,
			Tools:    availableTools,
		},
	)

	reply := "Execution finished. AI didn't respond."
	if chatErr != nil {
		reply = fmt.Sprintf("Execution finished, but AI encountered an error: %v", chatErr)
	} else if len(resp.Choices) > 0 {
		reply = resp.Choices[0].Message.Content
		if cmd.SessionID != "" {
			chatHistoryStore.AddMessage(cmd.SessionID, ChatMessage{
				Role:    "assistant",
				Content: reply,
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"output":    resultStr,
		"reply":     reply,
		"sessionId": cmd.SessionID,
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
	cmd, exists := pendingCommands[req.ID]
	delete(pendingCommands, req.ID)
	pendingCmdsMutex.Unlock()

	if exists && cmd.SessionID != "" {
		chatHistoryStore.AddMessage(cmd.SessionID, ChatMessage{
			Role:    "system",
			Content: "User rejected the execution of this command.",
		})
	}

	sessionID := ""
	if cmd != nil {
		sessionID = cmd.SessionID
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":    "rejected",
		"sessionId": sessionID,
	})
}

// ── Streaming Chat Handler (SSE) ────────────────────────────────────────────

func handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get or create session
	var session *ChatSession
	if req.SessionID != "" {
		session = chatHistoryStore.GetSession(req.SessionID)
	}
	if session == nil {
		session = chatHistoryStore.CreateSession("New Chat")
	}

	// Search for relevant memories
	relevantMemories := memoryStore.SearchMemories(req.Message, 5)
	memoryContext := ""
	if len(relevantMemories) > 0 {
		memoryContext = "\n\nRelevant memories from past interactions:\n"
		for _, mem := range relevantMemories {
			memoryContext += fmt.Sprintf("- [%s] %s\n", mem.Category, mem.Content)
			memoryStore.UpdateLastUsed(mem.ID)
		}
	}

	// Get system info context
	sysInfo := getSystemInfo()
	sysContext := fmt.Sprintf("\n\n[System Context]\nOS: %s\nUser: %s\nCWD: %s\nCPU: %d%%\nRAM: %d/%d MB (%d%%)", 
		sysInfo.OS, sysInfo.Username, sysInfo.CurrentDir, sysInfo.CPUUsage, sysInfo.RAMUsed, sysInfo.RAMTotal, sysInfo.RAMUsage)

	activeSoul := soulStore.GetActiveSoul()
	systemPrompt := activeSoul.SystemPrompt + memoryContext + sysContext

	var messages []openai.ChatCompletionMessage
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	for _, msg := range session.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Message,
	})

	s := getSettings()
	if s.ApiKey == "" {
		http.Error(w, "API Key is missing.", http.StatusBadRequest)
		return
	}

	client := getOpenAIClient()
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	stream, err := client.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model:    s.Model,
			Messages: messages,
			Tools:    availableTools,
			Stream:   true,
		},
	)
	if err != nil {
		log.Printf("Stream error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Save user message
	chatHistoryStore.AddMessage(session.ID, ChatMessage{Role: "user", Content: req.Message})

	var fullContent strings.Builder
	var toolCallID, toolCallName, toolCallArgs string
	hasToolCall := false

	for {
		response, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			// Send error event
			errData, _ := json.Marshal(map[string]string{"type": "error", "content": err.Error()})
			fmt.Fprintf(w, "data: %s\n\n", errData)
			flusher.Flush()
			break
		}

		delta := response.Choices[0].Delta

		// Check for tool calls
		if len(delta.ToolCalls) > 0 {
			hasToolCall = true
			for _, tc := range delta.ToolCalls {
				if tc.ID != "" {
					toolCallID = tc.ID
				}
				if tc.Function.Name != "" {
					toolCallName = tc.Function.Name
				}
				toolCallArgs += tc.Function.Arguments
			}
			continue
		}

		// Regular content token
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			tokenData, _ := json.Marshal(map[string]string{
				"type":    "token",
				"content": delta.Content,
			})
			fmt.Fprintf(w, "data: %s\n\n", tokenData)
			flusher.Flush()
		}
	}

	// Save assistant message
	if fullContent.Len() > 0 {
		chatHistoryStore.AddMessage(session.ID, ChatMessage{Role: "assistant", Content: fullContent.String()})
	}

	// Handle tool call if present
	if hasToolCall {
		if toolCallID == "" {
			toolCallID = generateID()
		}

		if toolCallName == "run_powershell_command" {
			var args map[string]string
			json.Unmarshal([]byte(toolCallArgs), &args)

			pendingCmd := &PendingCommand{
				ID:          toolCallID,
				SessionID:   session.ID,
				Command:     args["command"],
				Description: args["description"],
			}

			pendingCmdsMutex.Lock()
			pendingCommands[toolCallID] = pendingCmd
			pendingCmdsMutex.Unlock()

			toolData, _ := json.Marshal(map[string]interface{}{
				"type":    "tool_call",
				"id":      toolCallID,
				"command": args["command"],
				"description": args["description"],
			})
			fmt.Fprintf(w, "data: %s\n\n", toolData)
			flusher.Flush()
		} else if toolCallName == "web_search_and_read" {
			var args map[string]string
			json.Unmarshal([]byte(toolCallArgs), &args)
			url := args["url"]

			// Notify UI that we are searching
			statusData, _ := json.Marshal(map[string]string{
				"type":    "token",
				"content": fmt.Sprintf("\n*Searching web: %s*\n", url),
			})
			fmt.Fprintf(w, "data: %s\n\n", statusData)
			flusher.Flush()

			content, err := searchWebAndRead(url)
			if err != nil {
				content = fmt.Sprintf("Error searching web: %v", err)
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleAssistant,
				ToolCalls:  []openai.ToolCall{{
					ID: toolCallID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name: toolCallName,
						Arguments: toolCallArgs,
					},
				}},
			})

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: toolCallID,
			})

			// Call stream again with tool result
			stream2, err2 := client.CreateChatCompletionStream(
				ctx,
				openai.ChatCompletionRequest{
					Model:    s.Model,
					Messages: messages,
					Tools:    availableTools,
					Stream:   true,
				},
			)
			if err2 == nil {
				var finalContent strings.Builder
				for {
					response2, err := stream2.Recv()
					if err != nil {
						break
					}
					delta2 := response2.Choices[0].Delta
					if delta2.Content != "" {
						finalContent.WriteString(delta2.Content)
						tokenData, _ := json.Marshal(map[string]string{
							"type":    "token",
							"content": delta2.Content,
						})
						fmt.Fprintf(w, "data: %s\n\n", tokenData)
						flusher.Flush()
					}
				}
				stream2.Close()
				if finalContent.Len() > 0 {
					chatHistoryStore.AddMessage(session.ID, ChatMessage{Role: "assistant", Content: finalContent.String()})
				}
			}
		}
	}

	// Send done event
	memStats := memoryStore.GetMemoryStats()
	memoryCount := memStats["total"].(int)

	doneData, _ := json.Marshal(map[string]interface{}{
		"type":        "done",
		"sessionId":   session.ID,
		"activeSoul":  activeSoul.Name,
		"soulEmoji":   activeSoul.Emoji,
		"memoryCount": memoryCount,
	})
	fmt.Fprintf(w, "data: %s\n\n", doneData)
	flusher.Flush()
}
