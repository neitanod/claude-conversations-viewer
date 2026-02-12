package main

import (
	"bufio"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Config holds application configuration
type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Session represents a user session
type Session struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// SessionStore manages sessions
type SessionStore struct {
	sessions map[string]*Session
	filePath string
	mu       sync.RWMutex
}

func NewSessionStore(filePath string) *SessionStore {
	ss := &SessionStore{
		sessions: make(map[string]*Session),
		filePath: filePath,
	}
	ss.load()
	return ss
}

func (ss *SessionStore) load() {
	data, err := os.ReadFile(ss.filePath)
	if err != nil {
		return
	}
	var sessions map[string]*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return
	}
	// Filter expired sessions
	now := time.Now()
	for token, session := range sessions {
		if session.ExpiresAt.After(now) {
			ss.sessions[token] = session
		}
	}
}

func (ss *SessionStore) save() {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	data, err := json.MarshalIndent(ss.sessions, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(ss.filePath, data, 0600)
}

func (ss *SessionStore) Create() string {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	token := generateToken()
	ss.sessions[token] = &Session{
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}
	go ss.save()
	return token
}

func (ss *SessionStore) Validate(token string) bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	session, exists := ss.sessions[token]
	if !exists {
		return false
	}
	return session.ExpiresAt.After(time.Now())
}

func (ss *SessionStore) Delete(token string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, token)
	go ss.save()
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func loadConfig(configPath string) (*Config, error) {
	// Default config
	config := &Config{
		Username: "user",
		Password: "conversations#",
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config file
			defaultData, _ := json.MarshalIndent(config, "", "  ")
			os.WriteFile(configPath, defaultData, 0600)
			return config, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}

// ContentBlock represents either text or a tool invocation (in order)
type ContentBlock struct {
	Type     string `json:"type"` // "text", "thinking", or "tool"
	Text     string `json:"text,omitempty"`
	ToolID   string `json:"toolId,omitempty"`
	ToolName string `json:"toolName,omitempty"`
	Input    string `json:"input,omitempty"`
	Result   string `json:"result,omitempty"`
}

// Message represents a single message in a conversation (user or assistant)
type Message struct {
	UUID          string         `json:"uuid"`
	Type          string         `json:"type"` // "user" or "assistant"
	Timestamp     time.Time      `json:"timestamp"`
	Content       string         `json:"content"`       // Legacy: all text concatenated
	Role          string         `json:"role"`
	Model         string         `json:"model,omitempty"`
	ToolBlocks    []ToolBlock    `json:"toolBlocks,omitempty"`    // Legacy
	ContentBlocks []ContentBlock `json:"contentBlocks,omitempty"` // New: ordered blocks
}

// ToolBlock represents a tool invocation with its result (legacy, kept for compatibility)
type ToolBlock struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Input  string `json:"input"`
	Result string `json:"result"`
}

// Conversation represents a full conversation with all messages
type Conversation struct {
	SessionID     string    `json:"sessionId"`
	Project       string    `json:"project"`
	ProjectName   string    `json:"projectName"`
	ProjectDir    string    `json:"projectDir"` // Directory name in .claude/projects/
	Titles        []string  `json:"titles"`     // Summary/titles found
	FirstMessage  string    `json:"firstMessage"`
	FirstTime     time.Time `json:"firstTime"`
	LastTime      time.Time `json:"lastTime"`
	Messages      []Message `json:"messages"`
	MessageCount  int       `json:"messageCount"`
	UserCount     int       `json:"userCount"`
	AssistantCount int      `json:"assistantCount"`
	FullyLoaded   bool      `json:"fullyLoaded"`
}

// Project represents a project with its conversations
type Project struct {
	Path          string          `json:"path"`
	Name          string          `json:"name"`
	Conversations []*Conversation `json:"conversations"`
	TotalMessages int             `json:"totalMessages"`
	LastActivity  time.Time       `json:"lastActivity"`
	FirstActivity time.Time       `json:"firstActivity"`
}

// Metadata for caching conversation info
type ConversationMeta struct {
	SessionID    string    `json:"sessionId"`
	Project      string    `json:"project"`
	Titles       []string  `json:"titles"`
	FirstMessage string    `json:"firstMessage"`
	FirstTime    time.Time `json:"firstTime"`
	LastTime     time.Time `json:"lastTime"`
	MessageCount int       `json:"messageCount"`
	UserCount    int       `json:"userCount"`
	AssistantCount int     `json:"assistantCount"`
	FileModTime  time.Time `json:"fileModTime"`
}

