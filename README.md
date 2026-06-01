**English** · [Español](README.es.md)

# Claude Conversations Viewer

Local web interface to explore, search and read all Claude Code conversations, regardless of which project they originated from.

## Problem it solves

Claude Code stores conversations in `~/.claude/projects/` fragmented by project. This app unifies all conversations in a browsable web interface with search and sorting.

## Installation

### Linux / macOS

```bash
git clone https://github.com/neitanod/claude-conversations-viewer.git
cd claude-conversations-viewer
./build
./run
```

### Windows (PowerShell)

```powershell
git clone https://github.com/neitanod/claude-conversations-viewer.git
cd claude-conversations-viewer
.\build.ps1
.\run.ps1
```

### Assisted by an AI agent

If you use an agent with terminal access (Claude Code, Cursor, etc.), you can
install by pasting this prompt:

<https://github.com/neitanod/claude-conversations-viewer/blob/main/install_prompt.md>

## Usage

```bash
# Start on default port (8042)
./conversations-viewer           # Linux/macOS
.\conversations-viewer.exe       # Windows

# Start on custom port
./conversations-viewer 9000      # Linux/macOS
.\conversations-viewer.exe 9000  # Windows
```

Then open http://localhost:8042 in your browser.

## Features

- Optional authentication with username/password (30-day sessions)
- View all projects with conversations
- Navigate by project and individual conversation
- Full-text search in user and Agent messages
- Project-scoped search (on project page)
- Local conversation search (JavaScript)
- Vim-style navigation: j/k, gg/G, v (selection), y (yank), h/l (collapse/expand)
- Sorting by last/first activity, name, message count
- Toggle ascending/descending by clicking the same criterion
- Light and dark mode with persistence
- Collapsible content for long messages
- Metadata cache for fast loading

## Examples

### View all conversations

```bash
./run
# Open http://localhost:8042
```

### Search in conversations

1. Use the search field in the header
2. Type the term and press Enter
3. Results show previews with context around the match

### Change sorting

- Click "Last activity" to sort by last message date
- Click again to invert the order (asc/desc)
- Most recent projects appear first by default

## JSON API

```bash
# List projects
curl http://localhost:8042/api/projects

# Get full conversation
curl http://localhost:8042/api/conversation?id=SESSION_ID
```

## Tests

```bash
./test
# Or directly:
go test -v -cover
```

Current coverage: 82.7%

## Authentication (optional)

By default the application does not require login. To enable authentication, create the file `~/.claude/conversations-viewer-config.json`:

```json
{
  "username": "my_user",
  "password": "my_secure_password"
}
```

If the file does not exist or has empty credentials, the app works without authentication. Sessions last 30 days and are stored in `~/.claude/conversations-viewer-sessions.json`.

## Configuration

- **Port:** First CLI argument (default: 8042)
- **Cache:** `~/.claude/conversations-viewer-cache.json` (created automatically)
- **Config:** `~/.claude/conversations-viewer-config.json` (credentials)
- **Sessions:** `~/.claude/conversations-viewer-sessions.json` (session tokens)
- **Theme:** Saved in browser localStorage

## File structure

```
claude-conversations-viewer/
├── main.go           # Main source code
├── main_test.go      # Unit tests
├── go.mod            # Go module
├── run               # Run script
├── test              # Test script
├── build             # Build script
├── README.md         # This documentation
├── ai/
│   └── specs/        # Technical specifications
├── templates/        # HTML templates
└── static/
    └── style.css     # CSS styles
```
