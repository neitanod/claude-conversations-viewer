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