type MetadataCache struct {
	Version       int                          `json:"version"`
	LastUpdated   time.Time                    `json:"lastUpdated"`
	Conversations map[string]*ConversationMeta `json:"conversations"`
}

// App holds application state
type App struct {
	claudeDir     string
	metadataPath  string
	conversations map[string]*Conversation
	projects      map[string]*Project
	cache         *MetadataCache
	mu            sync.RWMutex
	config        *Config
	sessions      *SessionStore
}

// RawMessage types for parsing JSONL
type RawEntry struct {
	Type      string          `json:"type"`
	UUID      string          `json:"uuid"`
	Timestamp string          `json:"timestamp"`
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message"`
	Summary   string          `json:"summary"`
}

type RawMessageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Model   string          `json:"model"`
}

func NewApp() (*App, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeDir := filepath.Join(homeDir, ".claude")

	// Load config
	configPath := filepath.Join(claudeDir, "conversations-viewer-config.json")
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize session store
	sessionsPath := filepath.Join(claudeDir, "conversations-viewer-sessions.json")
	sessions := NewSessionStore(sessionsPath)

	app := &App{
		claudeDir:     claudeDir,
		metadataPath:  filepath.Join(claudeDir, "conversations-viewer-cache.json"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Version: 1, Conversations: make(map[string]*ConversationMeta)},
		config:        config,
		sessions:      sessions,
	}

	// Load existing cache
	app.loadCache()

	// Load conversations (using cache when possible)
	if err := app.loadAllConversations(); err != nil {
		return nil, fmt.Errorf("failed to load conversations: %w", err)
	}

	// Save updated cache in background
	go app.saveCache()

	return app, nil
}

func (a *App) loadCache() {
	data, err := os.ReadFile(a.metadataPath)
	if err != nil {
		return // No cache yet
	}
	json.Unmarshal(data, a.cache)
}

func (a *App) saveCache() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.cache.LastUpdated = time.Now()
	data, err := json.MarshalIndent(a.cache, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal cache: %v", err)
		return
	}
	if err := os.WriteFile(a.metadataPath, data, 0644); err != nil {
		log.Printf("Failed to write cache: %v", err)
	}
}

