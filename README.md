# Claude Conversations Viewer

Interfaz web local para explorar, buscar y leer todas las conversaciones con Claude Code, sin importar en qué proyecto se originaron.

## Problema que resuelve

Claude Code almacena las conversaciones en `~/.claude/projects/` de forma fragmentada por proyecto. Esta app unifica todas las conversaciones en una interfaz web navegable con búsqueda y ordenamiento.

## Instalación

```bash
# El binario está en:
~/robotin/apps/claude-conversations-viewer/claude-conversations-viewer

# O ejecutar directamente:
cd ~/robotin/apps/claude-conversations-viewer
./run
```

## Uso

```bash
# Iniciar en puerto por defecto (8042)
./claude-conversations-viewer

# Iniciar en puerto personalizado
./claude-conversations-viewer 9000
```

Luego abrir http://localhost:8042 en el navegador.

## Características

- Vista de todos los proyectos con conversaciones
- Navegación por proyecto y conversación individual
- Búsqueda full-text en mensajes de usuario y Claude
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

## Configuración

- **Puerto:** Primer argumento CLI (default: 8042)
- **Cache:** `~/.claude/conversations-viewer-cache.json` (se crea automáticamente)
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
