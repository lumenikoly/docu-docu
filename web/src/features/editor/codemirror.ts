import { basicSetup, EditorView } from 'codemirror';
import { Compartment, EditorState } from '@codemirror/state';
import { markdown } from '@codemirror/lang-markdown';
import { json } from '@codemirror/lang-json';
import { yaml } from '@codemirror/lang-yaml';
import { go } from '@codemirror/lang-go';
import { java } from '@codemirror/lang-java';
import { javascript } from '@codemirror/lang-javascript';
import { setDiagnostics } from '@codemirror/lint';
import { MergeView } from '@codemirror/merge';
import { utf16OffsetForUTF8Column } from './state';
function languageExtension(language: any) {
    if (language === 'json')
        return json();
    if (language === 'yaml')
        return yaml();
    if (language === 'markdown')
        return markdown();
    if (language === 'go')
        return go();
    if (language === 'java')
        return java();
    if (language === 'javascript')
        return javascript({ jsx: true });
    if (language === 'typescript')
        return javascript({ jsx: true, typescript: true });
    return [];
}
function position(view: any, line: any, column: any) {
    const safeLine: any = Math.max(1, Math.min(Number(line) || 1, view.state.doc.lines));
    const record: any = view.state.doc.line(safeLine);
    return record.from + utf16OffsetForUTF8Column(record.text, column);
}
function appearanceTheme(theme: any) {
    return EditorView.theme({
        '&': { color: 'var(--td-text)', backgroundColor: 'var(--td-surface)' },
        '.cm-content': { caretColor: 'var(--td-accent)' },
        '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--td-accent)' },
        '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': { backgroundColor: 'var(--td-accent-soft)' },
        '.cm-gutters': { color: 'var(--td-text-muted)', backgroundColor: 'var(--td-surface-muted)', borderRightColor: 'var(--td-border)' },
        '.cm-activeLine, .cm-activeLineGutter': { backgroundColor: 'color-mix(in srgb, var(--td-accent) 7%, transparent)' },
        '&.cm-merge-a .cm-changedLine, &.cm-merge-b .cm-changedLine': { backgroundColor: 'transparent' },
        '&.cm-merge-a .cm-activeLine, &.cm-merge-b .cm-activeLine, &.cm-merge-a .cm-activeLineGutter, &.cm-merge-b .cm-activeLineGutter': { backgroundColor: 'transparent' },
        '&.cm-merge-a .cm-changedText': { background: 'var(--td-status-danger-soft)' },
        '&.cm-merge-b .cm-changedText': { background: 'var(--td-status-success-soft)' },
    }, { dark: theme === 'dark' });
}
const currentTheme: any = () => document.documentElement.dataset.theme || 'light';
function reviewSelection(view: any, side: any) {
    const range: any = view.state.selection.main;
    if (range.empty)
        return null;
    const toPosition: any = (offset: any) => {
        const line: any = view.state.doc.lineAt(offset);
        return { line: line.number, column: Array.from(view.state.sliceDoc(line.from, offset)).length + 1 };
    };
    return { side, start: toPosition(range.from), end: toPosition(range.to) };
}
function reviewPosition(view: any, value: any) {
    const lineNumber: any = Math.max(1, Math.min(Number(value?.line) || 1, view.state.doc.lines));
    const line: any = view.state.doc.line(lineNumber);
    const scalarPrefix: any = Array.from(view.state.sliceDoc(line.from, line.to)).slice(0, Math.max(0, (Number(value?.column) || 1) - 1)).join('');
    return line.from + scalarPrefix.length;
}
function highlightReviewRange(view: any, start: any, end: any) {
    const anchor: any = reviewPosition(view, start);
    const head: any = Math.max(anchor, reviewPosition(view, end));
    view.dispatch({ selection: { anchor, head }, scrollIntoView: true });
    view.focus();
}
window.ToudocuCodeMirror = {
    createMerge({ parent, before, after, language, onSelect }: any) {
        const themeA: any = new Compartment();
        const themeB: any = new Compartment();
        let highlighting: any = false;
        const readOnly: any = [basicSetup, languageExtension(language), EditorState.readOnly.of(true), EditorView.editable.of(false)];
        const selectionExtension: any = (side: any) => EditorView.updateListener.of((update: any) => {
            if (update.selectionSet && !highlighting)
                onSelect?.(reviewSelection(update.view, side));
        });
        const view: any = new MergeView({
            parent,
            a: { doc: before, extensions: [readOnly, selectionExtension('old'), themeA.of(appearanceTheme(currentTheme()))] },
            b: { doc: after, extensions: [readOnly, selectionExtension('new'), themeB.of(appearanceTheme(currentTheme()))] },
            collapseUnchanged: { margin: 3, minSize: 6 },
            highlightChanges: true,
            gutter: true,
        });
        return {
            destroy: () => view.destroy(),
            setTheme(theme: any) {
                view.a.dispatch({ effects: themeA.reconfigure(appearanceTheme(theme)) });
                view.b.dispatch({ effects: themeB.reconfigure(appearanceTheme(theme)) });
            },
            highlight(side: any, start: any, end: any) {
                highlighting = true;
                try {
                    highlightReviewRange(side === 'old' ? view.a : view.b, start, end);
                }
                finally {
                    highlighting = false;
                }
            },
        };
    },
    createViewer({ parent, doc, language, onSelect }: any) {
        const themeCompartment: any = new Compartment();
        let highlighting: any = false;
        const view: any = new EditorView({
            parent,
            state: EditorState.create({
                doc,
                extensions: [
                    basicSetup,
                    languageExtension(language),
                    EditorView.lineWrapping,
                    EditorState.readOnly.of(true),
                    EditorView.editable.of(false),
                    themeCompartment.of(appearanceTheme(currentTheme())),
                    EditorView.updateListener.of((update: any) => {
                        if (update.selectionSet && !highlighting)
                            onSelect?.(reviewSelection(update.view, 'new'));
                    }),
                ],
            }),
        });
        return {
            destroy: () => view.destroy(),
            focus: () => view.focus(),
            setTheme(theme: any) { view.dispatch({ effects: themeCompartment.reconfigure(appearanceTheme(theme)) }); },
            highlight(_side: any, start: any, end: any) {
                highlighting = true;
                try {
                    highlightReviewRange(view, start, end);
                }
                finally {
                    highlighting = false;
                }
            },
        };
    },
    create({ parent, doc, language, onChange }: any) {
        let applying: any = false;
        const themeCompartment: any = new Compartment();
        const view: any = new EditorView({
            parent,
            state: EditorState.create({
                doc,
                extensions: [
                    basicSetup,
                    languageExtension(language),
                    EditorView.lineWrapping,
                    themeCompartment.of(appearanceTheme(currentTheme())),
                    EditorView.updateListener.of((update: any) => {
                        if (update.docChanged && !applying)
                            onChange();
                    }),
                ],
            }),
        });
        return {
            getValue: () => view.state.doc.toString(),
            setValue(value: any) {
                applying = true;
                view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: value } });
                applying = false;
            },
            focus: () => view.focus(),
            destroy: () => view.destroy(),
            setTheme(theme: any) { view.dispatch({ effects: themeCompartment.reconfigure(appearanceTheme(theme)) }); },
            gotoLine(line: any, column: any) {
                const anchor: any = position(view, line, column);
                view.dispatch({ selection: { anchor }, scrollIntoView: true });
                view.focus();
            },
            setDiagnostics(diagnostics: any) {
                view.dispatch(setDiagnostics(view.state, diagnostics.map((item: any) => {
                    const from: any = position(view, item.line, item.column);
                    return { from, to: Math.min(view.state.doc.length, from + 1), severity: item.severity === 'error' ? 'error' : 'warning', message: item.message, source: item.code };
                })));
            },
        };
    },
};
