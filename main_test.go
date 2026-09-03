package main

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test helper: Create a temporary test environment
func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "claude-viewer-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create the .claude/projects structure
	projectsDir := filepath.Join(tmpDir, ".claude", "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// Create a test JSONL conversation file
func createTestConversation(t *testing.T, dir, sessionID string, entries []map[string]interface{}) {
	t.Helper()

	filePath := filepath.Join(dir, sessionID+".jsonl")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			t.Fatalf("Failed to encode entry: %v", err)
		}
	}
}

// Test projectDirToPath conversion
func TestProjectDirToPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"-home-sebas-robotin", "/home/sebas/robotin"},
		{"-home-user-projects-myapp", "/home/user/projects/myapp"},
		{"home-sebas", "/home/sebas"},
		{"-var-www", "/var/www"},
		{"", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := projectDirToPath(tt.input)
			if result != tt.expected {
				t.Errorf("projectDirToPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// El nombre de carpeta pisa con "-" todo lo que no es alfanumérico, así que un "-"
// del nombre puede venir de una barra, de un punto o de un guión de verdad. El
// validador tiene que aceptar las tres y rechazar cualquier otra diferencia.
func TestDirNameEncodesPath(t *testing.T) {
	tests := []struct {
		dirName string
		path    string
		want    bool
	}{
		{"-home-sebas-robotin", "/home/sebas/robotin", true},
		{"-tmp-at-import-test", "/tmp/at-import-test", true},
		{"-home-sebas-apps-invaders--workspace-t-318-bala", "/home/sebas/apps/invaders/.workspace/t-318-bala", true},
		{"C--Users-Juan-Patricio", `C:\Users\Juan-Patricio`, true},
		{"-home-sebas-robotin", "/home/sebas/otro-x", false},
		{"-home-sebas", "/home/sebas/robotin", false},
		{"-home-sebas-x", "/home/sebas", false},
		{"", "", true},
		{"-home-café-x", "/home/café/x", true},
	}

	for _, tt := range tests {
		t.Run(tt.dirName+" "+tt.path, func(t *testing.T) {
			if got := dirNameEncodesPath(tt.dirName, tt.path); got != tt.want {
				t.Errorf("dirNameEncodesPath(%q, %q) = %v, want %v", tt.dirName, tt.path, got, tt.want)
			}
		})
	}
}

// El cwd que guarda cada conversación es la ruta literal, así que gana sobre la
// heurística — sobre todo cuando la carpeta real ya no está en el disco.
func TestProjectPathForDirUsesCwd(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	dirName := "-tmp-borrado--workspace-t-320-ruta-larga"
	convDir := filepath.Join(tmpDir, ".claude", "projects", dirName)
	os.MkdirAll(convDir, 0755)

	createTestConversation(t, convDir, "conv1", []map[string]interface{}{
		{"type": "summary", "summary": "sin cwd todavía"},
		{
			"type":      "user",
			"uuid":      "u1",
			"cwd":       "/tmp/borrado/.workspace/t-320-ruta-larga",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	})

	want := "/tmp/borrado/.workspace/t-320-ruta-larga"
	if got := projectPathForDir(convDir, dirName); got != want {
		t.Errorf("projectPathForDir = %q, want %q", got, want)
	}
}

// Una conversación puede haber arrancado en otro lado y terminar copiada acá: si el
// cwd no codifica a este nombre de carpeta, no dice nada de esta carpeta.
func TestProjectPathForDirIgnoresForeignCwd(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	dirName := "-tmp-borrado-uno"
	convDir := filepath.Join(tmpDir, ".claude", "projects", dirName)
	os.MkdirAll(convDir, 0755)

	createTestConversation(t, convDir, "conv1", []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"cwd":       "/tmp/otra/cosa/completamente",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	})

	want := projectDirToPath(dirName)
	if got := projectPathForDir(convDir, dirName); got != want {
		t.Errorf("projectPathForDir = %q, want fallback %q", got, want)
	}
}

// Sin cwd en ninguna conversación queda la heurística de siempre.
func TestProjectPathForDirFallsBackWithoutCwd(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	dirName := "-tmp-sin-cwd"
	convDir := filepath.Join(tmpDir, ".claude", "projects", dirName)
	os.MkdirAll(convDir, 0755)

	createTestConversation(t, convDir, "conv1", []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	})

	want := projectDirToPath(dirName)
	if got := projectPathForDir(convDir, dirName); got != want {
		t.Errorf("projectPathForDir = %q, want fallback %q", got, want)
	}
}

// El índice tiene que quedar armado con la ruta del cwd, no con la adivinada.
func TestLoadAllConversationsUsesCwdForProjectPath(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	convDir := filepath.Join(tmpDir, ".claude", "projects", "-tmp-borrado--workspace-t-320-ruta-larga")
	os.MkdirAll(convDir, 0755)

	createTestConversation(t, convDir, "conv1", []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"cwd":       "/tmp/borrado/.workspace/t-320-ruta-larga",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	})

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Version: 1, Conversations: make(map[string]*ConversationMeta)},
	}
	if err := app.loadAllConversations(); err != nil {
		t.Fatalf("loadAllConversations failed: %v", err)
	}

	conv := app.conversations["conv1"]
	if conv == nil {
		t.Fatal("Expected conv1 in the index")
	}
	if want := "/tmp/borrado/.workspace/t-320-ruta-larga"; conv.Project != want {
		t.Errorf("conv.Project = %q, want %q", conv.Project, want)
	}
	if want := "t-320-ruta-larga"; conv.ProjectName != want {
		t.Errorf("conv.ProjectName = %q, want %q", conv.ProjectName, want)
	}
}