func (a *App) loadAllConversations() error {
	projectsDir := filepath.Join(a.claudeDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return fmt.Errorf("failed to read projects dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := entry.Name()
		projectPath := projectDirToPath(projectDir)

		convDir := filepath.Join(projectsDir, projectDir)
		convFiles, err := os.ReadDir(convDir)
		if err != nil {
			continue
		}

		for _, cf := range convFiles {
			if !strings.HasSuffix(cf.Name(), ".jsonl") {
				continue
			}

			// Skip warmup agent conversations (internal Claude Code sidechain processes)
			if strings.HasPrefix(cf.Name(), "agent-") {
				continue
			}

			sessionID := strings.TrimSuffix(cf.Name(), ".jsonl")
			filePath := filepath.Join(convDir, cf.Name())

			info, err := cf.Info()
			if err != nil {
				continue
			}

			// Check cache
			cached := a.cache.Conversations[sessionID]
			if cached != nil && cached.FileModTime.Equal(info.ModTime()) {
				// Use cached metadata
				conv := &Conversation{
					SessionID:      sessionID,
					Project:        projectPath,
					ProjectName:    filepath.Base(projectPath),
					ProjectDir:     projectDir,
					Titles:         cached.Titles,
					FirstMessage:   cached.FirstMessage,
					FirstTime:      cached.FirstTime,
					LastTime:       cached.LastTime,
					MessageCount:   cached.MessageCount,
					UserCount:      cached.UserCount,
					AssistantCount: cached.AssistantCount,
					FullyLoaded:    false,
				}
				a.conversations[sessionID] = conv
			} else {
				// Parse the file
				conv, err := a.parseConversationFile(filePath, sessionID, projectPath, projectDir)
				if err != nil {
					continue
				}
				if conv.MessageCount == 0 {
					continue
				}

				a.conversations[sessionID] = conv

				// Update cache
				a.cache.Conversations[sessionID] = &ConversationMeta{
					SessionID:      sessionID,
					Project:        projectPath,
					Titles:         conv.Titles,
					FirstMessage:   conv.FirstMessage,
					FirstTime:      conv.FirstTime,
					LastTime:       conv.LastTime,
					MessageCount:   conv.MessageCount,
					UserCount:      conv.UserCount,
					AssistantCount: conv.AssistantCount,
					FileModTime:    info.ModTime(),
				}
			}
		}
	}

	// Group by project
	for _, conv := range a.conversations {
		proj, exists := a.projects[conv.Project]
		if !exists {
			proj = &Project{
				Path:          conv.Project,
				Name:          filepath.Base(conv.Project),
				Conversations: []*Conversation{},
			}
			a.projects[conv.Project] = proj
		}
		proj.Conversations = append(proj.Conversations, conv)
		proj.TotalMessages += conv.MessageCount
		if proj.LastActivity.Before(conv.LastTime) {
			proj.LastActivity = conv.LastTime
		}
		if proj.FirstActivity.IsZero() || conv.FirstTime.Before(proj.FirstActivity) {
			proj.FirstActivity = conv.FirstTime
		}
	}

	return nil
}

func projectDirToPath(dirName string) string {
	// Convert "-home-sebas-robotin" to "/home/sebas/robotin"
	path := strings.ReplaceAll(dirName, "-", "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func (a *App) parseConversationFile(filePath, sessionID, projectPath, projectDir string) (*Conversation, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	conv := &Conversation{
		SessionID:   sessionID,
		Project:     projectPath,
		ProjectName: filepath.Base(projectPath),
		ProjectDir:  projectDir,
		Messages:    []Message{},
		Titles:      []string{},
		FullyLoaded: true,
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB buffer for long lines

	titlesMap := make(map[string]bool) // Avoid duplicates

	// Map to collect tool_use IDs to their results
	toolResults := make(map[string]string) // tool_use_id -> result content

	// First pass: collect all tool results
	var entries []RawEntry
	for scanner.Scan() {
		var entry RawEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)

		// Collect tool results
		if entry.Type == "user" && entry.Message != nil {
			var msgContent RawMessageContent
			if err := json.Unmarshal(entry.Message, &msgContent); err == nil {
				collectToolResults(msgContent.Content, toolResults)
			}
		}
	}

	// Second pass: build messages with tool blocks
	var lastAssistantMsg *Message // Track last assistant message for merging

	for _, entry := range entries {
		// Collect summaries as titles
		if entry.Type == "summary" && entry.Summary != "" {
			if !titlesMap[entry.Summary] {
				titlesMap[entry.Summary] = true
				conv.Titles = append(conv.Titles, entry.Summary)
			}
			continue
		}

		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}

		// Skip tool_result entries (they are merged into assistant messages)
		if entry.Type == "user" && entry.Message != nil {
			var msgContent RawMessageContent
			if err := json.Unmarshal(entry.Message, &msgContent); err == nil {
				if isToolResultContent(msgContent.Content) {
					continue
				}
			}
		}

		ts, _ := time.Parse(time.RFC3339, entry.Timestamp)

		// Parse message content
		var content string
		var toolBlocks []ToolBlock
		var contentBlocks []ContentBlock
		var role, model string

		if entry.Message != nil {
			var msgContent RawMessageContent
			if err := json.Unmarshal(entry.Message, &msgContent); err == nil {
				role = msgContent.Role
				model = msgContent.Model
				content, toolBlocks, contentBlocks = extractContentAndTools(msgContent.Content, toolResults)
			}
		}

		// Skip empty messages (no text and no tools)
		if strings.TrimSpace(content) == "" && len(toolBlocks) == 0 {
			continue
		}

		// For assistant messages, try to merge consecutive ones
		if entry.Type == "assistant" {
			if lastAssistantMsg != nil {
				// Merge into the last assistant message
				if content != "" {
					if lastAssistantMsg.Content != "" {
						lastAssistantMsg.Content += "\n\n" + content
					} else {
						lastAssistantMsg.Content = content
					}
				}
				lastAssistantMsg.ToolBlocks = append(lastAssistantMsg.ToolBlocks, toolBlocks...)
				lastAssistantMsg.ContentBlocks = append(lastAssistantMsg.ContentBlocks, contentBlocks...)
				lastAssistantMsg.Timestamp = ts // Update to latest timestamp
				continue
			}

			// Start a new assistant message
			msg := Message{
				UUID:          entry.UUID,
				Type:          entry.Type,
				Timestamp:     ts,
				Content:       content,
				Role:          role,
				Model:         model,
				ToolBlocks:    toolBlocks,
				ContentBlocks: contentBlocks,
			}
			conv.Messages = append(conv.Messages, msg)
			lastAssistantMsg = &conv.Messages[len(conv.Messages)-1]
			conv.AssistantCount++
		} else {
			// User message - reset the assistant merge tracking
			lastAssistantMsg = nil

			msg := Message{
				UUID:          entry.UUID,
				Type:          entry.Type,
				Timestamp:     ts,
				Content:       content,
				Role:          role,
				Model:         model,
				ToolBlocks:    toolBlocks,
				ContentBlocks: contentBlocks,
			}
			conv.Messages = append(conv.Messages, msg)
			conv.UserCount++
		}

		// Track timestamps
		if conv.FirstTime.IsZero() || ts.Before(conv.FirstTime) {
			conv.FirstTime = ts
			lastMsg := &conv.Messages[len(conv.Messages)-1]
			if lastMsg.Content != "" {
				conv.FirstMessage = truncateString(lastMsg.Content, 100)
			} else if len(lastMsg.ToolBlocks) > 0 {
				conv.FirstMessage = fmt.Sprintf("[%s]", lastMsg.ToolBlocks[0].Name)
			}
		}
		if ts.After(conv.LastTime) {
			conv.LastTime = ts
		}
	}

	conv.MessageCount = len(conv.Messages)
	return conv, nil
}

