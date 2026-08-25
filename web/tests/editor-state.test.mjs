import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import ts from 'typescript';

const source = await readFile(new URL('../src/features/editor/state.ts', import.meta.url), 'utf8');
const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 } }).outputText;
const state = await import(`data:text/javascript;base64,${Buffer.from(code).toString('base64')}`);

test('editor diagnostics only contain diagnostics for the open file', () => {
    const diagnostics = [
        { code: 'current', path: 'docs/current.md' },
        { code: 'other', path: 'docs/other.md' },
        { code: 'pathless' },
    ];

    assert.deepEqual(
        state.diagnosticsForEditor(diagnostics, 'docs/current.md').map((diagnostic) => diagnostic.code),
        ['current'],
    );
    assert.equal(diagnostics.length, 3);
});

test('editor workflow wires file-local diagnostics and current-response gates', async () => {
    const editorSource = await readFile(new URL('../src/features/editor/app.tsx', import.meta.url), 'utf8');
    assert.match(editorSource, /setDiagnostics\(diagnosticsForEditor\(diagnostics, file\.path\)\)/);
    assert.equal(editorSource.match(/editorResponseIsCurrent\(/g)?.length, 2);
    assert.match(editorSource, /const applyFile[\s\S]*?validateGeneration\.current\+\+[\s\S]*?previewGeneration\.current\+\+/);
    assert.match(editorSource, /handle\.current\?\.destroy\(\)/);
});

test('editor ignores responses for another file or an older request', () => {
    assert.equal(state.editorResponseIsCurrent('docs/current.md', 'docs/current.md', 4, 4), true);
    assert.equal(state.editorResponseIsCurrent('docs/old.md', 'docs/current.md', 4, 4), false);
    assert.equal(state.editorResponseIsCurrent('docs/current.md', 'docs/current.md', 3, 4), false);
});

test('editor builds a file tree with root files, nested directories, and repeated names', () => {
    const files = [
        { path: 'index.md', language: 'markdown' },
        { path: 'docs/guide/setup.md', language: 'markdown' },
        { path: 'docs/reference/setup.md', language: 'markdown' },
        { path: 'settings/config.yaml', language: 'yaml' },
    ];

    assert.deepEqual(state.buildFileTree(files), [
        { kind: 'file', name: 'index.md', file: files[0] },
        { kind: 'directory', name: 'docs', path: 'docs', children: [
            { kind: 'directory', name: 'guide', path: 'docs/guide', children: [
                { kind: 'file', name: 'setup.md', file: files[1] },
            ] },
            { kind: 'directory', name: 'reference', path: 'docs/reference', children: [
                { kind: 'file', name: 'setup.md', file: files[2] },
            ] },
        ] },
        { kind: 'directory', name: 'settings', path: 'settings', children: [
            { kind: 'file', name: 'config.yaml', file: files[3] },
        ] },
    ]);
});

test('editor converts Go UTF-8 byte columns to CodeMirror UTF-16 offsets', () => {
    assert.equal(state.utf16OffsetForUTF8Column('ascii', 4), 3);
    assert.equal(state.utf16OffsetForUTF8Column('{"ключ": }', 14), 9);
    assert.equal(state.utf16OffsetForUTF8Column('я😀x', 3), 1);
    assert.equal(state.utf16OffsetForUTF8Column('я😀x', 7), 3);
    assert.equal(state.utf16OffsetForUTF8Column('я😀x', 8), 4);
});
