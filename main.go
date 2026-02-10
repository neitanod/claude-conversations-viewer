package main

import (
	"bufio"
	"embed"
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

// Message represents a single message in a conversation (user or assistant)
type Message struct {
	UUID      string    `json:"uuid"`
	Type      string    `json:"type"` // "user" or "assistant"
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	Role      string    `json:"role"`
	Model     string    `json:"model,omitempty"`
	ToolUses  []ToolUse `json:"toolUses,omitempty"`
}

// ToolUse represents a tool usage in a message
type ToolUse struct {
	Name  string `json:"name"`
	Input string `json:"input"`
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
	app := &App{
		claudeDir:     claudeDir,
		metadataPath:  filepath.Join(claudeDir, "conversations-viewer-cache.json"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Version: 1, Conversations: make(map[string]*ConversationMeta)},
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

	for scanner.Scan() {
		var entry RawEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

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

		ts, _ := time.Parse(time.RFC3339, entry.Timestamp)

		msg := Message{
			UUID:      entry.UUID,
			Type:      entry.Type,
			Timestamp: ts,
		}

		// Parse message content
		if entry.Message != nil {
			var msgContent RawMessageContent
			if err := json.Unmarshal(entry.Message, &msgContent); err == nil {
				msg.Role = msgContent.Role
				msg.Model = msgContent.Model

				// Extract text content
				msg.Content = extractTextContent(msgContent.Content)
			}
		}

		// Skip empty messages
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}

		conv.Messages = append(conv.Messages, msg)

		if entry.Type == "user" {
			conv.UserCount++
		} else {
			conv.AssistantCount++
		}

		// Track timestamps
		if conv.FirstTime.IsZero() || ts.Before(conv.FirstTime) {
			conv.FirstTime = ts
			conv.FirstMessage = truncateString(msg.Content, 100)
		}
		if ts.After(conv.LastTime) {
			conv.LastTime = ts
		}
	}

	conv.MessageCount = len(conv.Messages)
	return conv, nil
}

func extractTextContent(raw json.RawMessage) string {
	// Try as string first
	var strContent string
	if err := json.Unmarshal(raw, &strContent); err == nil {
		return strContent
	}

	// Try as array of content blocks
	var blocks []map[string]interface{}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var texts []string
		for _, block := range blocks {
			if block["type"] == "text" {
				if text, ok := block["text"].(string); ok {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "\n")
	}

	return ""
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
	if query == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	type SearchResult struct {
		Conversation *Conversation
		MatchCount   int
		Previews     []string
	}

	var results []SearchResult

	for _, conv := range a.conversations {
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

	data := struct {
		Query   string
		Results []SearchResult
		Count   int
	}{
		Query:   r.URL.Query().Get("q"),
		Results: results,
		Count:   len(results),
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

	// Routes
	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/project", app.handleProject)
	http.HandleFunc("/conversation", app.handleConversation)
	http.HandleFunc("/search", app.handleSearch)
	http.HandleFunc("/api/projects", app.handleAPIProjects)
	http.HandleFunc("/api/conversation", app.handleAPIConversation)

	addr := ":" + port
	log.Printf("Starting Claude Conversations Viewer on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