// collectToolResults extracts tool_result content and maps them by tool_use_id
func collectToolResults(raw json.RawMessage, results map[string]string) {
	var blocks []map[string]interface{}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return
	}
	for _, block := range blocks {
		if block["type"] == "tool_result" {
			toolUseID, _ := block["tool_use_id"].(string)
			content, _ := block["content"].(string)
			if toolUseID != "" {
				results[toolUseID] = content
			}
		}
	}
}

// extractContentAndTools separates text content from tool_use blocks
// Returns: concatenated text (legacy), tool blocks (legacy), and ordered content blocks (new)
func extractContentAndTools(raw json.RawMessage, toolResults map[string]string) (string, []ToolBlock, []ContentBlock) {
	// Try as string first
	var strContent string
	if err := json.Unmarshal(raw, &strContent); err == nil {
		if strContent != "" {
			return strContent, nil, []ContentBlock{{Type: "text", Text: strContent}}
		}
		return strContent, nil, nil
	}

	// Try as array of content blocks
	var blocks []map[string]interface{}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", nil, nil
	}

	var textParts []string
	var toolBlocks []ToolBlock
	var contentBlocks []ContentBlock

	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		switch blockType {
		case "text":
			if text, ok := block["text"].(string); ok && text != "" {
				textParts = append(textParts, text)
				contentBlocks = append(contentBlocks, ContentBlock{Type: "text", Text: text})
			}
		case "thinking":
			if thinking, ok := block["thinking"].(string); ok && thinking != "" {
				textParts = append(textParts, fmt.Sprintf("💭  %s", thinking))
				contentBlocks = append(contentBlocks, ContentBlock{Type: "thinking", Text: thinking})
			}
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			if name != "" {
				inputStr := formatToolInput(block["input"])
				result := toolResults[id]
				toolBlocks = append(toolBlocks, ToolBlock{
					ID:     id,
					Name:   name,
					Input:  inputStr,
					Result: result,
				})
				contentBlocks = append(contentBlocks, ContentBlock{
					Type:     "tool",
					ToolID:   id,
					ToolName: name,
					Input:    inputStr,
					Result:   result,
				})
			}
		}
	}

	return strings.Join(textParts, "\n\n"), toolBlocks, contentBlocks
}

