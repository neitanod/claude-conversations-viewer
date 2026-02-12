# Testing Plan

## How to run tests

```bash
cd ~/robotin/apps/claude-conversations-viewer

# Run tests with coverage
go test -v -cover

# View detailed coverage
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Current coverage

**80.7%** of statements covered.

## Implemented unit tests

### Utility functions
- `TestProjectDirToPath` - Directory name to path conversion
- `TestTruncateString` - String truncation with ellipsis
- `TestExtractTextContent` - Text extraction from content blocks (string and array)

### Sorting
- `TestSortConversations` - Sorting by name, date, messages
- `TestGetProjectsSorted` - Project sorting by various criteria

### Template functions
- `TestTemplateFunctions` - formatTime, formatDate, formatDateTime, truncate, join, nl2br
- `TestTemplateFunctionsStringOps` - hasPrefix, contains, lower

### File parsing
- `TestAppParseConversationFile` - Complete JSONL file parsing
- `TestAppLoadAllConversations` - Loading multiple projects
- `TestParseEmptyConversation` - Empty file handling
- `TestParseConversationWithArrayContent` - Content blocks as array
- `TestDuplicateTitlesRemoved` - Title deduplication

### Cache
- `TestCacheLoadSave` - Save and load cache
- `TestCacheLoadMissingFile` - Missing cache doesn't cause error
- `TestLoadAllConversationsWithCache` - Correct cache usage
- `TestLoadAllConversationsSkipsNonJSONL` - Ignores non-.jsonl files

### HTTP Handlers
- `TestHandlerIndex` - Main page
- `TestHandlerIndexWithSort` - Sorting on main page
- `TestHandlerProject` - Project view (valid, missing path, not found)
- `TestHandlerConversation` - Conversation view (valid, missing id, not found)
- `TestHandlerSearch` - Search (empty query, valid search, no results)
- `TestHandlerAPIProjects` - Projects JSON API
- `TestHandlerAPIConversation` - Conversation JSON API

### Lazy loading
- `TestLoadFullConversation` - On-demand loading
- `TestSearchInTitles` - Search includes titles

## Manual test cases

### Test 1: App startup
```bash
./claude-conversations-viewer
```
**Expected result:**
- Log shows "Loading conversations..."
- Log shows number of conversations and projects loaded
- Server starts at http://localhost:8042

### Test 2: Basic navigation
1. Open http://localhost:8042
2. Click on a project
3. Click on a conversation

**Expected result:**
- Each page loads without errors
- Data displays correctly

### Test 3: Theme change
1. Click on theme button (sun/moon)
2. Verify it changes to dark
3. Reload page

**Expected result:**
- Theme persists after reload

### Test 4: Search
1. Type a term in the search box
2. Press Enter

**Expected result:**
- Results show conversations containing the term
- Previews show context around the match

### Test 5: Sorting
1. On main page, click "Name"
2. Verify alphabetical order
3. Click "More messages"
4. Verify order by count

**Expected result:**
- Projects reorder according to selected criterion

### Test 6: Custom port
```bash
./claude-conversations-viewer 9000
```
**Expected result:**
- Server starts at http://localhost:9000

## Covered edge cases

- Empty JSONL files
- Content as string vs as array of blocks
- Duplicate titles in a conversation
- Conversations without titles (uses first message)
- Search in titles as well as content
- Cache with modified files (must re-parse)
- Cache with unmodified files (must use cache)
- Projects directory with non-JSONL files

---

# Plan de Testing

## Cómo ejecutar tests

```bash
cd ~/robotin/apps/claude-conversations-viewer

# Ejecutar tests con cobertura
go test -v -cover

# Ver cobertura detallada
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Cobertura actual

**80.7%** de statements cubiertos.

## Tests unitarios implementados

### Funciones de utilidad
- `TestProjectDirToPath` - Conversión de nombres de directorio a paths
- `TestTruncateString` - Truncado de strings con ellipsis
- `TestExtractTextContent` - Extracción de texto de content blocks (string y array)

### Ordenamiento
- `TestSortConversations` - Ordenamiento por nombre, fecha, mensajes
- `TestGetProjectsSorted` - Ordenamiento de proyectos por varios criterios

### Template functions
- `TestTemplateFunctions` - formatTime, formatDate, formatDateTime, truncate, join, nl2br
- `TestTemplateFunctionsStringOps` - hasPrefix, contains, lower

### Parsing de archivos
- `TestAppParseConversationFile` - Parsing de archivo JSONL completo
- `TestAppLoadAllConversations` - Carga de múltiples proyectos
- `TestParseEmptyConversation` - Manejo de archivos vacíos
- `TestParseConversationWithArrayContent` - Content blocks como array
- `TestDuplicateTitlesRemoved` - Deduplicación de títulos

### Cache
- `TestCacheLoadSave` - Guardar y cargar cache
- `TestCacheLoadMissingFile` - Cache inexistente no causa error
- `TestLoadAllConversationsWithCache` - Uso correcto del cache
- `TestLoadAllConversationsSkipsNonJSONL` - Ignora archivos no-.jsonl

### HTTP Handlers
- `TestHandlerIndex` - Página principal
- `TestHandlerIndexWithSort` - Ordenamiento en página principal
- `TestHandlerProject` - Vista de proyecto (válido, missing path, not found)
- `TestHandlerConversation` - Vista de conversación (válido, missing id, not found)
- `TestHandlerSearch` - Búsqueda (empty query, valid search, no results)
- `TestHandlerAPIProjects` - API JSON de proyectos
- `TestHandlerAPIConversation` - API JSON de conversación

### Lazy loading
- `TestLoadFullConversation` - Carga bajo demanda
- `TestSearchInTitles` - Búsqueda incluye títulos

## Casos de prueba manuales

### Test 1: Inicio de la app
```bash
./claude-conversations-viewer
```
**Resultado esperado:**
- Log muestra "Loading conversations..."
- Log muestra cantidad de conversaciones y proyectos cargados
- Servidor inicia en http://localhost:8042

### Test 2: Navegación básica
1. Abrir http://localhost:8042
2. Click en un proyecto
3. Click en una conversación

**Resultado esperado:**
- Cada página carga sin errores
- Los datos se muestran correctamente

### Test 3: Cambio de tema
1. Click en botón de tema (sol/luna)
2. Verificar que cambia a oscuro
3. Recargar página

**Resultado esperado:**
- El tema persiste después de recargar

### Test 4: Búsqueda
1. Escribir un término en el buscador
2. Presionar Enter

**Resultado esperado:**
- Resultados muestran conversaciones que contienen el término
- Previews muestran contexto alrededor del match

### Test 5: Ordenamiento
1. En página principal, click en "Nombre"
2. Verificar orden alfabético
3. Click en "Más mensajes"
4. Verificar orden por cantidad

**Resultado esperado:**
- Los proyectos se reordenan según el criterio seleccionado

### Test 6: Puerto personalizado
```bash
./claude-conversations-viewer 9000
```
**Resultado esperado:**
- Servidor inicia en http://localhost:9000

## Edge cases cubiertos

- Archivos JSONL vacíos
- Content como string vs como array de blocks
- Títulos duplicados en una conversación
- Conversaciones sin títulos (usa primer mensaje)
- Búsqueda en títulos además de contenido
- Cache con archivos modificados (debe re-parsear)
- Cache con archivos no modificados (debe usar cache)
- Directorio de proyectos con archivos no-JSONL
