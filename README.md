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

### Asisted by an AI agent

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

---

# Claude Conversations Viewer

Interfaz web local para explorar, buscar y leer todas las conversaciones con Claude Code, sin importar en qué proyecto se originaron.

## Problema que resuelve

Claude Code almacena las conversaciones en `~/.claude/projects/` de forma fragmentada por proyecto. Esta app unifica todas las conversaciones en una interfaz web navegable con búsqueda y ordenamiento.

## Instalación

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

### Asistida por un agente de IA

Si usás un agente con acceso a tu terminal (Claude Code, Cursor, etc.), podés
instalar pegándole este prompt:

<https://github.com/neitanod/claude-conversations-viewer/blob/main/install_prompt.md>

## Uso

```bash
# Iniciar en puerto por defecto (8042)
./conversations-viewer           # Linux/macOS
.\conversations-viewer.exe       # Windows

# Iniciar en puerto personalizado
./conversations-viewer 9000      # Linux/macOS
.\conversations-viewer.exe 9000  # Windows
```

Luego abrir http://localhost:8042 en el navegador.

## Características

- Autenticación opcional con usuario/password (sesiones de 30 días)
- Vista de todos los proyectos con conversaciones
- Navegación por proyecto y conversación individual
- Búsqueda full-text en mensajes de usuario y Agent
- Búsqueda por proyecto (en página de proyecto)
- Búsqueda local en conversación (JavaScript)
- Navegación Vim-style: j/k, gg/G, v (selección), y (yank), h/l (colapsar/expandir)
- Ordenamiento por última/primera actividad, nombre, cantidad de mensajes
- Toggle ascendente/descendente clickeando en el mismo criterio
- Modo claro y oscuro con persistencia
- Contenido colapsable para mensajes largos
- Cache de metadatos para carga rápida

## Ejemplos

### Ver todas las conversaciones

```bash
./run
# Abrir http://localhost:8042
```

### Buscar en conversaciones

1. Usar el campo de búsqueda en el header
2. Escribir el término y presionar Enter
3. Los resultados muestran previews con contexto alrededor del match

### Cambiar ordenamiento

- Click en "Última actividad" para ordenar por fecha de último mensaje
- Click de nuevo para invertir el orden (asc/desc)
- Los proyectos más recientes aparecen primero por defecto

## API JSON

```bash
# Listar proyectos
curl http://localhost:8042/api/projects

# Obtener conversación completa
curl http://localhost:8042/api/conversation?id=SESSION_ID
```

## Tests

```bash
./test
# O directamente:
go test -v -cover
```

Cobertura actual: 82.7%

## Autenticación (opcional)

Por defecto la aplicación no requiere login. Para habilitar autenticación, crear el archivo `~/.claude/conversations-viewer-config.json`:

```json
{
  "username": "mi_usuario",
  "password": "mi_password_seguro"
}
```

Si el archivo no existe o tiene credenciales vacías, la app funciona sin autenticación. Las sesiones duran 30 días y se almacenan en `~/.claude/conversations-viewer-sessions.json`.

## Configuración

- **Puerto:** Primer argumento CLI (default: 8042)
- **Cache:** `~/.claude/conversations-viewer-cache.json` (se crea automáticamente)
- **Config:** `~/.claude/conversations-viewer-config.json` (credenciales)
- **Sesiones:** `~/.claude/conversations-viewer-sessions.json` (tokens de sesión)
- **Tema:** Se guarda en localStorage del navegador

## Estructura de archivos

```
claude-conversations-viewer/
├── main.go           # Código fuente principal
├── main_test.go      # Tests unitarios
├── go.mod            # Módulo Go
├── run               # Script para ejecutar
├── test              # Script para tests
├── build             # Script para compilar
├── README.md         # Esta documentación
├── ai/
│   └── specs/        # Especificaciones técnicas
├── templates/        # Templates HTML
└── static/
    └── style.css     # Estilos CSS
```
