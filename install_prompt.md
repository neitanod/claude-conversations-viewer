# Prompt de instalación de claude-conversations-viewer

Este archivo está pensado para ser pegado tal cual a un agente de IA con acceso
a una terminal (Claude Code, Cursor, etc.) para que instale
**claude-conversations-viewer** de forma autónoma en Linux, macOS o Windows.

---

## Prompt

Instalá **claude-conversations-viewer** desde el código fuente en esta máquina.
Es una webapp local en Go que unifica todas las conversaciones de Claude Code
almacenadas en `~/.claude/projects/` en una interfaz web navegable. El
repositorio oficial es <https://github.com/neitanod/claude-conversations-viewer>.

Seguí estos pasos en orden y al final reportá brevemente qué quedó instalado y
cómo arrancar la app.

### 1. Detectar el sistema operativo

Identificá si estás en Linux, macOS o Windows antes de empezar. Los scripts
y los nombres de binarios cambian según la plataforma.

### 2. Verificar requisitos

- **Go 1.21 o superior** — verificá con `go version`.
- **git** — verificá con `git --version`.

Si alguno falta, avisá al usuario y pedile que lo instale. No intentes
instalar Go automáticamente.

### 3. Clonar el repositorio

Elegí un directorio razonable según el sistema:

- Linux/macOS: `~/code/claude-conversations-viewer` o `~/src/claude-conversations-viewer`.
- Windows: `%USERPROFILE%\code\claude-conversations-viewer`.

Cloná con:

```
git clone https://github.com/neitanod/claude-conversations-viewer.git <ruta-elegida>
```

Si el directorio ya existe y tiene un clon válido, hacé `git pull --ff-only` en
vez de volver a clonar.

### 4. Compilar

**En Linux o macOS:**

```bash
cd <ruta-elegida>
./build              # genera ./conversations-viewer
```

**En Windows (PowerShell):**

```powershell
cd <ruta-elegida>
.\build.ps1          # genera .\conversations-viewer.exe
```

Si PowerShell bloquea el script por política de ejecución, usá:

```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

### 5. (Opcional) Configurar autenticación

Por defecto la app no requiere login. Si el usuario quiere protegerla,
preguntale y creá:

- Linux/macOS: `~/.claude/conversations-viewer-config.json`
- Windows: `%USERPROFILE%\.claude\conversations-viewer-config.json`

Con contenido:

```json
{
  "username": "<usuario>",
  "password": "<password>"
}
```

### 6. Arrancar la app

**En Linux o macOS:**

```bash
./run                # arranca en el puerto 8042
./conversations-viewer 9000   # arranca en otro puerto
```

**En Windows (PowerShell):**

```powershell
.\run.ps1
# o:
.\conversations-viewer.exe 9000
```

### 7. Verificar que esté andando

Abrí <http://localhost:8042> en el navegador o desde la terminal:

```
curl http://localhost:8042/api/projects
```

Debería devolver un JSON con los proyectos detectados en
`~/.claude/projects/`. Si la respuesta es `[]`, es porque todavía no hay
conversaciones de Claude Code en esta máquina, no porque la app falle.

### 8. Reportar resultado

Al final, informá al usuario:

- Ruta donde quedó instalado el repo.
- Comando exacto para arrancar la app la próxima vez.
- URL para abrir en el navegador.
- Si dejaste o no la app corriendo en background.
- Cualquier paso opcional que no completaste.

### Notas importantes

- **No modifiques `git config --global`.** Si necesitás identidad git para
  algún paso, usá `git config` local al repo y avisalo.
- En Windows el path mostrado por la app para cada proyecto es una
  reconstrucción heurística (el formato de carpeta `~/.claude/projects/`
  reemplaza separadores y espacios por `-`, lo que es ambiguo). Si un
  proyecto se muestra con un `\` en lugar de un espacio, es esperado.
- La app sólo lee `~/.claude/projects/` y escribe tres archivos sueltos en
  `~/.claude/conversations-viewer-{cache,config,sessions}.json`. No requiere
  base de datos ni servicios externos.