// La ruta se imprime con puntos de quiebre en los separadores para que no estire el
// ancho de la página, y lo que va adentro sigue escapado.
func TestBreakablePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/home/sebas/robotin", "/<wbr>home/<wbr>sebas/<wbr>robotin"},
		{`C:\Users\Juan`, `C:\<wbr>Users\<wbr>Juan`},
		{"/tmp/a-b-c", "/<wbr>tmp/<wbr>a-<wbr>b-<wbr>c"},
		{"", ""},
		{"/tmp/<script>", "/<wbr>tmp/<wbr>&lt;script&gt;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := string(breakablePath(tt.input)); got != tt.want {
				t.Errorf("breakablePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Test truncateString
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello World", 20, "Hello World"},
		{"Hello World", 5, "He..."},
		{"Hello\nWorld", 20, "Hello World"},
		{"  Spaces  ", 20, "Spaces"},
		{"", 10, ""},
		{"Short", 100, "Short"},
		{"A very long string that should be truncated", 10, "A very ..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// Test extractContentAndTools (text extraction)
func TestExtractContentAndTools(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    `"Hello World"`,
			expected: "Hello World",
		},
		{
			name:     "array with text blocks",
			input:    `[{"type": "text", "text": "First"}, {"type": "text", "text": "Second"}]`,
			expected: "First\n\nSecond",
		},
		{
			name:     "array with mixed types",
			input:    `[{"type": "text", "text": "Hello"}, {"type": "tool_use", "name": "bash"}]`,
			expected: "Hello",
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: "",
		},
		{
			name:     "invalid json",
			input:    `{invalid}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, _ := extractContentAndTools(json.RawMessage(tt.input), nil)
			if result != tt.expected {
				t.Errorf("extractContentAndTools(%s) text = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test sortConversations
func TestSortConversations(t *testing.T) {
	now := time.Now()
	convs := []*Conversation{
		{
			SessionID:    "1",
			Titles:       []string{"Zebra"},
			FirstMessage: "First A",
			FirstTime:    now.Add(-2 * time.Hour),
			LastTime:     now.Add(-1 * time.Hour),
			MessageCount: 10,
		},
		{
			SessionID:    "2",
			Titles:       []string{"Apple"},
			FirstMessage: "First B",
			FirstTime:    now.Add(-3 * time.Hour),
			LastTime:     now,
			MessageCount: 5,
		},
		{
			SessionID:    "3",
			Titles:       []string{},
			FirstMessage: "Middle",
			FirstTime:    now.Add(-1 * time.Hour),
			LastTime:     now.Add(-30 * time.Minute),
			MessageCount: 20,
		},
	}

	t.Run("sort by name asc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByName, SortAsc)

		if c[0].SessionID != "2" { // Apple
			t.Errorf("Expected Apple first, got %s", c[0].Titles)
		}
		if c[1].SessionID != "3" { // Middle (uses FirstMessage since no title)
			t.Errorf("Expected Middle second, got session %s", c[1].SessionID)
		}
	})

	t.Run("sort by name desc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByName, SortDesc)

		if c[0].SessionID != "1" { // Zebra
			t.Errorf("Expected Zebra first, got %s", c[0].Titles)
		}
	})

	t.Run("sort by first activity asc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByFirstActivity, SortAsc)

		if c[0].SessionID != "2" { // Oldest first activity
			t.Errorf("Expected session 2 first, got %s", c[0].SessionID)
		}
	})

	t.Run("sort by first activity desc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByFirstActivity, SortDesc)

		if c[0].SessionID != "3" { // Most recent first activity
			t.Errorf("Expected session 3 first, got %s", c[0].SessionID)
		}
	})

	t.Run("sort by last activity desc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByLastActivity, SortDesc)

		if c[0].SessionID != "2" { // Most recent last activity
			t.Errorf("Expected session 2 first, got %s", c[0].SessionID)
		}
	})

	t.Run("sort by last activity asc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByLastActivity, SortAsc)

		if c[0].SessionID != "1" { // Oldest last activity
			t.Errorf("Expected session 1 first, got %s", c[0].SessionID)
		}
	})

	t.Run("sort by messages desc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByMessages, SortDesc)

		if c[0].SessionID != "3" { // Most messages (20)
			t.Errorf("Expected session 3 first, got %s", c[0].SessionID)
		}
	})

	t.Run("sort by messages asc", func(t *testing.T) {
		c := make([]*Conversation, len(convs))
		copy(c, convs)
		sortConversations(c, SortByMessages, SortAsc)

		if c[0].SessionID != "2" { // Least messages (5)
			t.Errorf("Expected session 2 first, got %s", c[0].SessionID)
		}
	})
}

// Test getProjectsSorted
func TestGetProjectsSorted(t *testing.T) {
	now := time.Now()
	app := &App{
		projects: map[string]*Project{
			"/home/zebra": {
				Path:          "/home/zebra",
				Name:          "Zebra",
				FirstActivity: now.Add(-3 * time.Hour),
				LastActivity:  now.Add(-1 * time.Hour),
				TotalMessages: 100,
			},
			"/home/apple": {
				Path:          "/home/apple",
				Name:          "Apple",
				FirstActivity: now.Add(-1 * time.Hour),
				LastActivity:  now,
				TotalMessages: 50,
			},
			"/home/middle": {
				Path:          "/home/middle",
				Name:          "Middle",
				FirstActivity: now.Add(-2 * time.Hour),
				LastActivity:  now.Add(-30 * time.Minute),
				TotalMessages: 200,
			},
		},
	}

	t.Run("sort by name asc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByName, SortAsc)
		if projects[0].Name != "Apple" {
			t.Errorf("Expected Apple first, got %s", projects[0].Name)
		}
	})

	t.Run("sort by name desc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByName, SortDesc)
		if projects[0].Name != "Zebra" {
			t.Errorf("Expected Zebra first, got %s", projects[0].Name)
		}
	})

	t.Run("sort by first activity asc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByFirstActivity, SortAsc)
		if projects[0].Name != "Zebra" {
			t.Errorf("Expected Zebra first (oldest), got %s", projects[0].Name)
		}
	})

	t.Run("sort by first activity desc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByFirstActivity, SortDesc)
		if projects[0].Name != "Apple" {
			t.Errorf("Expected Apple first (most recent), got %s", projects[0].Name)
		}
	})

	t.Run("sort by last activity desc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByLastActivity, SortDesc)
		if projects[0].Name != "Apple" {
			t.Errorf("Expected Apple first (most recent), got %s", projects[0].Name)
		}
	})

	t.Run("sort by last activity asc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByLastActivity, SortAsc)
		if projects[0].Name != "Zebra" {
			t.Errorf("Expected Zebra first (oldest), got %s", projects[0].Name)
		}
	})

	t.Run("sort by messages desc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByMessages, SortDesc)
		if projects[0].Name != "Middle" {
			t.Errorf("Expected Middle first (200 msgs), got %s", projects[0].Name)
		}
	})

	t.Run("sort by messages asc", func(t *testing.T) {
		projects := app.getProjectsSorted(SortByMessages, SortAsc)
		if projects[0].Name != "Apple" {
			t.Errorf("Expected Apple first (50 msgs), got %s", projects[0].Name)
		}
	})
}

// Test template functions
func TestTemplateFunctions(t *testing.T) {
	testTime := time.Date(2025, 6, 15, 14, 30, 45, 0, time.UTC)

	t.Run("formatTime", func(t *testing.T) {
		fn := templateFuncs["formatTime"].(func(time.Time) string)
		result := fn(testTime)
		if result != "14:30:45" {
			t.Errorf("formatTime = %q, want %q", result, "14:30:45")
		}
	})

	t.Run("formatDate", func(t *testing.T) {
		fn := templateFuncs["formatDate"].(func(time.Time) string)
		result := fn(testTime)
		if result != "2025-06-15" {
			t.Errorf("formatDate = %q, want %q", result, "2025-06-15")
		}
	})

	t.Run("formatDateTime", func(t *testing.T) {
		fn := templateFuncs["formatDateTime"].(func(time.Time) string)
		result := fn(testTime)
		if result != "2025-06-15 14:30" {
			t.Errorf("formatDateTime = %q, want %q", result, "2025-06-15 14:30")
		}
	})

	t.Run("truncate", func(t *testing.T) {
		fn := templateFuncs["truncate"].(func(string, int) string)
		result := fn("Hello World", 5)
		if result != "He..." {
			t.Errorf("truncate = %q, want %q", result, "He...")
		}
	})

	t.Run("join", func(t *testing.T) {
		fn := templateFuncs["join"].(func([]string, string) string)
		result := fn([]string{"a", "b", "c"}, ", ")
		if result != "a, b, c" {
			t.Errorf("join = %q, want %q", result, "a, b, c")
		}
	})

	t.Run("nl2br", func(t *testing.T) {
		fn := templateFuncs["nl2br"].(func(string) template.HTML)
		result := fn("Hello\nWorld")
		expected := template.HTML("Hello<br>World")
		if result != expected {
			t.Errorf("nl2br = %q, want %q", result, expected)
		}
	})

	t.Run("nl2br escapes html", func(t *testing.T) {
		fn := templateFuncs["nl2br"].(func(string) template.HTML)
		result := fn("<script>alert('xss')</script>")
		if strings.Contains(string(result), "<script>") {
			t.Errorf("nl2br should escape HTML: %q", result)
		}
	})
}

// Test App with real file parsing
func TestAppParseConversationFile(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-home-test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create test conversation
	entries := []map[string]interface{}{
		{
			"type":      "summary",
			"summary":   "Test Conversation Title",
			"timestamp": "2025-01-15T10:00:00Z",
		},
		{
			"type":      "user",
			"uuid":      "user-1",
			"timestamp": "2025-01-15T10:00:00Z",
			"sessionId": "test-session",
			"message": map[string]interface{}{
				"role":    "user",
				"content": "Hello, how are you?",
			},
		},
		{
			"type":      "assistant",
			"uuid":      "assistant-1",
			"timestamp": "2025-01-15T10:00:01Z",
			"sessionId": "test-session",
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": "I'm doing well, thank you!",
				"model":   "claude-3",
			},
		},
		{
			"type":      "user",
			"uuid":      "user-2",
			"timestamp": "2025-01-15T10:00:02Z",
			"sessionId": "test-session",
			"message": map[string]interface{}{
				"role":    "user",
				"content": "Great!",
			},
		},
	}

	createTestConversation(t, projectDir, "test-session", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Version: 1, Conversations: make(map[string]*ConversationMeta)},
	}

	conv, err := app.parseConversationFile(
		filepath.Join(projectDir, "test-session.jsonl"),
		"test-session",
		"/home/test/project",
		"-home-test-project",
	)

	if err != nil {
		t.Fatalf("parseConversationFile failed: %v", err)
	}

	if conv.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", conv.SessionID, "test-session")
	}

	if len(conv.Titles) != 1 || conv.Titles[0] != "Test Conversation Title" {
		t.Errorf("Titles = %v, want [Test Conversation Title]", conv.Titles)
	}

	if conv.MessageCount != 3 {
		t.Errorf("MessageCount = %d, want 3", conv.MessageCount)
	}

	if conv.UserCount != 2 {
		t.Errorf("UserCount = %d, want 2", conv.UserCount)
	}

	if conv.AssistantCount != 1 {
		t.Errorf("AssistantCount = %d, want 1", conv.AssistantCount)
	}

	if !conv.FullyLoaded {
		t.Error("FullyLoaded should be true")
	}
}

// Test loadAllConversations
func TestAppLoadAllConversations(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create two projects
	project1Dir := filepath.Join(tmpDir, ".claude", "projects", "-home-project1")
	project2Dir := filepath.Join(tmpDir, ".claude", "projects", "-home-project2")
	os.MkdirAll(project1Dir, 0755)
	os.MkdirAll(project2Dir, 0755)

	// Add conversations to each
	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}

	createTestConversation(t, project1Dir, "conv1", entries)
	createTestConversation(t, project1Dir, "conv2", entries)
	createTestConversation(t, project2Dir, "conv3", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Version: 1, Conversations: make(map[string]*ConversationMeta)},
	}

	err := app.loadAllConversations()
	if err != nil {
		t.Fatalf("loadAllConversations failed: %v", err)
	}

	if len(app.conversations) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(app.conversations))
	}

	if len(app.projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(app.projects))
	}
}

// Test cache functionality
func TestCacheLoadSave(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create app and save cache
	app := &App{
		metadataPath: cachePath,
		cache: &MetadataCache{
			Version:     1,
			LastUpdated: time.Now(),
			Conversations: map[string]*ConversationMeta{
				"test-session": {
					SessionID:    "test-session",
					Project:      "/test/project",
					Titles:       []string{"Test Title"},
					MessageCount: 10,
				},
			},
		},
	}

	app.saveCache()

	// Verify file was created
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Load cache into new app
	app2 := &App{
		metadataPath: cachePath,
		cache:        &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app2.loadCache()

	if app2.cache.Version != 1 {
		t.Errorf("Cache version = %d, want 1", app2.cache.Version)
	}

	cached := app2.cache.Conversations["test-session"]
	if cached == nil {
		t.Fatal("Cached conversation not found")
	}

	if cached.MessageCount != 10 {
		t.Errorf("Cached MessageCount = %d, want 10", cached.MessageCount)
	}
}

// Test HTTP handlers
func TestHandlerIndex(t *testing.T) {
	app := &App{
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
	}

	app.projects["/test"] = &Project{
		Path:          "/test",
		Name:          "Test",
		LastActivity:  time.Now(),
		TotalMessages: 5,
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	app.handleIndex(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Claude Conversations") {
		t.Error("Response should contain 'Claude Conversations'")
	}
}

func TestHandlerIndexWithSort(t *testing.T) {
	app := &App{
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
	}

	for _, sort := range []string{"last", "first", "name", "messages"} {
		req := httptest.NewRequest("GET", "/?sort="+sort, nil)
		w := httptest.NewRecorder()

		app.handleIndex(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Sort=%s: Status = %d, want %d", sort, resp.StatusCode, http.StatusOK)
		}
	}
}

func TestHandlerProject(t *testing.T) {
	app := &App{
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
	}

	conv := &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		LastTime:    time.Now(),
	}

	app.conversations["conv1"] = conv
	app.projects["/test/project"] = &Project{
		Path:          "/test/project",
		Name:          "project",
		Conversations: []*Conversation{conv},
	}

	t.Run("valid project", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/project?path=/test/project", nil)
		w := httptest.NewRecorder()

		app.handleProject(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/project", nil)
		w := httptest.NewRecorder()

		app.handleProject(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/project?path=/nonexistent", nil)
		w := httptest.NewRecorder()

		app.handleProject(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestHandlerConversation(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	createTestConversation(t, projectDir, "conv1", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app.conversations["conv1"] = &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		LastTime:    time.Now(),
		FullyLoaded: false,
	}

	t.Run("valid conversation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/conversation?id=conv1", nil)
		w := httptest.NewRecorder()

		app.handleConversation(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/conversation", nil)
		w := httptest.NewRecorder()

		app.handleConversation(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/conversation?id=nonexistent", nil)
		w := httptest.NewRecorder()

		app.handleConversation(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestHandlerSearch(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello searchable content"},
		},
	}
	createTestConversation(t, projectDir, "conv1", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app.conversations["conv1"] = &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		LastTime:    time.Now(),
		FullyLoaded: false,
	}

	t.Run("empty query redirects", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search", nil)
		w := httptest.NewRecorder()

		app.handleSearch(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusFound {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusFound)
		}
	})

	t.Run("valid search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search?q=searchable", nil)
		w := httptest.NewRecorder()

		app.handleSearch(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "searchable") {
			t.Error("Response should contain search term")
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/search?q=nonexistentterm", nil)
		w := httptest.NewRecorder()

		app.handleSearch(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})
}

func TestHandlerAPIProjects(t *testing.T) {
	app := &App{
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
	}

	app.projects["/test"] = &Project{
		Path:          "/test",
		Name:          "Test",
		LastActivity:  time.Now(),
		TotalMessages: 5,
	}

	req := httptest.NewRequest("GET", "/api/projects", nil)
	w := httptest.NewRecorder()

	app.handleAPIProjects(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var projects []*Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}
}

func TestHandlerAPIConversation(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "API test"},
		},
	}
	createTestConversation(t, projectDir, "conv1", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app.conversations["conv1"] = &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		LastTime:    time.Now(),
		FullyLoaded: false,
	}

	t.Run("valid request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/conversation?id=conv1", nil)
		w := httptest.NewRecorder()

		app.handleAPIConversation(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var conv Conversation
		if err := json.NewDecoder(resp.Body).Decode(&conv); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if conv.SessionID != "conv1" {
			t.Errorf("SessionID = %q, want conv1", conv.SessionID)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/conversation", nil)
		w := httptest.NewRecorder()

		app.handleAPIConversation(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/conversation?id=nonexistent", nil)
		w := httptest.NewRecorder()

		app.handleAPIConversation(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

// Test loadFullConversation
func TestLoadFullConversation(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Message 1"},
		},
		{
			"type":      "assistant",
			"uuid":      "a1",
			"timestamp": "2025-01-15T10:00:01Z",
			"message":   map[string]interface{}{"role": "assistant", "content": "Response 1"},
		},
	}
	createTestConversation(t, projectDir, "conv1", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app.conversations["conv1"] = &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		FullyLoaded: false,
	}

	t.Run("loads conversation", func(t *testing.T) {
		conv, err := app.loadFullConversation("conv1")
		if err != nil {
			t.Fatalf("loadFullConversation failed: %v", err)
		}

		if !conv.FullyLoaded {
			t.Error("Conversation should be fully loaded")
		}

		if len(conv.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(conv.Messages))
		}
	})

	t.Run("returns cached if already loaded", func(t *testing.T) {
		app.conversations["conv1"].FullyLoaded = true
		app.conversations["conv1"].Messages = []Message{{Content: "cached"}}

		conv, err := app.loadFullConversation("conv1")
		if err != nil {
			t.Fatalf("loadFullConversation failed: %v", err)
		}

		if conv.Messages[0].Content != "cached" {
			t.Error("Should return cached conversation")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := app.loadFullConversation("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent conversation")
		}
	})
}

// Una conversación que nació DESPUÉS de que se armó el índice se abre igual.
//
// Es el caso que rompía en la vida real: el visor corre como servicio y carga el
// índice al arrancar, así que toda sesión nueva —justo la que uno quiere espiar por
// un link recién generado— contestaba "conversation not found" con el .jsonl ahí
// nomás, en el disco.
func TestLoadFullConversationAdoptsOneBornAfterTheIndex(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	// El índice ya está armado y vacío: la conversación aparece recién ahora.
	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-recien-nacida")
	os.MkdirAll(projectDir, 0755)
	createTestConversation(t, projectDir, "nacida-despues", []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hola"},
		},
		{
			"type":      "assistant",
			"uuid":      "a1",
			"timestamp": "2025-01-15T10:00:01Z",
			"message":   map[string]interface{}{"role": "assistant", "content": "Hola de vuelta"},
		},
	})

	conv, err := app.loadFullConversation("nacida-despues")
	if err != nil {
		t.Fatalf("no la encontró en el disco: %v", err)
	}
	if len(conv.Messages) != 2 {
		t.Errorf("esperaba 2 mensajes, hay %d", len(conv.Messages))
	}
	if _, ok := app.conversations["nacida-despues"]; !ok {
		t.Error("quedó fuera del índice: la próxima visita la vuelve a buscar en el disco")
	}
	// Y colgada de su proyecto: si solo entrara al mapa, se abriría por link pero no
	// figuraría en la lista de su proyecto, que se lee como si el visor la perdiera.
	proj, ok := app.projects[conv.Project]
	if !ok || len(proj.Conversations) != 1 {
		t.Errorf("no quedó colgada de su proyecto: %v", app.projects)
	}
}

// El id viaja en la query string y termina adentro de un glob. Uno con separadores de
// ruta adentro elegiría qué archivo abrir, así que se rechaza antes de mirar el disco.
func TestLoadFullConversationRejectsAnIDThatIsAPath(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	for _, id := range []string{"../../../etc/passwd", "*", "-test/conv1"} {
		if _, err := app.loadFullConversation(id); err == nil {
			t.Errorf("aceptó %q como id de conversación", id)
		}
	}
}

// Test search in titles
func TestSearchInTitles(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Regular content"},
		},
	}
	createTestConversation(t, projectDir, "conv1", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app.conversations["conv1"] = &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		Titles:      []string{"Special Unique Title"},
		LastTime:    time.Now(),
		FullyLoaded: false,
	}

	req := httptest.NewRequest("GET", "/search?q=unique", nil)
	w := httptest.NewRecorder()

	app.handleSearch(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	// Should find the conversation because "unique" is in the title
	if !strings.Contains(string(body), "conversations found") {
		t.Error("Search should find conversation by title")
	}
}

// Test empty conversation file
func TestParseEmptyConversation(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	// Create empty file
	filePath := filepath.Join(projectDir, "empty.jsonl")
	os.WriteFile(filePath, []byte{}, 0644)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	conv, err := app.parseConversationFile(filePath, "empty", "/test", "-test-project")
	if err != nil {
		t.Fatalf("parseConversationFile failed: %v", err)
	}

	if conv.MessageCount != 0 {
		t.Errorf("Expected 0 messages for empty file, got %d", conv.MessageCount)
	}
}

// Test conversation with array content blocks
func TestParseConversationWithArrayContent(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "assistant",
			"uuid":      "a1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []map[string]interface{}{
					{"type": "text", "text": "First part"},
					{"type": "tool_use", "name": "bash"},
					{"type": "text", "text": "Second part"},
				},
			},
		},
	}
	createTestConversation(t, projectDir, "array-conv", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	conv, err := app.parseConversationFile(
		filepath.Join(projectDir, "array-conv.jsonl"),
		"array-conv",
		"/test",
		"-test-project",
	)

	if err != nil {
		t.Fatalf("parseConversationFile failed: %v", err)
	}

	if conv.MessageCount != 1 {
		t.Errorf("Expected 1 message, got %d", conv.MessageCount)
	}

	if !strings.Contains(conv.Messages[0].Content, "First part") {
		t.Error("Content should contain 'First part'")
	}

	if !strings.Contains(conv.Messages[0].Content, "Second part") {
		t.Error("Content should contain 'Second part'")
	}
}

// Test search preview generation with edge cases
func TestSearchPreviewEdgeCases(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	// Create conversation with content that tests preview extraction
	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "test at the beginning"},
		},
		{
			"type":      "assistant",
			"uuid":      "a1",
			"timestamp": "2025-01-15T10:00:01Z",
			"message":   map[string]interface{}{"role": "assistant", "content": "This is a very long message with test somewhere in the middle of it to verify preview extraction works correctly"},
		},
		{
			"type":      "user",
			"uuid":      "u2",
			"timestamp": "2025-01-15T10:00:02Z",
			"message":   map[string]interface{}{"role": "user", "content": "ending with test"},
		},
		{
			"type":      "assistant",
			"uuid":      "a2",
			"timestamp": "2025-01-15T10:00:03Z",
			"message":   map[string]interface{}{"role": "assistant", "content": "test test test multiple matches"},
		},
	}
	createTestConversation(t, projectDir, "preview-test", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	app.conversations["preview-test"] = &Conversation{
		SessionID:   "preview-test",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		LastTime:    time.Now(),
		FullyLoaded: false,
	}

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()

	app.handleSearch(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	// Should have found the conversation
	if !strings.Contains(string(body), "conversations found") {
		t.Error("Search should find conversation with test content")
	}
}

// Test project sorting with empty conversations
func TestProjectSortingEdgeCases(t *testing.T) {
	app := &App{
		projects: map[string]*Project{
			"/empty": {
				Path:          "/empty",
				Name:          "Empty",
				Conversations: []*Conversation{},
				TotalMessages: 0,
			},
		},
	}

	projects := app.getProjectsSorted(SortByLastActivity, SortDesc)
	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}
}

// Test hasPrefix and contains template functions
func TestTemplateFunctionsStringOps(t *testing.T) {
	hasPrefixFn := templateFuncs["hasPrefix"].(func(string, string) bool)
	containsFn := templateFuncs["contains"].(func(string, string) bool)
	lowerFn := templateFuncs["lower"].(func(string) string)

	if !hasPrefixFn("Hello World", "Hello") {
		t.Error("hasPrefix should return true for 'Hello World', 'Hello'")
	}

	if hasPrefixFn("Hello World", "World") {
		t.Error("hasPrefix should return false for 'Hello World', 'World'")
	}

	if !containsFn("Hello World", "World") {
		t.Error("contains should return true for 'Hello World', 'World'")
	}

	if containsFn("Hello World", "Foo") {
		t.Error("contains should return false for 'Hello World', 'Foo'")
	}

	if lowerFn("HELLO") != "hello" {
		t.Errorf("lower('HELLO') = %q, want 'hello'", lowerFn("HELLO"))
	}
}

// Test cache with missing file (no error)
func TestCacheLoadMissingFile(t *testing.T) {
	app := &App{
		metadataPath: "/nonexistent/path/cache.json",
		cache:        &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	// Should not panic
	app.loadCache()

	if app.cache.Version != 0 {
		t.Errorf("Cache version should be 0 (unloaded), got %d", app.cache.Version)
	}
}

// Test loadAllConversations with non-JSONL files
func TestLoadAllConversationsSkipsNonJSONL(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	// Create a non-JSONL file
	os.WriteFile(filepath.Join(projectDir, "readme.txt"), []byte("not a conversation"), 0644)
	os.WriteFile(filepath.Join(projectDir, ".hidden"), []byte("hidden file"), 0644)

	// Create a valid JSONL file
	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	createTestConversation(t, projectDir, "valid", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Version: 1, Conversations: make(map[string]*ConversationMeta)},
	}

	err := app.loadAllConversations()
	if err != nil {
		t.Fatalf("loadAllConversations failed: %v", err)
	}

	// Should only load the valid.jsonl file
	if len(app.conversations) != 1 {
		t.Errorf("Expected 1 conversation, got %d", len(app.conversations))
	}
}

// Test loading with cached data
func TestLoadAllConversationsWithCache(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	createTestConversation(t, projectDir, "cached-conv", entries)

	// Get file info for cache
	fileInfo, _ := os.Stat(filepath.Join(projectDir, "cached-conv.jsonl"))

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache: &MetadataCache{
			Version: 1,
			Conversations: map[string]*ConversationMeta{
				"cached-conv": {
					SessionID:    "cached-conv",
					Project:      "/test/project",
					Titles:       []string{"Cached Title"},
					FirstMessage: "Cached first message",
					MessageCount: 100, // Different from actual to prove cache is used
					FileModTime:  fileInfo.ModTime(),
				},
			},
		},
	}

	err := app.loadAllConversations()
	if err != nil {
		t.Fatalf("loadAllConversations failed: %v", err)
	}

	// Should use cached data
	conv := app.conversations["cached-conv"]
	if conv.MessageCount != 100 {
		t.Errorf("Expected cached MessageCount 100, got %d", conv.MessageCount)
	}
	if len(conv.Titles) != 1 || conv.Titles[0] != "Cached Title" {
		t.Errorf("Expected cached title, got %v", conv.Titles)
	}
}

// Test duplicate titles are not added
func TestDuplicateTitlesRemoved(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{"type": "summary", "summary": "Same Title", "timestamp": "2025-01-15T10:00:00Z"},
		{"type": "summary", "summary": "Same Title", "timestamp": "2025-01-15T10:00:01Z"},
		{"type": "summary", "summary": "Different Title", "timestamp": "2025-01-15T10:00:02Z"},
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:03Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
	}
	createTestConversation(t, projectDir, "dup-titles", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		conversations: make(map[string]*Conversation),
		projects:      make(map[string]*Project),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}

	conv, err := app.parseConversationFile(
		filepath.Join(projectDir, "dup-titles.jsonl"),
		"dup-titles",
		"/test",
		"-test-project",
	)

	if err != nil {
		t.Fatalf("parseConversationFile failed: %v", err)
	}

	if len(conv.Titles) != 2 {
		t.Errorf("Expected 2 unique titles, got %d: %v", len(conv.Titles), conv.Titles)
	}
}

func TestHandlerConversationRendersTypeFilter(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	projectDir := filepath.Join(tmpDir, ".claude", "projects", "-test-project")
	os.MkdirAll(projectDir, 0755)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"uuid":      "u1",
			"timestamp": "2025-01-15T10:00:00Z",
			"message":   map[string]interface{}{"role": "user", "content": "Hello"},
		},
		{
			"type":      "assistant",
			"uuid":      "a1",
			"timestamp": "2025-01-15T10:00:01Z",
			"message":   map[string]interface{}{"role": "assistant", "content": "Hi there"},
		},
		{
			"type":      "queue-operation",
			"uuid":      "q1",
			"timestamp": "2025-01-15T10:00:02Z",
		},
	}
	createTestConversation(t, projectDir, "conv1", entries)

	app := &App{
		claudeDir:     filepath.Join(tmpDir, ".claude"),
		projects:      make(map[string]*Project),
		conversations: make(map[string]*Conversation),
		cache:         &MetadataCache{Conversations: make(map[string]*ConversationMeta)},
	}
	app.conversations["conv1"] = &Conversation{
		SessionID:   "conv1",
		Project:     "/test/project",
		ProjectName: "project",
		ProjectDir:  "-test-project",
		LastTime:    time.Now(),
		FullyLoaded: false,
	}

	req := httptest.NewRequest("GET", "/conversation?id=conv1", nil)
	w := httptest.NewRecorder()
	app.handleConversation(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	for _, want := range []string{
		`id="typeFilter"`,
		`id="typeFilterDropdown"`,
		`data-type="user"`,
		`data-type="assistant"`,
		`class="system-entry-badge" data-type="queue-operation"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("conversation page is missing %q", want)
		}
	}
}
