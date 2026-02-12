# Claude Conversations Viewer

## Problem it solves

Claude Code stores conversations in `~/.claude/projects/` fragmented by project. There is no easy way to:
- View all conversations in a unified manner
- Search text in old conversations
- Navigate history visually

## Goal

Provide a local web interface to explore, search and read all Claude Code conversations, regardless of which project they originated from.

## Users

- **Sebastián**: To review past conversations, search for previously implemented solutions, and have context of the work done.
- **Robotín**: To consult previous conversations when needing context about past decisions or implementations.

## Main use cases

- View list of all projects with conversations, sorted by recent activity
- Navigate conversations of a specific project
- Read a complete conversation with all messages (user + assistant)
- Search text in all conversations (includes user and assistant messages)
- Switch between light and dark theme according to preference

## Implemented features

- Complete parsing of `.jsonl` files with user/assistant messages
- Metadata cache system for fast loading (~5 seconds first time, instant afterwards)
- Multiple sorting options (last activity, first activity, name, message count)
- Extraction and display of conversation titles/summaries
- Full-text search in all content
- UI with collapsible content for long messages
- Light and dark mode with localStorage persistence
- JSON API for programmatic access

## Out of scope

- Conversation editing
- Export to other formats
- Synchronization with Claude web
- Regular expression search
- Advanced filters (by date, message type, etc.)

---

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
- Buscar texto en todas las conversaciones (incluye mensajes del usuario y del asistente)
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
