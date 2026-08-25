import { text } from "./locale";
(() => {
    'use strict';
    const root: any = document.documentElement;
    const script: any = document.currentScript;
    const media: any = matchMedia('(prefers-color-scheme: dark)');
    const definitions: any = {
        siteTheme: { attribute: 'siteTheme', key: 'toudocu-site-theme', values: ['classic', 'paper', 'terminal'], fallback: 'classic' },
        colorScheme: { attribute: 'colorScheme', key: 'toudocu-color-scheme', values: ['system', 'light', 'dark'], fallback: 'system' },
        accent: { attribute: 'accent', key: 'toudocu-accent', values: ['indigo', 'blue', 'teal', 'green', 'amber', 'rose', 'violet'], fallback: 'indigo' },
        density: { attribute: 'density', key: 'toudocu-density', values: ['comfortable', 'compact'], fallback: 'comfortable' },
        contentWidth: { attribute: 'contentWidth', key: 'toudocu-content-width', values: ['narrow', 'standard', 'wide'], fallback: 'standard' },
    };
    const state: any = {};
    const valid: any = (definition: any, value: any) => definition.values.includes(value);
    const initial: any = (definition: any, name: any) => {
        let saved: any = null;
        try {
            saved = localStorage.getItem(definition.key);
        }
        catch { /* privacy mode */ }
        if (valid(definition, saved))
            return saved;
        const serverDefault: any = root.dataset[definition.attribute] || script?.dataset[`default${name[0].toUpperCase()}${name.slice(1)}`];
        return valid(definition, serverDefault) ? serverDefault : definition.fallback;
    };
    Object.entries(definitions).forEach(([name, definition]: any) => { state[name] = initial(definition, name); });
    function syncControls() {
        document.querySelectorAll('[data-site-theme-select]').forEach((select: any) => { select.value = state.siteTheme; });
        document.querySelectorAll('[data-color-scheme-select]').forEach((select: any) => { select.value = state.colorScheme; });
        const themeLabels: any = { classic: text("core.preferences.001"), paper: text("core.preferences.002"), terminal: text("core.preferences.003") };
        const indicators: any = { classic: 'C', paper: 'P', terminal: 'T' };
        const schemeLabels: any = { system: text("core.preferences.004"), light: text("core.preferences.005"), dark: text("core.preferences.006") };
        document.querySelectorAll('[data-site-theme-label]').forEach((node: any) => { node.textContent = themeLabels[state.siteTheme]; });
        document.querySelectorAll('[data-site-theme-indicator]').forEach((node: any) => { node.textContent = indicators[state.siteTheme]; });
        document.querySelectorAll('[data-theme-label]').forEach((node: any) => { node.textContent = schemeLabels[state.colorScheme]; });
    }
    function apply(announce: any = true) {
        Object.entries(definitions).forEach(([name, definition]: any) => { root.dataset[definition.attribute] = state[name]; });
        const resolved: any = state.colorScheme === 'system' ? (media.matches ? 'dark' : 'light') : state.colorScheme;
        root.dataset.theme = resolved;
        syncControls();
        if (announce)
            document.dispatchEvent(new CustomEvent('toudocu:themechange', { detail: { ...state, mode: state.colorScheme, theme: resolved } }));
    }
    function syncEditorLink(path?: string) {
        const editorLink: any = document.querySelector('[data-workspace="editor"]');
        if (editorLink) {
            try {
                const selected: any = path || sessionStorage.getItem('toudocu-editor-path');
                if (selected) {
                    const target: any = new URL(editorLink.href, location.href);
                    target.searchParams.set('path', selected);
                    editorLink.setAttribute('href', `${target.pathname}${target.search}`);
                }
            }
            catch { /* storage may be unavailable */ }
        }
    }
    function bind() {
        syncEditorLink();
        document.addEventListener('toudocu:editorpathchange', (event: any) => syncEditorLink(event.detail?.path));
        document.querySelectorAll('[data-site-theme-select]').forEach((select: any) => select.addEventListener('change', () => set('siteTheme', select.value)));
        document.querySelectorAll('[data-color-scheme-select]').forEach((select: any) => select.addEventListener('change', () => set('colorScheme', select.value)));
        syncControls();
    }
    function set(name: any, value: any) {
        const definition: any = definitions[name];
        if (!definition?.values.includes(value))
            return;
        state[name] = value;
        try {
            localStorage.setItem(definition.key, value);
        }
        catch { /* privacy mode */ }
        apply();
    }
    apply(false);
    if (document.readyState === 'loading')
        document.addEventListener('DOMContentLoaded', bind, { once: true });
    else
        bind();
    media.addEventListener?.('change', () => {
        if (state.colorScheme === 'system')
            apply();
    });
    window.ToudocuAppearance = { get: () => ({ ...state, theme: root.dataset.theme }), set, apply };
})();
