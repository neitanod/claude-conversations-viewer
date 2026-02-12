# Changelog - Claude Conversations Viewer

## 2026-02-12

### Autenticacion (opcional)
- Sistema de login con usuario/password (deshabilitado por defecto)
- Para habilitar: crear `~/.claude/conversations-viewer-config.json` con username/password
- Sesiones persistentes de 30 dias en `~/.claude/conversations-viewer-sessions.json`

### Renombrado de Claude a Agent
- Los mensajes del asistente ahora muestran "Agent" en lugar de "Claude"
- Cambio en todos los templates y funciones de yank

### Navegacion Vim mejorada
- **j/k**: Ahora posicionan el mensaje debajo del header + 20px (antes se centraba o quedaba oculto)
- **J/K** (mayusculas): Agregado scroll normal del browser (80px por pulsacion) para poder leer mensajes largos sin cambiar de item
- Las flechas arriba/abajo siguen funcionando como scroll normal del browser

### Busqueda por proyecto
- En la pagina de un proyecto, el buscador ahora solo busca dentro de ese proyecto
- En resultados de busqueda se muestra el contexto del proyecto y un link para expandir la busqueda globalmente
- En la pagina de conversacion hay busqueda local con JavaScript (resaltado de coincidencias)

### Controles Vim anteriores (mismo dia)
- **gg/G**: Ir al primer/ultimo mensaje
- **v**: Modo seleccion visual
- **y/Y**: Copiar mensaje(s) seleccionado(s)
- **h/l**: Colapsar/expandir secciones del mensaje actual
- **H** o **?**: Mostrar/ocultar ayuda de atajos
- **Ctrl+d/Ctrl+u**: Bajar/subir media pagina
- **/**: Activar busqueda local
- **n/N**: Siguiente/anterior coincidencia de busqueda
