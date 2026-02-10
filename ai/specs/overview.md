# Claude Conversations Viewer

## Problema que resuelve

Claude Code almacena las conversaciones en `~/.claude/projects/` de forma fragmentada por proyecto. No existe una forma sencilla de:
- Ver todas las conversaciones de forma unificada
- Buscar texto en conversaciones antiguas
- Navegar el historial de forma visual

## Objetivo

Proveer una interfaz web local para explorar, buscar y leer todas las conversaciones con Claude Code, sin importar en qué proyecto se originaron.

## Usuarios

- **Sebastián**: Para revisar conversaciones pasadas, buscar soluciones implementadas anteriormente, y tener contexto de lo trabajado.
- **Robotín**: Para consultar conversaciones anteriores cuando necesite contexto sobre decisiones o implementaciones previas.

## Casos de uso principales

- Ver lista de todos los proyectos con conversaciones, ordenados por actividad reciente
- Navegar las conversaciones de un proyecto específico
- Leer una conversación completa con todos los mensajes (user + assistant)
- Buscar texto en todas las conversaciones (incluye mensajes del usuario y de Claude)
- Cambiar entre tema claro y oscuro según preferencia

## Características implementadas

- Parsing completo de archivos `.jsonl` con mensajes user/assistant
- Sistema de cache de metadatos para carga rápida (~5 segundos primera vez, instantáneo después)
- Múltiples opciones de ordenamiento (última actividad, primera actividad, nombre, cantidad de mensajes)
- Extracción y display de títulos/summaries de conversaciones
- Búsqueda full-text en todo el contenido
- UI con contenido colapsable para mensajes largos
- Modo claro y oscuro con persistencia en localStorage
- API JSON para acceso programático

## Fuera de alcance

- Edición de conversaciones
- Exportación a otros formatos
- Sincronización con Claude web
- Búsqueda con expresiones regulares
- Filtros avanzados (por fecha, por tipo de mensaje, etc.)
