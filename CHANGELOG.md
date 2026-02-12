# Changelog - Claude Conversations Viewer

## 2026-02-12

### Authentication (optional)
- Login system with username/password (disabled by default)
- To enable: create `~/.claude/conversations-viewer-config.json` with username/password
- Persistent 30-day sessions in `~/.claude/conversations-viewer-sessions.json`

### Renamed Claude to Agent
- Assistant messages now display "Agent" instead of "Claude"
- Changed in all templates and yank functions

### Improved Vim navigation
- **j/k**: Now position the message below the header + 20px (previously it was centered or hidden)
- **J/K** (uppercase): Added normal browser scroll (80px per keystroke) to read long messages without changing items
- Arrow keys up/down still work as normal browser scroll

### Project-scoped search
- On a project page, the search now only searches within that project
- Search results show project context and a link to expand to global search
- On conversation page there's local JavaScript search (match highlighting)

### Previous Vim controls (same day)
- **gg/G**: Go to first/last message
- **v**: Visual selection mode
- **y/Y**: Copy selected message(s)
- **h/l**: Collapse/expand sections of current message
- **H** or **?**: Show/hide keyboard shortcuts help
- **Ctrl+d/Ctrl+u**: Scroll down/up half page
- **/**: Activate local search
- **n/N**: Next/previous search match

---

# Changelog - Claude Conversations Viewer

## 2026-02-12

### Autenticación (opcional)
- Sistema de login con usuario/password (deshabilitado por defecto)
- Para habilitar: crear `~/.claude/conversations-viewer-config.json` con username/password
- Sesiones persistentes de 30 días en `~/.claude/conversations-viewer-sessions.json`

### Renombrado de Claude a Agent
- Los mensajes del asistente ahora muestran "Agent" en lugar de "Claude"
- Cambio en todos los templates y funciones de yank

### Navegación Vim mejorada
- **j/k**: Ahora posicionan el mensaje debajo del header + 20px (antes se centraba o quedaba oculto)
- **J/K** (mayúsculas): Agregado scroll normal del browser (80px por pulsación) para poder leer mensajes largos sin cambiar de item
- Las flechas arriba/abajo siguen funcionando como scroll normal del browser

### Búsqueda por proyecto
- En la página de un proyecto, el buscador ahora solo busca dentro de ese proyecto
- En resultados de búsqueda se muestra el contexto del proyecto y un link para expandir la búsqueda globalmente
- En la página de conversación hay búsqueda local con JavaScript (resaltado de coincidencias)

### Controles Vim anteriores (mismo día)
- **gg/G**: Ir al primer/último mensaje
- **v**: Modo selección visual
- **y/Y**: Copiar mensaje(s) seleccionado(s)
- **h/l**: Colapsar/expandir secciones del mensaje actual
- **H** o **?**: Mostrar/ocultar ayuda de atajos
- **Ctrl+d/Ctrl+u**: Bajar/subir media página
- **/**: Activar búsqueda local
- **n/N**: Siguiente/anterior coincidencia de búsqueda