// formatToolInput formats the input of a tool_use for display
func formatToolInput(input interface{}) string {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return ""
	}

	// For Bash, show the command
	if cmd, ok := inputMap["command"].(string); ok {
		return cmd
	}
	// For Read/Write/Edit, show file path
	if filePath, ok := inputMap["file_path"].(string); ok {
		if oldStr, ok := inputMap["old_string"].(string); ok {
			// Edit tool
			return fmt.Sprintf("%s\nold: %s", filePath, truncateString(oldStr, 50))
		}
		return filePath
	}
	// For Grep/Glob, show pattern
	if pattern, ok := inputMap["pattern"].(string); ok {
		return pattern
	}
	// For WebSearch/WebFetch
	if query, ok := inputMap["query"].(string); ok {
		return query
	}
	if url, ok := inputMap["url"].(string); ok {
		return url
	}

	// Fallback: JSON
	if data, err := json.Marshal(inputMap); err == nil {
		return truncateString(string(data), 200)
	}
	return ""
}

// isToolResultContent checks if the content is a tool_result array
func isToolResultContent(raw json.RawMessage) bool {
	var blocks []map[string]interface{}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, block := range blocks {
			if block["type"] == "tool_result" {
				return true
			}
		}
	}
	return false
}

func truncateString(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Sorting functions
type SortOrder string
type SortDir string

const (
	SortByLastActivity  SortOrder = "last"
	SortByFirstActivity SortOrder = "first"
	SortByName          SortOrder = "name"
	SortByMessages      SortOrder = "messages"

	SortAsc  SortDir = "asc"
	SortDesc SortDir = "desc"
)

func (a *App) getProjectsSorted(order SortOrder, dir SortDir) []*Project {
	projects := make([]*Project, 0, len(a.projects))
	for _, p := range a.projects {
		projects = append(projects, p)
	}

	asc := dir == SortAsc

	switch order {
	case SortByName:
		sort.Slice(projects, func(i, j int) bool {
			less := strings.ToLower(projects[i].Name) < strings.ToLower(projects[j].Name)
			if asc {
				return less
			}
			return !less
		})
	case SortByFirstActivity:
		sort.Slice(projects, func(i, j int) bool {
			less := projects[i].FirstActivity.Before(projects[j].FirstActivity)
			if asc {
				return less
			}
			return !less
		})
	case SortByMessages:
		sort.Slice(projects, func(i, j int) bool {
			less := projects[i].TotalMessages < projects[j].TotalMessages
			if asc {
				return less
			}
			return !less
		})
	default: // SortByLastActivity
		sort.Slice(projects, func(i, j int) bool {
			less := projects[i].LastActivity.Before(projects[j].LastActivity)
			if asc {
				return less
			}
			return !less
		})
	}

	return projects
}

func sortConversations(convs []*Conversation, order SortOrder, dir SortDir) {
	asc := dir == SortAsc

	switch order {
	case SortByName:
		sort.Slice(convs, func(i, j int) bool {
			ti := ""
			if len(convs[i].Titles) > 0 {
				ti = convs[i].Titles[0]
			} else {
				ti = convs[i].FirstMessage
			}
			tj := ""
			if len(convs[j].Titles) > 0 {
				tj = convs[j].Titles[0]
			} else {
				tj = convs[j].FirstMessage
			}
			less := strings.ToLower(ti) < strings.ToLower(tj)
			if asc {
				return less
			}
			return !less
		})
	case SortByFirstActivity:
		sort.Slice(convs, func(i, j int) bool {
			less := convs[i].FirstTime.Before(convs[j].FirstTime)
			if asc {
				return less
			}
			return !less
		})
	case SortByMessages:
		sort.Slice(convs, func(i, j int) bool {
			less := convs[i].MessageCount < convs[j].MessageCount
			if asc {
				return less
			}
			return !less
		})
	default: // SortByLastActivity
		sort.Slice(convs, func(i, j int) bool {
			less := convs[i].LastTime.Before(convs[j].LastTime)
			if asc {
				return less
			}
			return !less
		})
	}
}

// Load full messages for a conversation (lazy loading)
func (a *App) loadFullConversation(sessionID string) (*Conversation, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	conv, exists := a.conversations[sessionID]
	if !exists {
		return nil, fmt.Errorf("conversation not found")
	}

	if conv.FullyLoaded && len(conv.Messages) > 0 {
		return conv, nil
	}

	// Load from file
	filePath := filepath.Join(a.claudeDir, "projects", conv.ProjectDir, sessionID+".jsonl")
	fullConv, err := a.parseConversationFile(filePath, sessionID, conv.Project, conv.ProjectDir)
	if err != nil {
		return nil, err
	}

	// Update the conversation in place
	conv.Messages = fullConv.Messages
	conv.FullyLoaded = true

	return conv, nil
}

// Template functions
var templateFuncs = template.FuncMap{
	"formatTime": func(t time.Time) string {
		return t.Format("15:04:05")
	},
	"formatDate": func(t time.Time) string {
		return t.Format("2006-01-02")
	},
	"formatDateTime": func(t time.Time) string {
		return t.Format("2006-01-02 15:04")
	},
	"truncate": func(s string, n int) string {
		return truncateString(s, n)
	},
	"join": func(strs []string, sep string) string {
		return strings.Join(strs, sep)
	},
	"hasPrefix": strings.HasPrefix,
	"contains":  strings.Contains,
	"lower":     strings.ToLower,
	"nl2br": func(s string) template.HTML {
		escaped := template.HTMLEscapeString(s)
		return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>"))
	},
}

// Handlers

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	sortOrder := SortOrder(r.URL.Query().Get("sort"))
	if sortOrder == "" {
		sortOrder = SortByLastActivity
	}
	sortDir := SortDir(r.URL.Query().Get("dir"))
	if sortDir == "" {
		sortDir = SortDesc // Default: most recent/most messages first
	}

	tmpl, err := template.New("index.html").Funcs(templateFuncs).ParseFS(templatesFS, "templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Projects           []*Project
		TotalProjects      int
		TotalConversations int
		SortOrder          SortOrder
		SortDir            SortDir
	}{
		Projects:           a.getProjectsSorted(sortOrder, sortDir),
		TotalProjects:      len(a.projects),
		TotalConversations: len(a.conversations),
		SortOrder:          sortOrder,
		SortDir:            sortDir,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func (a *App) handleProject(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("path")
	if projectPath == "" {
		http.Error(w, "Project path required", http.StatusBadRequest)
		return
	}

	sortOrder := SortOrder(r.URL.Query().Get("sort"))
	if sortOrder == "" {
		sortOrder = SortByLastActivity
	}
	sortDir := SortDir(r.URL.Query().Get("dir"))
	if sortDir == "" {
		sortDir = SortDesc
	}

	project, exists := a.projects[projectPath]
	if !exists {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Make a copy and sort
	convs := make([]*Conversation, len(project.Conversations))
	copy(convs, project.Conversations)
	sortConversations(convs, sortOrder, sortDir)

	tmpl, err := template.New("project.html").Funcs(templateFuncs).ParseFS(templatesFS, "templates/project.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		*Project
		SortedConversations []*Conversation
		SortOrder           SortOrder
		SortDir             SortDir
	}{
		Project:             project,
		SortedConversations: convs,
		SortOrder:           sortOrder,
		SortDir:             sortDir,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func (a *App) handleConversation(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	conv, err := a.loadFullConversation(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Sort messages by timestamp
	messages := make([]Message, len(conv.Messages))
	copy(messages, conv.Messages)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	tmpl, err := template.New("conversation.html").Funcs(templateFuncs).ParseFS(templatesFS, "templates/conversation.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Conversation *Conversation
		Messages     []Message
	}{
		Conversation: conv,
		Messages:     messages,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("q"))
	projectFilter := r.URL.Query().Get("project")
	if query == "" {
		if projectFilter != "" {
			http.Redirect(w, r, "/project?path="+projectFilter, http.StatusFound)
		} else {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		return
	}

	type SearchResult struct {
		Conversation *Conversation
		MatchCount   int
		Previews     []string
	}

	var results []SearchResult

	for _, conv := range a.conversations {
		// Filter by project if specified
		if projectFilter != "" && conv.Project != projectFilter {
			continue
		}
		// Load full conversation for search
		fullConv, err := a.loadFullConversation(conv.SessionID)
		if err != nil {
			continue
		}

		matchCount := 0
		var previews []string

		for _, msg := range fullConv.Messages {
			content := strings.ToLower(msg.Content)
			if strings.Contains(content, query) {
				matchCount++
				// Extract preview around match
				idx := strings.Index(content, query)
				start := idx - 50
				if start < 0 {
					start = 0
				}
				end := idx + len(query) + 50
				if end > len(msg.Content) {
					end = len(msg.Content)
				}
				preview := msg.Content[start:end]
				if start > 0 {
					preview = "..." + preview
				}
				if end < len(msg.Content) {
					preview = preview + "..."
				}
				previews = append(previews, preview)
				if len(previews) >= 3 {
					break // Limit previews
				}
			}
		}

		// Also search in titles
		for _, title := range conv.Titles {
			if strings.Contains(strings.ToLower(title), query) {
				matchCount++
			}
		}

		if matchCount > 0 {
			results = append(results, SearchResult{
				Conversation: conv,
				MatchCount:   matchCount,
				Previews:     previews,
			})
		}
	}

	// Sort by match count, then by last activity
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		return results[i].Conversation.LastTime.After(results[j].Conversation.LastTime)
	})

	tmpl, err := template.New("search.html").Funcs(templateFuncs).ParseFS(templatesFS, "templates/search.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get project name for display
	var projectName string
	if projectFilter != "" {
		if proj, ok := a.projects[projectFilter]; ok {
			projectName = proj.Name
		}
	}

	data := struct {
		Query       string
		Results     []SearchResult
		Count       int
		Project     string
		ProjectName string
	}{
		Query:       r.URL.Query().Get("q"),
		Results:     results,
		Count:       len(results),
		Project:     projectFilter,
		ProjectName: projectName,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func (a *App) handleAPIProjects(w http.ResponseWriter, r *http.Request) {
	sortOrder := SortOrder(r.URL.Query().Get("sort"))
	sortDir := SortDir(r.URL.Query().Get("dir"))
	if sortDir == "" {
		sortDir = SortDesc
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.getProjectsSorted(sortOrder, sortDir))
}

func (a *App) handleAPIConversation(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, `{"error": "Session ID required"}`, http.StatusBadRequest)
		return
	}

	conv, err := a.loadFullConversation(sessionID)
	if err != nil {
		http.Error(w, `{"error": "Conversation not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (a *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// Clear current data and reload
	a.mu.Lock()
	a.conversations = make(map[string]*Conversation)
	a.projects = make(map[string]*Project)
	// Don't clear cache - let loadAllConversations check file mod times
	a.mu.Unlock()

	if err := a.loadAllConversations(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reload: %v", err), http.StatusInternalServerError)
		return
	}

	// Save updated cache
	go a.saveCache()

	// Redirect back to referrer or home
	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusFound)
}

// Authentication handlers

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show login form
		tmpl, err := template.New("login.html").Funcs(templateFuncs).ParseFS(templatesFS, "templates/login.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Error string
		}{}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
		return
	}

	// POST - process login
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == a.config.Username && password == a.config.Password {
		// Create session
		token := a.sessions.Create()
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			MaxAge:   30 * 24 * 60 * 60, // 30 days
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Invalid credentials
	tmpl, _ := template.New("login.html").Funcs(templateFuncs).ParseFS(templatesFS, "templates/login.html")
	data := struct {
		Error string
	}{
		Error: "Usuario o contraseña incorrectos",
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	tmpl.Execute(w, data)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		a.sessions.Delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (a *App) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return a.sessions.Validate(cookie.Value)
}

func (a *App) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		handler(w, r)
	}
}

func main() {
	port := "8042"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	log.Println("Loading conversations...")
	start := time.Now()

	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	log.Printf("Loaded %d conversations from %d projects in %v", len(app.conversations), len(app.projects), time.Since(start))

	// Static files
	staticContent, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	// Auth routes (no authentication required)
	http.HandleFunc("/login", app.handleLogin)
	http.HandleFunc("/logout", app.handleLogout)

	// Protected routes
	http.HandleFunc("/", app.requireAuth(app.handleIndex))
	http.HandleFunc("/project", app.requireAuth(app.handleProject))
	http.HandleFunc("/conversation", app.requireAuth(app.handleConversation))
	http.HandleFunc("/search", app.requireAuth(app.handleSearch))
	http.HandleFunc("/refresh", app.requireAuth(app.handleRefresh))
	http.HandleFunc("/api/projects", app.requireAuth(app.handleAPIProjects))
	http.HandleFunc("/api/conversation", app.requireAuth(app.handleAPIConversation))

	log.Printf("Config file: %s", filepath.Join(app.claudeDir, "conversations-viewer-config.json"))

	addr := ":" + port
	log.Printf("Starting Claude Conversations Viewer on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
