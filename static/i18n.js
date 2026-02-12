// Internationalization system - English strings are keys
const i18n = {
    currentLang: 'es',

    // Spanish translations (English is the default/key)
    es: {
        // Menu
        'Dark mode': 'Modo oscuro',
        'Light mode': 'Modo claro',
        'Log out': 'Cerrar sesion',
        'English': 'English',
        'Espanol': 'Espanol',

        // Header & Navigation
        'Search all conversations...': 'Buscar en todas las conversaciones...',
        'Search in this project...': 'Buscar en este proyecto...',
        'Search in this conversation...': 'Buscar en esta conversacion...',
        'Search': 'Buscar',
        'Reload conversations': 'Recargar conversaciones',
        'Menu': 'Menu',
        'Go to top': 'Ir arriba',
        'Go to bottom': 'Ir abajo',

        // Stats
        'Projects': 'Proyectos',
        'Conversations': 'Conversaciones',
        'conversations': 'conversaciones',
        'messages': 'mensajes',
        'conv.': 'conv.',
        'msgs': 'msgs',
        'Title': 'Titulo',

        // Sort
        'Sort by:': 'Ordenar por:',
        'Last activity': 'Ultima actividad',
        'First activity': 'Primera actividad',
        'Name': 'Nombre',
        'Most messages': 'Mas mensajes',

        // Messages
        'User': 'Usuario',
        'Agent': 'Agente',
        '[:n:] user messages': '[:n:] mensajes de usuario',
        '[:n:] agent responses': '[:n:] respuestas del Agente',
        '[:n:] total': '[:n:] total',
        'Show more': 'Mostrar mas',
        'Show less': 'Mostrar menos',

        // Search results
        'Results for "[:query:]"': 'Resultados para "[:query:]"',
        'Results for "[:query:]" in [:project:]': 'Resultados para "[:query:]" en [:project:]',
        '[:n:] conversations found': '[:n:] conversaciones encontradas',
        '[:n:] matches': '[:n:] coincidencias',
        'No conversations found with that term.': 'No se encontraron conversaciones con ese termino.',
        'Search includes user messages and agent responses.': 'La busqueda incluye mensajes de usuario y respuestas del Agente.',
        'Expand search to all projects': 'Expandir busqueda a todos los proyectos',

        // Breadcrumb
        'Conversation': 'Conversacion',
        'Search results': 'Resultados',

        // Vim help
        'Keyboard shortcuts (Vim-style)': 'Atajos de teclado (Vim-style)',
        'Navigation': 'Navegacion',
        'Next message': 'Siguiente mensaje',
        'Previous message': 'Mensaje anterior',
        'First message': 'Primer mensaje',
        'Last message': 'Ultimo mensaje',
        'Page down': 'Bajar media pagina',
        'Page up': 'Subir media pagina',
        'Normal browser scroll': 'Scroll normal del browser',
        'Expand/Collapse': 'Expandir/Colapsar',
        'Expand message sections': 'Expandir secciones del mensaje',
        'Collapse message sections': 'Colapsar secciones del mensaje',
        'Toggle message expansion': 'Toggle expansion mensaje',
        'Selection and Yank': 'Seleccion y Yank',
        'Selection mode (toggle)': 'Modo seleccion (toggle)',
        'Copy selected message(s)': 'Copiar mensaje(s) seleccionado(s)',
        'Copy current message': 'Copiar mensaje actual',
        'Exit selection mode': 'Salir de modo seleccion',
        'Focus search': 'Enfocar busqueda',
        'Next match': 'Siguiente coincidencia',
        'Previous match': 'Coincidencia anterior',
        'Other': 'Otros',
        'Show/hide help': 'Mostrar/ocultar ayuda',
        'Press [:keys:] to close': 'Presiona [:keys:] para cerrar',

        // Notifications
        'Copied [:n:] message(s)': 'Copiado [:n:] mensaje(s)',
        'Copy error': 'Error al copiar',
        'No results': 'Sin resultados',

        // Login
        'Log in': 'Iniciar sesion',
        'Username': 'Usuario',
        'Password': 'Contrasena',
        'Enter': 'Entrar',
        'Invalid username or password': 'Usuario o contrasena incorrectos',

        // Visual mode
        '-- VISUAL -- ([:n:] selected)': '-- VISUAL -- ([:n:] seleccionado(s))',

        // Session
        'Session': 'Sesion'
    },

    init() {
        this.currentLang = localStorage.getItem('lang') || 'es';
        document.documentElement.setAttribute('data-lang', this.currentLang);
        this.updateAllTexts();
    },

    t(text, params = {}) {
        // Get translation or use original text
        let result = text;
        if (this.currentLang !== 'en' && this[this.currentLang] && this[this.currentLang][text]) {
            result = this[this.currentLang][text];
        }

        // Replace [:param:] placeholders
        for (const [key, value] of Object.entries(params)) {
            result = result.replace(new RegExp(`\\[:${key}:\\]`, 'g'), value);
        }

        return result;
    },

    setLang(lang) {
        this.currentLang = lang;
        localStorage.setItem('lang', lang);
        document.documentElement.setAttribute('data-lang', lang);
        this.updateAllTexts();
        // Reload page to update server-rendered content
        location.reload();
    },

    toggleLang() {
        const newLang = this.currentLang === 'en' ? 'es' : 'en';
        this.setLang(newLang);
    },

    updateAllTexts() {
        // Update all elements with data-i18n attribute
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const text = el.getAttribute('data-i18n');
            const paramsStr = el.getAttribute('data-i18n-params');
            const params = paramsStr ? JSON.parse(paramsStr) : {};
            el.textContent = this.t(text, params);
        });

        // Update all elements with data-i18n-placeholder attribute
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const text = el.getAttribute('data-i18n-placeholder');
            el.placeholder = this.t(text);
        });

        // Update all elements with data-i18n-title attribute
        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            const text = el.getAttribute('data-i18n-title');
            el.title = this.t(text);
        });
    }
};

// Shorthand function
function t(text, params = {}) {
    return i18n.t(text, params);
}

// Initialize on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => i18n.init());
} else {
    i18n.init();
}
