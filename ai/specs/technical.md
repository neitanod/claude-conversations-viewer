# Technical Decisions

## Stack

- **Language:** Go 1.21+
- **Dependencies:** Standard library only (net/http, html/template, encoding/json)
- **Frontend:** HTML + vanilla CSS, no JavaScript frameworks
- **Templates:** Go templates embedded with `//go:embed`
- **Styles:** CSS with variables for theming

## Architecture

```
main.go              # Main code (entire app in one file)
├── Types
│   ├── Message      # Individual message (user/assistant)
│   ├── Conversation # Conversation with metadata
│   ├── Project      # Project with its conversations
│   └── App          # Application state
│
├── Parsing
│   ├── parseConversationFile()  # Reads and parses .jsonl
│   ├── extractTextContent()      # Extracts text from content blocks
│   └── loadAllConversations()    # Initial load with cache
│
├── Cache
│   ├── loadCache()   # Load cache from disk
│   └── saveCache()   # Save cache in background
│
├── Sorting
│   ├── getProjectsSorted()    # Sort projects
│   └── sortConversations()    # Sort conversations
│
└── HTTP Handlers
    ├── handleIndex        # GET /
    ├── handleProject      # GET /project?path=...
    ├── handleConversation # GET /conversation?id=...
    ├── handleSearch       # GET /search?q=...
    ├── handleAPIProjects  # GET /api/projects
    └── handleAPIConversation # GET /api/conversation?id=...

templates/           # Embedded HTML templates
├── index.html       # Project listing
├── project.html     # Project conversations
├── conversation.html # Conversation view
└── search.html      # Search results

static/
└── style.css        # Styles with CSS variables for themes
```

## Claude Code data structure

Claude Code stores conversations in:
```
~/.claude/
├── projects/
│   ├── -home-sebas-robotin/           # Project name (path with dashes)
│   │   ├── abc123.jsonl               # One conversation per file
│   │   └── def456.jsonl
│   └── -home-sebas-doc-prj-gobot/
│       └── xyz789.jsonl
└── conversations-viewer-cache.json    # Metadata cache (created by this app)
```

Each `.jsonl` file contains JSON lines with:
- `type: "user"` - User message
- `type: "assistant"` - Assistant response
- `type: "summary"` - Conversation titles/summaries

## Configuration

- **Port:** 8042 by default, configurable as first CLI argument
- **Cache:** `~/.claude/conversations-viewer-cache.json`
- **Theme:** Saved in browser localStorage

## Design decisions

### Lazy loading of messages
Full messages are only loaded when viewing a conversation. The initial list only loads metadata (title, timestamps, counts).

### Metadata cache
Metadata is cached per file with its `modTime`. If the file hasn't changed, cache is used. This allows the second load to be instant.

### In-memory sorting
All sorting is done in memory. With 3000+ conversations, this is still instant.

### No database
No SQLite or similar is used to maintain simplicity. The JSONL files are already on disk and are fast enough.

## Security considerations

- The app only serves on localhost (doesn't expose anything to the network)
- Read-only: doesn't modify any Claude Code files
- Cache is optional and doesn't contain sensitive data
- Templates escape HTML to prevent XSS

---

# Decisiones Técnicas

## Stack

- **Lenguaje:** Go 1.21+
- **Dependencias:** Solo librería estándar (net/http, html/template, encoding/json)
- **Frontend:** HTML + CSS vanilla, sin JavaScript frameworks
- **Templates:** Go templates embebidos con `//go:embed`
- **Estilos:** CSS con variables para theming

## Arquitectura

```
main.go              # Código principal (toda la app en un archivo)
├── Tipos
│   ├── Message      # Mensaje individual (user/assistant)
│   ├── Conversation # Conversación con metadatos
│   ├── Project      # Proyecto con sus conversaciones
│   └── App          # Estado de la aplicación
│
├── Parsing
│   ├── parseConversationFile()  # Lee y parsea .jsonl
│   ├── extractTextContent()      # Extrae texto de content blocks
│   └── loadAllConversations()    # Carga inicial con cache
│
├── Cache
│   ├── loadCache()   # Carga cache desde disco
│   └── saveCache()   # Guarda cache en background
│
├── Ordenamiento
│   ├── getProjectsSorted()    # Ordena proyectos
│   └── sortConversations()    # Ordena conversaciones
│
└── Handlers HTTP
    ├── handleIndex        # GET /
    ├── handleProject      # GET /project?path=...
    ├── handleConversation # GET /conversation?id=...
    ├── handleSearch       # GET /search?q=...
    ├── handleAPIProjects  # GET /api/projects
    └── handleAPIConversation # GET /api/conversation?id=...

templates/           # Templates HTML embebidos
├── index.html       # Listado de proyectos
├── project.html     # Conversaciones de un proyecto
├── conversation.html # Vista de una conversación
└── search.html      # Resultados de búsqueda

static/
└── style.css        # Estilos con CSS variables para themes
```

## Estructura de datos de Claude Code

Claude Code almacena las conversaciones en:
```
~/.claude/
├── projects/
│   ├── -home-sebas-robotin/           # Nombre de proyecto (path con guiones)
│   │   ├── abc123.jsonl               # Una conversación por archivo
│   │   └── def456.jsonl
│   └── -home-sebas-doc-prj-gobot/
│       └── xyz789.jsonl
└── conversations-viewer-cache.json    # Cache de metadatos (creado por esta app)
```

Cada archivo `.jsonl` contiene líneas JSON con:
- `type: "user"` - Mensaje del usuario
- `type: "assistant"` - Respuesta del asistente
- `type: "summary"` - Títulos/resúmenes de la conversación

## Configuración

- **Puerto:** 8042 por defecto, configurable como primer argumento CLI
- **Cache:** `~/.claude/conversations-viewer-cache.json`
- **Tema:** Guardado en localStorage del browser

## Decisiones de diseño

### Lazy loading de mensajes
Los mensajes completos solo se cargan cuando se visualiza una conversación. La lista inicial solo carga metadatos (título, timestamps, conteos).

### Cache de metadatos
Se cachean los metadatos por archivo con el `modTime`. Si el archivo no cambió, se usa la cache. Esto permite que la segunda carga sea instantánea.

### Ordenamiento en memoria
Todo el ordenamiento se hace en memoria. Con 3000+ conversaciones, esto sigue siendo instantáneo.

### Sin base de datos
No se usa SQLite ni similar para mantener la simplicidad. Los archivos JSONL ya están en disco y son suficientemente rápidos.

## Consideraciones de seguridad

- La app solo sirve en localhost (no expone nada a la red)
- Solo lectura: no modifica ningún archivo de Claude Code
- El cache es opcional y no contiene datos sensibles
- Los templates escapan HTML para prevenir XSS
