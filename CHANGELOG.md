# Changelog - Claude Conversations Viewer

## 2026-02-12

### Navegacion Vim mejorada
- **j/k**: Ahora posicionan el mensaje seleccionado a 20px del tope del viewport (antes lo centraban, lo cual era molesto para mensajes largos)
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
