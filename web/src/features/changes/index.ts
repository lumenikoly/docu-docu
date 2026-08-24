import { text } from "../../core/locale";
import { createDiscussionPanel } from "../../components/discussion-panel";
(() => {
    'use strict';
    const page: any = window.ToudocuPage;
    const locale: any = page?.ui.locale || 'en';
    const API: any = page?.runtime === 'serve' && page.capabilities?.changes ? page.endpoints?.changes : '';
    const REVIEW: any = page?.runtime === 'serve' && page.capabilities?.review ? page.endpoints?.review : '';
    const REPOSITORY_REVIEW: any = API ? `${API}/review` : '';
    const EDITOR_WORKSPACE: any = page?.runtime === 'serve' && page.capabilities?.editor ? page.endpoints?.editorWorkspace : '';
    const $: any = (selector: any, root: any = document) => root.querySelector(selector);
    const escapeHTML: any = (value: any) => String(value ?? '').replace(/[&<>"']/g, (character: any) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' } as Record<string, string>)[character]);
    const { panel, scrim } = createDiscussionPanel({
        ariaLabel: text("changes.discussions"),
        className: 'changes-review-panel',
        closeAttribute: 'data-discussions-close',
        closeLabel: text("core.portal.057"),
        copyAttribute: 'data-send-feedback',
        copyHelp: text("core.portal.076"),
        copyLabel: text("core.portal.075"),
        id: 'project-discussions-panel',
        listAttribute: 'data-discussion-list',
        panelAttribute: 'data-discussions-panel',
        scrimAttribute: 'data-discussions-scrim',
        scrimLabel: text("changes.closeDiscussions"),
        summaryAttribute: 'data-review-summary',
        title: text("changes.discussions"),
    });
    const scrimMount: any = $('[data-discussions-scrim-mount]');
    const panelMount: any = $('[data-discussions-panel-mount]');
    if (scrimMount)
        scrimMount.replaceWith(scrim);
    else
        document.body.append(scrim);
    if (panelMount)
        panelMount.replaceWith(panel);
    else
        document.body.append(panel);
    const state: any = { report: null, repository: null, files: [], linked: [], linkedPaths: new Set(), selected: null, tab: 'source', merge: null, etag: '', repositoryEtag: '', reviewEtag: '', review: null, composerTarget: null, composerQuote: '', composerReturn: null, discussionsReturn: null, pendingDelete: '', activeDiscussion: '', discussionScroll: 0, detailRequest: 0 };
    const elements: any = {
        base: $('[data-base]'), branchBase: $('[data-branch-base]'), target: $('[data-target]'), targetRevision: $('[data-target-revision]'), targetRevisionWrap: $('[data-target-revision-wrap]'), apply: $('[data-apply-range]'), range: $('[data-range-summary]'), rangeMeta: $('[data-range-meta]'),
        summary: $('[data-summary]'), rangeDetails: $('[data-range-details]'), stale: $('[data-stale]'), search: $('[data-search]'), status: $('[data-status]'), scope: $('[data-scope]'), list: $('[data-file-list]'),
        count: $('[data-result-count]'), detail: $('[data-detail]'), toast: $('[data-changes-toast]'), toastMessage: $('[data-toast-message]'),
        discussions: $('[data-discussions-panel]'), discussionsScrim: $('[data-discussions-scrim]'), discussionList: $('[data-discussion-list]'), openDiscussionCount: $('[data-open-discussion-count]'), sendFeedback: $('[data-send-feedback]'), reviewSummary: $('[data-review-summary]'),
        composer: $('[data-review-composer]'), composerForm: $('[data-review-form]'), composerMessage: $('[data-review-message]'), composerIntent: $('[data-review-intent]'), composerError: $('[data-review-error]'), composerTarget: $('[data-review-target-summary]'), deleteConfirm: $('[data-review-delete-confirm]'), deleteForm: $('[data-review-delete-form]'),
        filesPanel: $('[data-files-panel]'), filePicker: $('[data-file-picker]'), filePickerQuery: $('[data-file-picker-query]'), filePickerResults: $('[data-file-picker-results]'), openFileStale: $('[data-open-file-stale]'),
    };
    if (!REVIEW)
        document.querySelectorAll('[data-discussions-toggle]').forEach((element: any) => element.hidden = true);
    document.querySelectorAll('[data-global-comment], [data-linked-file-open]').forEach((element: any) => element.hidden = true);
    if (!API) {
        elements.detail.innerHTML = `<div class="changes-error" data-ui-state="capability-unavailable">${escapeHTML(text("changes.viewerUnavailable"))}</div>`;
        return;
    }
    document.addEventListener('toudocu:themechange', async (event: any) => {
        state.merge?.setTheme?.(event.detail.theme);
        if (state.selected && (state.tab === 'mermaid' || state.tab === 'rendered'))
            await renderDetail();
    });
    const statusLabel: any = (status: any) => ({ added: text("features.changes.index.002"), untracked: text("features.changes.index.153"), modified: text("features.changes.index.003"), deleted: text("features.changes.index.004"), renamed: text("features.changes.index.005"), copied: text("features.changes.index.006"), 'type-changed': text("features.changes.index.007"), linked: text("features.changes.index.091") } as Record<string, string>)[status] || status;
    const outcomeLabel: any = (outcome: any) => ({ answered: text("core.portal.067"), changed: text("core.portal.090"), no_change: text("core.portal.068"), needs_clarification: text("core.portal.069"), failed: text("core.portal.091") } as Record<string, string>)[outcome] || outcome;
    const placementLabel: any = (status: any) => ({ current: text("features.changes.index.145"), moved: text("features.changes.index.146"), stale: text("features.changes.index.147"), deleted: text("features.changes.index.148") } as Record<string, string>)[status] || status;
    const selectedTarget: any = () => elements.target.value === 'revision' ? elements.targetRevision.value.trim() : elements.target.value;
    const query: any = () => {
        const params: any = new URLSearchParams();
        if (elements.base.value.trim() && elements.base.value.trim() !== 'HEAD')
            params.set('base', elements.base.value.trim());
        if (elements.branchBase.value.trim())
            params.set('branchBase', elements.branchBase.value.trim());
        const target: any = selectedTarget();
        if (target && target !== 'working-tree')
            params.set('target', target);
        return params;
    };
    const apiURL: any = (endpoint: any = '', extra: any = {}) => { const params: any = query(); Object.entries(extra).forEach(([key, value]: any) => value != null && params.set(key, value)); return `${API}${endpoint}?${params}`; };
    const reviewURL: any = (endpoint: any = '', extra: any = {}) => { const params: any = query(); Object.entries(extra).forEach(([key, value]: any) => value != null && params.set(key, value)); return `${REVIEW}${endpoint}?${params}`; };
    const repositoryReviewURL: any = (endpoint: any = '', extra: any = {}) => { const params: any = query(); Object.entries(extra).forEach(([key, value]: any) => value != null && params.set(key, value)); return `${REPOSITORY_REVIEW}${endpoint}?${params}`; };
    const languageFor: any = (path: any) => path.endsWith('.json') ? 'json' : /\.ya?ml$/i.test(path) ? 'yaml' : path.endsWith('.go') ? 'go' : path.endsWith('.java') ? 'java' : /\.(js|jsx|mjs|cjs)$/i.test(path) ? 'javascript' : /\.(ts|tsx|mts|cts)$/i.test(path) ? 'typescript' : /\.md$/i.test(path) ? 'markdown' : 'text';
    const changeText: any = (change: any) => [change.path, change.oldPath].join(' ').toLocaleLowerCase(locale);
    const isDocumentationFile: any = (change: any) => !!change.documentation || change.path === 'CHANGELOG.md' || change.oldPath === 'CHANGELOG.md';
    const feedbackWritable: any = () => !!REVIEW && selectedTarget() === 'working-tree' && state.repository?.feedbackWritable !== false;
    function normalizedReviewFile(file: any) {
        const documentation: any = file.documentation || {};
        return {
            status: file.status || 'linked', path: file.path, oldPath: file.oldPath || '', gitState: file.gitState || {}, lines: file.lines || { added: 0, deleted: 0 }, binary: !!file.binary,
            oldSize: documentation.oldSize || 0, newSize: file.size || documentation.newSize || 0, classification: documentation.classification || 'repository-file',
            entitiesBefore: documentation.entitiesBefore || [], entitiesAfter: documentation.entitiesAfter || [], sourceDiffAvailable: file.status !== 'linked', renderedDiffAvailable: !!documentation.renderedDiffAvailable,
            semanticDiffAvailable: !!documentation.semanticDiffAvailable, sourceDiff: documentation.sourceDiff || '', sourceDiffHunks: documentation.sourceDiffHunks || [], renderedSections: documentation.renderedSections || [],
            mermaidBlocks: documentation.mermaidBlocks || [], semanticChanges: documentation.semanticChanges || [], relationChanges: documentation.relationChanges || [], diagnostics: documentation.diagnostics || [],
            asset: documentation.asset, screen: documentation.screen, language: file.language || languageFor(file.path), documentation: file.documentation || null, _reviewFile: file,
        };
    }
    function reconcileLinkedFiles() {
        const changedPaths: any = new Set(state.files.flatMap((file: any) => [file.path, file.oldPath].filter(Boolean)));
        for (const discussion of state.review?.session?.discussions || []) {
            const path: any = discussion.target?.path;
            if (path)
                state.linkedPaths.add(path);
        }
        state.linked = [...state.linkedPaths]
            .filter((path: any) => !changedPaths.has(path))
            .map((path: any) => normalizedReviewFile({ status: 'linked', path, language: languageFor(path), gitState: {}, lines: { added: 0, deleted: 0 } }));
    }
    function updateURL() {
        const params: any = query();
        if (state.selected)
            params.set('path', state.selected.path);
        if (state.tab !== 'source')
            params.set('tab', state.tab);
        if (elements.search.value)
            params.set('q', elements.search.value);
        if (elements.status.value)
            params.set('status', elements.status.value);
        if (elements.scope.value)
            params.set('scope', elements.scope.value);
        history.replaceState(null, '', `${location.pathname}?${params}`);
    }
    function renderSummary() {
        const report: any = state.repository || state.report;
        const summary: any = report.summary;
        elements.range.textContent = `${report.comparison.base.displayRef} → ${report.comparison.target.displayRef}`;
        elements.rangeMeta.textContent = text("features.changes.index.149", [report.comparison.base.resolved.slice(0, 7), report.comparison.target.resolved ? report.comparison.target.resolved.slice(0, 7) : report.comparison.target.displayRef, report.repository.branch || text("features.changes.index.150"), text(report.repository.dirty ? "features.changes.index.151" : "features.changes.index.152")]);
        elements.summary.textContent = text("features.changes.index.131", [state.files.length, summary.lines.added, summary.lines.deleted]);
    }
    function matches(change: any) {
        const search: any = elements.search.value.trim().toLocaleLowerCase(locale);
        if (search && !changeText(change).includes(search))
            return false;
        if (elements.status.value && change.status !== elements.status.value)
            return false;
        if (elements.scope.value === 'documents' && !isDocumentationFile(change))
            return false;
        if (elements.scope.value === 'other' && isDocumentationFile(change))
            return false;
        return true;
    }
    function renderList() {
        const changed: any = state.files.filter(matches).sort((left: any, right: any) => left.path.localeCompare(right.path, locale));
        const linked: any = state.linked.filter(matches).sort((left: any, right: any) => left.path.localeCompare(right.path, locale));
        elements.count.textContent = text("features.changes.index.013", [changed.length + linked.length, state.files.length + state.linked.length]);
        elements.list.replaceChildren();
        const appendSection: any = (label: any, items: any) => {
            if (!items.length)
                return;
            const heading: any = document.createElement('h3');
            heading.textContent = label;
            elements.list.append(heading);
            const tree: any = { directories: new Map(), files: [] };
            items.forEach((change: any) => {
                const parts: any = change.path.split('/');
                let directory: any = tree;
                parts.slice(0, -1).forEach((part: any) => {
                    if (!directory.directories.has(part))
                        directory.directories.set(part, { directories: new Map(), files: [] });
                    directory = directory.directories.get(part);
                });
                directory.files.push(change);
            });
            const appendTree: any = (node: any, parent: any) => {
                [...node.directories].sort(([left]: any, [right]: any) => left.localeCompare(right, locale)).forEach(([name, child]: any) => {
                    const folder: any = document.createElement('details');
                    folder.className = 'changes-tree-folder';
                    folder.open = true;
                    const summary: any = document.createElement('summary');
                    summary.innerHTML = `<span aria-hidden="true"></span><strong>${escapeHTML(name)}</strong>`;
                    folder.append(summary);
                    appendTree(child, folder);
                    parent.append(folder);
                });
                node.files.forEach((change: any) => {
                const button: any = document.createElement('button');
                button.type = 'button';
                button.className = `changes-file ${state.selected?.path === change.path ? 'is-active' : ''}`;
                button.dataset.path = change.path;
                button.title = change.path;
                const discussionCount: any = discussionsForPath(change.path).filter((item: any) => item.state === 'open').length;
                const filename: any = change.path.split('/').pop();
                const statusCode: any = ({ added: 'A', untracked: 'U', modified: 'M', deleted: 'D', renamed: 'R', copied: 'C', 'type-changed': 'T', linked: 'L' } as Record<string, string>)[change.status] || '•';
                button.innerHTML = `<span class="changes-file-icon" aria-hidden="true"></span><strong>${escapeHTML(filename)}${discussionCount ? ` <span class="review-file-badge" aria-label="${escapeHTML(text("features.changes.index.092", [discussionCount]))}">${discussionCount}</span>` : ''}</strong><span class="changes-line-stats">+${change.lines.added} −${change.lines.deleted}</span><span class="changes-file-status status-${escapeHTML(change.status)}" aria-label="${escapeHTML(statusLabel(change.status))}" title="${escapeHTML(statusLabel(change.status))}">${statusCode}</span>`;
                button.addEventListener('click', () => selectChange(change));
                    parent.append(button);
                });
            };
            const root: any = document.createElement('div');
            root.className = 'changes-file-tree';
            appendTree(tree, root);
            elements.list.append(root);
        };
        appendSection(text("features.changes.index.093"), changed);
        appendSection(text("features.changes.index.094"), linked);
        if (!changed.length && !linked.length)
            elements.list.innerHTML = `<div class="changes-list-empty"><strong>${escapeHTML(text("changes.noMatches"))}</strong><p>${escapeHTML(text("changes.resetFilters"))}</p></div>`;
        return [...changed, ...linked];
    }
    const tabsFor: any = (change: any) => {
        const documentation: any = !REVIEW || !!change.documentation;
        return [['source', text("features.changes.index.022")], ...(!change.binary && !change.asset ? [['file', text("features.changes.index.133")]] : []), ...(documentation && change.renderedDiffAvailable ? [['rendered', text("features.changes.index.023")]] : []), ...(documentation && change.semanticDiffAvailable ? [['semantic', text("features.changes.index.024")]] : []), ...(documentation ? [['relations', text("features.changes.index.025")]] : []), ...(change.classification === 'contract' ? [['openapi', 'OpenAPI']] : []), ...(change.mermaidBlocks?.length ? [['mermaid', 'Mermaid']] : []), ...(change.classification === 'asset' ? [['assets', text("features.changes.index.154")]] : []), ...([...(change.entitiesBefore || []), ...(change.entitiesAfter || [])].some((item: any) => item.type === 'screen' || item.type === 'transition') ? [['map', text("features.changes.index.026")]] : [])];
    };
    async function selectChange(change: any, tab: any = state.tab) {
        const request: any = ++state.detailRequest;
        state.selected = change;
        tab = tab === 'summary' ? 'source' : tab;
        state.tab = tabsFor(change).some(([id]: any) => id === tab) ? tab : 'source';
        state.merge?.destroy?.();
        state.merge = null;
        elements.filesPanel.classList.remove('is-open');
        $('[data-mobile-files]')?.setAttribute('aria-expanded', 'false');
        renderList();
        updateURL();
        if (REVIEW && !change._reviewDetail) {
            elements.detail.innerHTML = detailHeader(change);
            elements.detail.setAttribute('aria-busy', 'true');
            elements.detail.querySelectorAll('[data-tab]').forEach((button: any) => button.disabled = true);
            $('[data-tab-panel]', elements.detail).innerHTML = `<div class="changes-loading" data-ui-state="loading" role="status">${escapeHTML(text("changes.loadingFile"))}</div>`;
            try {
                const response: any = await fetch(repositoryReviewURL('/repository/file', { path: change.path }), { cache: 'no-store' });
                const detail: any = await response.json();
                if (!response.ok)
                    throw new Error(detail.diagnostics?.[0]?.message || `HTTP ${response.status}`);
                if (request !== state.detailRequest || state.selected !== change)
                    return;
                Object.assign(change, { sourceDiff: detail.patch || '', sourceDiffHunks: detail.hunks || [], sourceDiffAvailable: !!detail.patch, _before: detail.before || '', _current: detail.current || '', _repositoryRevision: detail.repositoryRevision, _reviewDetail: detail });
                if ('renderedBefore' in detail)
                    change._renderedBefore = detail.renderedBefore || '';
                if ('renderedCurrent' in detail)
                    change._renderedCurrent = detail.renderedCurrent || '';
            }
            catch (error: any) {
                if (request !== state.detailRequest || state.selected !== change)
                    return;
                elements.detail.removeAttribute('aria-busy');
                $('[data-tab-panel]', elements.detail).innerHTML = `<div class="changes-error">${escapeHTML(text("changes.loadFileFailed", [error.message]))}</div>`;
                announce(error.message);
                return;
            }
        }
        if (request !== state.detailRequest || state.selected !== change)
            return;
        elements.detail.removeAttribute('aria-busy');
        await renderDetail();
    }
    function detailHeader(change: any) {
        const editorLink: any = EDITOR_WORKSPACE && change.path.startsWith('docs/') ? `<a class="changes-button secondary" href="${escapeHTML(EDITOR_WORKSPACE)}?path=${encodeURIComponent(change.path.replace(/^docs\//, ''))}">${escapeHTML(text("changes.editCurrentFile"))}</a>` : '';
        const comment: any = feedbackWritable() && change.status !== 'linked' ? `<button type="button" class="changes-button secondary" data-file-comment>${text("features.changes.index.095")}</button>` : '';
        const diagnostics: any = change.diagnostics?.length ? `<details class="changes-diagnostics" ${change.diagnostics.some((item: any) => item.severity === 'error') ? 'open' : ''}><summary>${text("features.changes.index.155")} · ${change.diagnostics.length}</summary><ul>${change.diagnostics.map((item: any) => `<li class="is-${escapeHTML(item.severity)}"><code>${escapeHTML(item.code)}</code> ${escapeHTML(item.message)}</li>`).join('')}</ul></details>` : '';
        const tabs: any = tabsFor(change).map(([id, label]: any) => `<button type="button" role="tab" aria-selected="${state.tab === id}" data-tab="${id}">${escapeHTML(label)}</button>`).join('');
        return `<header class="changes-detail-header"><div><span class="changes-file-status status-${escapeHTML(change.status)}">${escapeHTML(statusLabel(change.status))}</span><h2>${escapeHTML(change.path.split('/').pop())}</h2><p>${escapeHTML(change.oldPath ? `${change.oldPath} → ${change.path}` : change.path)} · +${change.lines.added} −${change.lines.deleted}</p></div><div class="changes-detail-actions">${comment}${editorLink}</div></header>${diagnostics}<nav class="changes-tabs" role="tablist" aria-label="${escapeHTML(text("changes.changeViews"))}">${tabs}</nav><div class="changes-tab-panel" data-tab-panel></div>`;
    }
    async function renderDetail() {
        const change: any = state.selected;
        if (!change)
            return;
        elements.detail.innerHTML = detailHeader(change);
        elements.detail.querySelectorAll('[data-tab]').forEach((button: any) => button.addEventListener('click', () => selectChange(change, button.dataset.tab)));
        elements.detail.querySelector('[data-file-comment]')?.addEventListener('click', (event: any) => openComposer({ kind: 'file', path: change.path }, event.currentTarget));
        const panel: any = $('[data-tab-panel]', elements.detail);
        if (state.tab === 'source')
            await renderSource(panel, change);
        if (state.tab === 'file')
            await renderFile(panel, change);
        if (state.tab === 'rendered')
            await renderBeforeAfter(panel, change);
        if (state.tab === 'semantic')
            renderSemantic(panel, change);
        if (state.tab === 'relations')
            renderRelations(panel, change);
        if (state.tab === 'openapi')
            renderSemantic(panel, change);
        if (state.tab === 'mermaid')
            await renderMermaid(panel, change);
        if (state.tab === 'assets')
            renderAssets(panel, change);
        if (state.tab === 'map')
            renderMap(panel, change);
    }
    async function fetchSide(change: any, side: any, render: any = false) {
        const cacheKey: any = render ? (side === 'before' ? '_renderedBefore' : '_renderedCurrent') : (side === 'before' ? '_before' : '_current');
        if (cacheKey in change)
            return change[cacheKey] || '';
        const response: any = await fetch(apiURL(render ? '/render' : '/content', { side, path: change.path }), { cache: 'no-store' });
        if (response.status === 204) {
            change[cacheKey] = '';
            return change[cacheKey];
        }
        if (!response.ok)
            throw new Error(`HTTP ${response.status}`);
        change[cacheKey] = await response.text();
        return change[cacheKey];
    }
    function createSelectionMenu(panel: any) {
        const menu: any = document.createElement('div');
        menu.className = 'review-selection-menu';
        menu.hidden = true;
        menu.setAttribute('role', 'toolbar');
        menu.setAttribute('aria-label', text("features.changes.index.137"));
        menu.innerHTML = `<button type="button" data-selection-copy>${text("core.portal.035")}</button><button type="button" data-selection-context>${text("core.portal.036")}</button><button type="button" data-selection-question>${text("core.portal.037")}</button>`;
        menu.addEventListener('mousedown', (event: any) => event.preventDefault());
        panel.append(menu);
        return menu;
    }
    function positionSelectionMenu(menu: any, rectangle: any) {
        menu.style.visibility = 'hidden';
        menu.hidden = false;
        requestAnimationFrame(() => {
            if (menu.hidden)
                return;
            const bounds: any = menu.getBoundingClientRect();
            const gap: any = 8;
            const left: any = Math.min(innerWidth - bounds.width - gap, Math.max(gap, rectangle.left + rectangle.width / 2 - bounds.width / 2));
            const above: any = rectangle.top - bounds.height - gap;
            const top: any = Math.min(innerHeight - bounds.height - gap, Math.max(gap, above >= gap ? above : rectangle.bottom + gap));
            menu.style.left = `${left}px`;
            menu.style.top = `${top}px`;
            menu.style.visibility = '';
        });
    }
    function oldSelectionQuote(change: any, startLine: any, endLine: any, selectedText: any) {
        const path: any = change.oldPath || change.path;
        const lines: any = startLine === endLine ? `${startLine}` : `${startLine}–${endLine}`;
        return `> ${path}:${lines}\n> ${selectedText.replace(/\n/g, '\n> ')}`;
    }
    function unifiedSelection(change: any, host: any) {
        const selection: any = window.getSelection();
        if (!selection || selection.isCollapsed || !selection.rangeCount)
            return null;
        const range: any = selection.getRangeAt(0);
        const nodeElement: any = (node: any) => node.nodeType === Node.TEXT_NODE ? node.parentElement : node;
        const startRow: any = nodeElement(range.startContainer)?.closest?.('[data-review-line]');
        const endRow: any = nodeElement(range.endContainer)?.closest?.('[data-review-line]');
        if (!startRow || !endRow || !host.contains(startRow) || !host.contains(endRow))
            return null;
        const rows: any[] = [...host.querySelectorAll('[data-review-line]')];
        const startIndex: any = rows.indexOf(startRow);
        const endIndex: any = rows.indexOf(endRow);
        if (startIndex < 0 || endIndex < startIndex)
            return null;
        const selectedRows: any[] = rows.slice(startIndex, endIndex + 1);
        const textOffset: any = (content: any, node: any, offset: any, fallback: any) => {
            if (!content.contains(node))
                return fallback;
            const prefix: any = document.createRange();
            prefix.selectNodeContents(content);
            prefix.setEnd(node, offset);
            return Array.from(prefix.toString()).length;
        };
        const parts: any[] = selectedRows.map((row: any, index: any) => {
            const content: any = row.querySelector('.diff-line-content');
            const characters: any[] = Array.from(content?.textContent || '');
            const start: any = index === 0 ? textOffset(content, range.startContainer, range.startOffset, 0) : 0;
            const end: any = index === selectedRows.length - 1 ? textOffset(content, range.endContainer, range.endOffset, characters.length) : characters.length;
            return characters.slice(start, end).join('');
        });
        const selectedText: any = parts.join('\n');
        if (!selectedText.trim())
            return null;
        const sides: any = new Set(selectedRows.map((row: any) => row.dataset.reviewSide));
        const side: any = sides.size === 1 ? selectedRows[0].dataset.reviewSide : 'mixed';
        const firstLine: any = Number(startRow.dataset.reviewLine);
        const lastLine: any = Number(endRow.dataset.reviewLine);
        const startColumn: any = textOffset(startRow.querySelector('.diff-line-content'), range.startContainer, range.startOffset, 0) + 1;
        const endColumn: any = textOffset(endRow.querySelector('.diff-line-content'), range.endContainer, range.endOffset, Array.from(endRow.querySelector('.diff-line-content')?.textContent || '').length) + 1;
        let target: any = null;
        if (feedbackWritable() && change.status !== 'deleted' && side === 'new')
            target = { type: 'diff', path: change.path, side, start: { line: firstLine, column: startColumn }, end: { line: lastLine, column: endColumn } };
        if (feedbackWritable() && side === 'old')
            target = { kind: 'file', path: change.path, quote: oldSelectionQuote(change, firstLine, lastLine, selectedText) };
        const rectangles: any = range.getClientRects();
        return { text: selectedText, target, returnElement: startRow.querySelector('.diff-line-content'), rectangle: rectangles[rectangles.length - 1] || range.getBoundingClientRect() };
    }
    async function renderSource(panel: any, change: any) {
        const hunkNavigation: any = change.sourceDiffHunks?.length > 1 ? `<button type="button" class="changes-button secondary" data-hunk-previous>${escapeHTML(text("changes.previousHunk"))}</button><button type="button" class="changes-button secondary" data-hunk-next>${escapeHTML(text("changes.nextHunk"))}</button>` : '';
        panel.innerHTML = `<div class="source-actions"><div class="source-mode-toggle" role="group" aria-label="${escapeHTML(text("changes.comparisonMode"))}"><button type="button" data-source-mode="unified" aria-pressed="true">${escapeHTML(text("changes.unified"))}</button><button type="button" data-source-mode="merge" aria-pressed="false">${escapeHTML(text("changes.sideBySide"))}</button></div><button type="button" class="changes-button tertiary" data-copy-diff>${escapeHTML(text("changes.copyDiff"))}</button>${hunkNavigation}</div><div data-source-view></div>`;
        const host: any = $('[data-source-view]', panel);
        const menu: any = createSelectionMenu(panel);
        let pendingSelection: any = null;
        let unifiedController: any = null;
        let activeHunk: any = 0;
        const hideMenu: any = () => { menu.hidden = true; pendingSelection = null; };
        $('[data-selection-copy]', menu).addEventListener('click', async () => {
            if (!pendingSelection)
                return;
            try { await navigator.clipboard.writeText(pendingSelection.text); announce(text("core.portal.039")); hideMenu(); }
            catch { announce(text("core.portal.041")); }
        });
        $('[data-selection-context]', menu).addEventListener('click', async () => {
            if (!pendingSelection)
                return;
            const filename: any = change.path.split('/').pop();
            try { await navigator.clipboard.writeText(text("core.portal.038", [filename, change.path, pendingSelection.text])); announce(text("core.portal.040")); hideMenu(); }
            catch { announce(text("core.portal.041")); }
        });
        $('[data-selection-question]', menu).addEventListener('click', () => {
            if (!pendingSelection?.target)
                return;
            const pending: any = pendingSelection;
            hideMenu();
            openComposer(pending.target, pending.returnElement);
        });
        const setMode: any = (mode: any) => panel.querySelectorAll('[data-source-mode]').forEach((button: any) => button.setAttribute('aria-pressed', String(button.dataset.sourceMode === mode)));
        const renderHunkLine: any = (line: any, counters: any) => {
            let oldLine: any = '', newLine: any = '';
            const marker: any = line[0] || ' ';
            if (![' ', '-', '+'].includes(marker))
                return `<span class="diff-line diff-line-context"><span></span><span class="diff-line-number"></span><span class="diff-line-number"></span><span class="diff-line-marker">${escapeHTML(marker)}</span><span class="diff-line-content">${escapeHTML(line.slice(1))}</span></span>`;
            if (marker === ' ') {
                oldLine = counters.old++;
                newLine = counters.new++;
            }
            if (marker === '-')
                oldLine = counters.old++;
            if (marker === '+')
                newLine = counters.new++;
            const side: any = marker === '-' ? 'old' : 'new';
            const selectedLine: any = marker === '-' ? oldLine : newLine;
            const comment: any = feedbackWritable() ? `<button type="button" class="diff-comment" aria-label="${escapeHTML(text(side === 'old' ? "features.changes.index.096" : "features.changes.index.097", [selectedLine]))}">+</button>` : '<span></span>';
            return `<span class="diff-line diff-line-${marker === '+' ? 'added' : marker === '-' ? 'removed' : 'context'}" data-review-side="${side}" data-review-line="${selectedLine}">${comment}<span class="diff-line-number">${oldLine}</span><span class="diff-line-number">${newLine}</span><span class="diff-line-marker">${escapeHTML(marker)}</span><span class="diff-line-content">${escapeHTML(line.slice(1))}</span></span>`;
        };
        const focusHunk: any = (index: any) => {
            const hunks: any = [...host.querySelectorAll('.changes-hunk')];
            if (!hunks.length)
                return;
            activeHunk = (index + hunks.length) % hunks.length;
            hunks[activeHunk].scrollIntoView({ block: 'start', behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
            hunks[activeHunk].focus();
            location.hash = hunks[activeHunk].id;
        };
        const showUnified: any = () => {
            setMode('unified');
            hideMenu();
            unifiedController?.abort();
            unifiedController = new AbortController();
            state.merge?.destroy?.();
            state.merge = null;
            host.replaceChildren();
            if (!change.sourceDiffHunks?.length) {
                if (change._current !== undefined && window.ToudocuCodeMirror?.createViewer) {
                    state.merge = window.ToudocuCodeMirror.createViewer({ parent: host, doc: change._current || '', language: change.language || languageFor(change.path), onSelect: (selection: any) => selection && openComposer({ type: 'fileRange', path: change.path, start: selection.start, end: selection.end }, host) });
                }
                else {
                    const pre: any = document.createElement('pre');
                    pre.className = 'changes-diff';
                    pre.textContent = change.sourceDiff || text("features.changes.index.031");
                    host.append(pre);
                }
                return;
            }
            change.sourceDiffHunks.forEach((hunk: any, index: any) => {
                const article: any = document.createElement('article');
                article.className = 'changes-hunk';
                article.id = hunk.id;
                article.tabIndex = -1;
                const lines: any = hunk.patch.split('\n');
                const header: any = lines.shift();
                if (lines.at(-1) === '')
                    lines.pop();
                const counters: any = { old: hunk.oldStart, new: hunk.newStart };
                article.innerHTML = `<header><a href="#${escapeHTML(hunk.id)}" aria-label="${escapeHTML(text("changes.hunkLink", [index + 1]))}">${escapeHTML(header)}</a><span>−${hunk.oldStart},${hunk.oldLines} +${hunk.newStart},${hunk.newLines}</span><button type="button" class="changes-button secondary" data-copy-hunk>${escapeHTML(text("changes.copyHunk"))}</button></header><pre>${lines.map((line: any) => renderHunkLine(line, counters)).join('')}</pre>`;
                article.querySelector('a').addEventListener('click', (event: any) => { event.preventDefault(); activeHunk = index; history.replaceState(null, '', `${location.pathname}${location.search}#${hunk.id}`); article.focus(); });
                article.querySelector('[data-copy-hunk]').addEventListener('click', async () => { await navigator.clipboard.writeText(hunk.patch); announce(text("features.changes.index.033")); });
                host.append(article);
            });
            host.querySelectorAll('[data-review-line] .diff-comment').forEach((button: any) => button.addEventListener('click', () => {
                const row: any = button.closest('[data-review-line]');
                const content: any = row.querySelector('.diff-line-content').textContent || '';
                const target: any = row.dataset.reviewSide === 'old'
                    ? { kind: 'file', path: change.path, quote: oldSelectionQuote(change, Number(row.dataset.reviewLine), Number(row.dataset.reviewLine), content) }
                    : { type: 'diff', path: change.path, side: 'new', start: { line: Number(row.dataset.reviewLine), column: 1 }, end: { line: Number(row.dataset.reviewLine), column: Array.from(content).length + 1 } };
                openComposer(target, button);
            }));
            const showSelectionMenu: any = (focus = false) => {
                pendingSelection = unifiedSelection(change, host);
                if (!pendingSelection)
                    return hideMenu();
                $('[data-selection-question]', menu).disabled = !pendingSelection.target;
                positionSelectionMenu(menu, pendingSelection.rectangle);
                if (focus)
                    requestAnimationFrame(() => $('[data-selection-copy]', menu).focus());
            };
            host.addEventListener('pointerup', () => showSelectionMenu(), { signal: unifiedController.signal });
            host.addEventListener('keyup', (event: any) => { if (event.key === 'Shift' || !event.shiftKey) showSelectionMenu(true); }, { signal: unifiedController.signal });
            document.addEventListener('pointerdown', (event: any) => { if (!menu.contains(event.target)) hideMenu(); }, { signal: unifiedController.signal });
            document.addEventListener('scroll', hideMenu, { capture: true, signal: unifiedController.signal });
            window.addEventListener('resize', hideMenu, { signal: unifiedController.signal });
            document.addEventListener('keydown', (event: any) => { if (event.key === 'Escape' && !menu.hidden) { event.preventDefault(); event.stopPropagation(); hideMenu(); host.focus(); } }, { capture: true, signal: unifiedController.signal });
            const requested: any = decodeURIComponent(location.hash.slice(1));
            const requestedIndex: any = change.sourceDiffHunks.findIndex((hunk: any) => hunk.id === requested);
            if (requestedIndex >= 0)
                requestAnimationFrame(() => focusHunk(requestedIndex));
        };
        const showMerge: any = async () => {
            setMode('merge');
            hideMenu();
            unifiedController?.abort();
            host.innerHTML = `<div class="changes-loading">${escapeHTML(text("changes.loadingVersions"))}</div>`;
            try {
                const [before, after]: any = await Promise.all([fetchSide(change, 'before'), fetchSide(change, 'after')]);
                host.replaceChildren();
                if (window.ToudocuCodeMirror?.createMerge)
                    state.merge = window.ToudocuCodeMirror.createMerge({ parent: host, before, after, language: change.language || languageFor(change.path), onSelect: (selection: any) => selection && openComposer({ type: 'diff', path: change.path, side: selection.side, start: selection.start, end: selection.end }, host) });
                else {
                    host.innerHTML = `<div class="source-columns"><pre>${escapeHTML(before)}</pre><pre>${escapeHTML(after)}</pre></div>`;
                }
            }
            catch (error: any) {
                host.innerHTML = `<div class="changes-error">${escapeHTML(text("changes.loadVersionsFailed", [error.message]))}</div>`;
            }
        };
        panel.querySelector('[data-source-mode="unified"]').addEventListener('click', showUnified);
        panel.querySelector('[data-source-mode="merge"]').addEventListener('click', showMerge);
        panel.querySelector('[data-hunk-previous]')?.addEventListener('click', () => focusHunk(activeHunk - 1));
        panel.querySelector('[data-hunk-next]')?.addEventListener('click', () => focusHunk(activeHunk + 1));
        panel.querySelector('[data-copy-diff]').addEventListener('click', async () => { await navigator.clipboard.writeText(change.sourceDiff || ''); announce(text("features.changes.index.036")); });
        showUnified();
    }
    async function renderFile(panel: any, change: any) {
        panel.innerHTML = `<div class="changes-loading" data-ui-state="loading" role="status">${escapeHTML(text("changes.loadingFile"))}</div>`;
        try {
            const deleted: any = change.status === 'deleted';
            const content: any = await fetchSide(change, deleted ? 'before' : 'after');
            panel.innerHTML = `${deleted ? `<p class="changes-absence">${escapeHTML(text("changes.showingDeletedVersion"))}</p>` : ''}<div data-file-view></div>`;
            const host: any = $('[data-file-view]', panel);
            if (window.ToudocuCodeMirror?.createViewer) {
                const menu: any = createSelectionMenu(panel);
                $('[data-selection-question]', menu).disabled = !feedbackWritable();
                const controller: any = new AbortController();
                let pendingSelection: any = null;
                let pendingText: any = '';
                let selecting: any = false;
                const hideMenu: any = () => { menu.hidden = true; pendingSelection = null; pendingText = ''; };
                const showMenu: any = () => {
                    const selected: any = window.getSelection();
                    if (!pendingSelection || !selected || selected.isCollapsed || !selected.rangeCount || !host.contains(selected.anchorNode))
                        return hideMenu();
                    const rectangles: any = selected.getRangeAt(0).getClientRects();
                    const rectangle: any = rectangles[rectangles.length - 1] || selected.getRangeAt(0).getBoundingClientRect();
                    pendingText = selected.toString();
                    positionSelectionMenu(menu, rectangle);
                };
                const viewer: any = window.ToudocuCodeMirror.createViewer({ parent: host, doc: content, language: change.language || languageFor(change.path), onSelect: (selection: any) => { pendingSelection = selection; if (!selecting) showMenu(); } });
                state.merge = { ...viewer, destroy() { controller.abort(); menu.remove(); viewer.destroy(); } };
                host.addEventListener('pointerdown', () => { selecting = true; hideMenu(); }, { signal: controller.signal });
                document.addEventListener('pointerup', () => { if (selecting) { selecting = false; showMenu(); } }, { signal: controller.signal });
                document.addEventListener('pointercancel', () => { selecting = false; hideMenu(); }, { signal: controller.signal });
                host.addEventListener('keyup', (event: any) => { if (event.key === 'Shift' || !event.shiftKey) { showMenu(); requestAnimationFrame(() => $('[data-selection-copy]', menu).focus()); } }, { signal: controller.signal });
                document.addEventListener('pointerdown', (event: any) => { if (!menu.contains(event.target)) hideMenu(); }, { signal: controller.signal });
                document.addEventListener('scroll', hideMenu, { capture: true, signal: controller.signal });
                window.addEventListener('resize', hideMenu, { signal: controller.signal });
                document.addEventListener('keydown', (event: any) => { if (event.key === 'Escape' && !menu.hidden) { event.preventDefault(); event.stopPropagation(); hideMenu(); viewer.focus(); } }, { capture: true, signal: controller.signal });
                $('[data-selection-copy]', menu).addEventListener('click', async () => {
                    try {
                        await navigator.clipboard.writeText(pendingText);
                        announce(text("core.portal.039"));
                        hideMenu();
                        viewer.focus();
                    }
                    catch { announce(text("core.portal.041")); }
                });
                $('[data-selection-context]', menu).addEventListener('click', async () => {
                    try {
                        await navigator.clipboard.writeText(text("core.portal.038", [change.path.split('/').pop(), change.path, pendingText]));
                        announce(text("core.portal.040"));
                        hideMenu();
                        viewer.focus();
                    }
                    catch { announce(text("core.portal.041")); }
                });
                $('[data-selection-question]', menu)?.addEventListener('click', () => {
                    const selection: any = pendingSelection;
                    const selectedText: any = pendingText;
                    const returnElement: any = $('.cm-content', host);
                    hideMenu();
                    const target: any = deleted
                        ? { kind: 'file', path: change.path, quote: oldSelectionQuote(change, selection.start.line, selection.end.line, selectedText) }
                        : { type: 'fileRange', path: change.path, start: selection.start, end: selection.end };
                    openComposer(target, returnElement);
                });
            }
            else {
                const pre: any = document.createElement('pre');
                pre.className = 'changes-diff';
                pre.textContent = content;
                host.append(pre);
            }
        }
        catch (error: any) {
            panel.innerHTML = `<div class="changes-error">${escapeHTML(text("changes.loadFileFailed", [error.message]))}</div>`;
        }
    }
    async function renderBeforeAfter(panel: any, change: any) {
        panel.innerHTML = `<div class="changes-loading">${escapeHTML(text("changes.renderingVersions"))}</div>`;
        try {
            const [before, after]: any = await Promise.all([fetchSide(change, 'before', true), fetchSide(change, 'after', true)]);
            const beforeDocument: any = before || `<p class="changes-absence">${escapeHTML(text("changes.documentMissing"))}</p>`;
            const afterDocument: any = after || `<p class="changes-absence">${escapeHTML(text("changes.documentDeleted"))}</p>`;
            panel.innerHTML = `<div class="rendered-columns"><section><h3>${escapeHTML(text("changes.before"))}</h3><div class="rendered-document">${beforeDocument}</div></section><section><h3>${escapeHTML(text("changes.after"))}</h3><div class="rendered-document">${afterDocument}</div></section></div>`;
            const documents: any = panel.querySelectorAll('.rendered-document');
            const sectionElements: any = (root: any, anchor: any) => {
                const heading: any = anchor && root?.querySelector(`#${CSS.escape(anchor)}`);
                const content: any[] = [];
                for (let sibling: any = heading?.nextElementSibling; sibling && sibling.tagName !== 'H2'; sibling = sibling.nextElementSibling)
                    content.push(sibling);
                return { heading, content };
            };
            const unmatchedBlocks: any = (source: any[], target: any[]) => {
                const remaining: any = new Map();
                target.forEach((element: any) => remaining.set(element.outerHTML, (remaining.get(element.outerHTML) || 0) + 1));
                return source.filter((element: any) => {
                    const count: any = remaining.get(element.outerHTML) || 0;
                    if (!count)
                        return true;
                    remaining.set(element.outerHTML, count - 1);
                    return false;
                });
            };
            const mark: any = (elements: any[], status: any) => elements.forEach((element: any) => element?.classList.add('rendered-section-content', status));
            (change.renderedSections || []).forEach((section: any) => {
                if (section.status === 'unchanged-section')
                    return;
                const before: any = sectionElements(documents[0], section.anchorBefore);
                const after: any = sectionElements(documents[1], section.anchorAfter);
                [before.heading, after.heading].forEach((heading: any) => {
                    if (!heading)
                        return;
                    heading.classList.add('rendered-section-heading', section.status);
                    heading.dataset.changeLabel = section.status.replace('-section', '').replace('-', ' ');
                });
                if (section.status === 'modified-section') {
                    const beforeChanged: any[] = unmatchedBlocks(before.content, after.content);
                    const afterChanged: any[] = unmatchedBlocks(after.content, before.content);
                    mark(beforeChanged, section.status);
                    mark(afterChanged, section.status);
                }
                else {
                    mark(before.content, section.status);
                    mark(after.content, section.status);
                }
            });
            if (window.mermaid) {
                window.mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: document.documentElement.dataset.theme === 'dark' ? 'dark' : 'default' });
                for (const diagram of panel.querySelectorAll('.mermaid')) {
                    try {
                        await window.mermaid.run({ nodes: [diagram] });
                    }
                    catch {
                        diagram.insertAdjacentHTML('afterend', `<p class="changes-error">${escapeHTML(text("changes.mermaidSideFailed"))}</p>`);
                    }
                }
            }
        }
        catch (error: any) {
            panel.innerHTML = `<div class="changes-error">${escapeHTML(text("changes.renderedDiffFailed", [error.message]))}</div>`;
        }
    }
    async function renderMermaid(panel: any, change: any) {
        const blocks: any = change.mermaidBlocks || [];
        if (!blocks.length) {
            panel.innerHTML = `<div class="changes-empty"><h3>${escapeHTML(text("changes.noMermaidChanges"))}</h3><p>${escapeHTML(text("changes.noMermaidChangesHelp"))}</p></div>`;
            return;
        }
        panel.innerHTML = blocks.map((block: any) => `<section class="mermaid-change"><header><strong>${escapeHTML(block.caption || block.id)}</strong><span class="changes-file-status status-${escapeHTML(block.status)}">${escapeHTML(statusLabel(block.status))}</span></header><div class="mermaid-controls" role="group" aria-label="${escapeHTML(text("changes.diagramControls"))}"><button type="button" class="changes-button secondary" data-mermaid-zoom="out">−</button><button type="button" class="changes-button secondary" data-mermaid-zoom="reset">100%</button><button type="button" class="changes-button secondary" data-mermaid-zoom="in">+</button><button type="button" class="changes-button secondary" data-mermaid-fullscreen>${escapeHTML(text("changes.fullscreen"))}</button></div><div class="rendered-columns"><section><h3>${escapeHTML(text("changes.diagramBefore"))}</h3>${block.before ? `<div class="mermaid-canvas" data-mermaid-canvas><pre class="mermaid">${escapeHTML(block.before)}</pre></div>` : `<p class="changes-absence">${escapeHTML(text("changes.diagramMissing"))}</p>`}</section><section><h3>${escapeHTML(text("changes.diagramAfter"))}</h3>${block.after ? `<div class="mermaid-canvas" data-mermaid-canvas><pre class="mermaid">${escapeHTML(block.after)}</pre></div>` : `<p class="changes-absence">${escapeHTML(text("changes.diagramDeleted"))}</p>`}</section></div><h3>${escapeHTML(text("changes.mermaidSourceDiff"))}</h3><pre class="changes-diff mermaid-source-diff">${mermaidSourceDiff(block.before || '', block.after || '')}</pre></section>`).join('');
        if (!window.mermaid)
            return;
        window.mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: document.documentElement.dataset.theme === 'dark' ? 'dark' : 'default' });
        for (const diagram of panel.querySelectorAll('.mermaid')) {
            try {
                await window.mermaid.run({ nodes: [diagram] });
            }
            catch {
                diagram.insertAdjacentHTML('afterend', `<p class="changes-error">${escapeHTML(text("changes.mermaidVersionFailed"))}</p>`);
            }
        }
        panel.querySelectorAll('.mermaid-change').forEach((section: any) => setupMermaidControls(section));
    }
    function mermaidSourceDiff(before: any, after: any) {
        const oldLines: any = before ? before.split('\n') : [];
        const newLines: any = after ? after.split('\n') : [];
        const rows: any = Array.from({ length: oldLines.length + 1 }, () => Array(newLines.length + 1).fill(0));
        for (let i: any = oldLines.length - 1; i >= 0; i--)
            for (let j: any = newLines.length - 1; j >= 0; j--)
                rows[i][j] = oldLines[i] === newLines[j] ? rows[i + 1][j + 1] + 1 : Math.max(rows[i + 1][j], rows[i][j + 1]);
        const result: any = [];
        let i: any = 0;
        let j: any = 0;
        while (i < oldLines.length || j < newLines.length) {
            if (i < oldLines.length && j < newLines.length && oldLines[i] === newLines[j]) {
                result.push(`  ${oldLines[i++]}`);
                j++;
            }
            else if (j < newLines.length && (i === oldLines.length || rows[i][j + 1] >= rows[i + 1][j]))
                result.push(`+ ${newLines[j++]}`);
            else
                result.push(`- ${oldLines[i++]}`);
        }
        return escapeHTML(result.join('\n') || text("features.changes.index.044"));
    }
    function setupMermaidControls(section: any) {
        let zoom: any = 1;
        let x: any = 0;
        let y: any = 0;
        let pointer: any = null;
        const canvases: any = [...section.querySelectorAll('[data-mermaid-canvas]')];
        const apply: any = () => canvases.forEach((canvas: any) => { canvas.style.setProperty('--mermaid-zoom', zoom); canvas.style.setProperty('--mermaid-x', `${x}px`); canvas.style.setProperty('--mermaid-y', `${y}px`); });
        section.querySelectorAll('[data-mermaid-zoom]').forEach((button: any) => button.addEventListener('click', () => {
            const action: any = button.dataset.mermaidZoom;
            zoom = action === 'reset' ? 1 : Math.max(.5, Math.min(2.5, zoom + (action === 'in' ? .2 : -.2)));
            if (action === 'reset') {
                x = 0;
                y = 0;
            }
            apply();
        }));
        section.querySelector('[data-mermaid-fullscreen]')?.addEventListener('click', () => {
            if (document.fullscreenElement)
                document.exitFullscreen?.();
            else
                section.requestFullscreen?.();
        });
        canvases.forEach((canvas: any) => {
            canvas.addEventListener('pointerdown', (event: any) => { pointer = { id: event.pointerId, x: event.clientX, y: event.clientY, startX: x, startY: y }; canvas.setPointerCapture?.(event.pointerId); canvas.classList.add('is-panning'); });
            canvas.addEventListener('pointermove', (event: any) => {
                if (!pointer || pointer.id !== event.pointerId)
                    return;
                x = pointer.startX + event.clientX - pointer.x;
                y = pointer.startY + event.clientY - pointer.y;
                apply();
            });
            canvas.addEventListener('pointerup', () => { pointer = null; canvas.classList.remove('is-panning'); });
        });
    }
    function renderSemantic(panel: any, change: any) {
        panel.innerHTML = change.semanticChanges.length ? `<ol class="semantic-list">${change.semanticChanges.map((item: any) => `<li><div><strong>${escapeHTML(item.kind)}</strong>${item.compatibility ? `<span class="compatibility ${item.compatibility}">${escapeHTML(item.compatibility)}</span>` : ''}</div><p>${escapeHTML(item.summary)}</p>${item.before !== undefined || item.after !== undefined ? `<div class="semantic-values"><pre>${escapeHTML(JSON.stringify(item.before, null, 2) || '—')}</pre><pre>${escapeHTML(JSON.stringify(item.after, null, 2) || '—')}</pre></div>` : ''}</li>`).join('')}</ol>` : `<div class="changes-empty"><h3>${escapeHTML(text("changes.noSemanticChanges"))}</h3><p>${escapeHTML(text("changes.noSemanticChangesHelp"))}</p></div>`;
    }
    function renderRelations(panel: any, change: any) { panel.innerHTML = change.relationChanges.length ? `<ul>${change.relationChanges.map((item: any) => `<li>${escapeHTML(item.kind)}: ${escapeHTML(item.source.id)} → ${escapeHTML(item.target.id)}</li>`).join('')}</ul>` : `<div class="changes-empty"><h3>${escapeHTML(text("changes.noRelationChanges"))}</h3><p>${escapeHTML(text("changes.noRelationChangesHelp"))}</p></div>`; }
    function renderAssets(panel: any, change: any) {
        const before: any = apiURL('/content', { side: 'before', path: change.path });
        const after: any = apiURL('/content', { side: 'after', path: change.path });
        const meta: any = (side: any, bytes: any) => side ? text("changes.assetMeta", [side.width || '?', side.height || '?', side.aspectRatio || '?', bytes, side.transparency == null ? '' : ` · ${text(side.transparency ? "changes.alpha" : "changes.opaque")}`]) : text("changes.assetBytes", [bytes]);
        const beforePresent: any = change.status !== 'added' && change.status !== 'untracked';
        const afterPresent: any = change.status !== 'deleted';
        const overlay: any = beforePresent && afterPresent && change.asset?.before?.mediaType !== 'image/svg+xml' && change.asset?.after?.mediaType !== 'image/svg+xml';
        const beforeAsset: any = beforePresent ? `<img src="${escapeHTML(before)}" alt="${escapeHTML(text("changes.oldVersion", [change.path]))}">` : `<p class="changes-absence">${escapeHTML(text("changes.assetMissing"))}</p>`;
        const afterAsset: any = afterPresent ? `<img src="${escapeHTML(after)}" alt="${escapeHTML(text("changes.newVersion", [change.path]))}">` : `<p class="changes-absence">${escapeHTML(text("changes.assetDeleted"))}</p>`;
        const overlayView: any = overlay ? `<section class="asset-overlay"><h3>${escapeHTML(text("changes.overlay"))}</h3><div class="asset-overlay-stage"><img src="${escapeHTML(before)}" alt="${escapeHTML(text("changes.oldVersionShort"))}"><div data-overlay-after><img src="${escapeHTML(after)}" alt="${escapeHTML(text("changes.newVersionShort"))}"></div><span data-overlay-divider></span></div><label><span>${escapeHTML(text("changes.dividerPosition"))}</span><input type="range" min="0" max="100" value="50" data-overlay-range></label></section>` : '';
        panel.innerHTML = `<div class="rendered-columns asset-columns"><section><h3>${escapeHTML(text("changes.before"))} · ${escapeHTML(meta(change.asset?.before, change.oldSize))}</h3>${beforeAsset}</section><section><h3>${escapeHTML(text("changes.after"))} · ${escapeHTML(meta(change.asset?.after, change.newSize))}</h3>${afterAsset}</section></div>${overlayView}`;
        const range: any = panel.querySelector('[data-overlay-range]');
        if (range)
            range.addEventListener('input', () => { const value: any = `${range.value}%`; panel.querySelector('[data-overlay-after]').style.clipPath = `inset(0 0 0 ${value})`; panel.querySelector('[data-overlay-divider]').style.left = value; });
    }
    function renderMap(panel: any, change: any) {
        const screen: any = change.screen;
        if (!screen) {
            panel.innerHTML = `<div class="changes-empty"><h3>${escapeHTML(text("changes.screenDiffUnavailable"))}</h3><p>${escapeHTML(text("changes.screenDiffUnavailableHelp"))}</p></div>`;
            return;
        }
        const before: any = screen.before;
        const after: any = screen.after;
        const node: any = after || before;
        const edge: any = (transition: any) => { const oldValue: any = transition.before; const newValue: any = transition.after; return `<li class="map-change-edge status-${escapeHTML(transition.status)}"><header><strong>${escapeHTML(transition.id)}</strong><span>${escapeHTML(statusLabel(transition.status))}</span></header><div class="map-edge-values"><span>${oldValue ? `${escapeHTML(oldValue.source)} → ${escapeHTML(oldValue.target)}` : text("features.changes.index.049")}</span><span>${newValue ? `${escapeHTML(newValue.source)} → ${escapeHTML(newValue.target)}` : text("features.changes.index.050")}</span></div>${newValue?.action || oldValue?.action ? `<p>${escapeHTML(newValue?.action || oldValue?.action)} · ${escapeHTML(newValue?.condition || oldValue?.condition || '')}</p>` : ''}</li>`; };
        const transitions: any = screen.transitions.length ? `<ol class="map-change-edges">${screen.transitions.map(edge).join('')}</ol>` : `<p>${escapeHTML(text("changes.transitionsUnchanged"))}</p>`;
        panel.innerHTML = `<div class="map-change-preview"><h3>${escapeHTML(text("changes.screenMapChanges"))}</h3><div class="map-change-node status-${escapeHTML(change.status)} ${after ? '' : 'is-ghost'}"><strong>${escapeHTML(node?.id)}</strong><span>${escapeHTML(statusLabel(change.status))}</span><small>${escapeHTML(node?.title || '')}${node?.route ? ` · ${escapeHTML(node.route)}` : ''}</small></div><h4>${escapeHTML(text("changes.transitions"))}</h4>${transitions}<p><a href="${escapeHTML(apiURL('/screen-map'))}">${escapeHTML(text("changes.openChangedMapJSON"))}</a></p></div>`;
    }
    function discussionsForPath(path: any) {
        return state.review?.session?.discussions?.filter((discussion: any) => discussion.target?.path === path || discussion.placement?.path === path) || [];
    }
    function discussionInFlightClient(discussionId: any) {
        return (state.review?.deliveries || []).some((delivery: any) => delivery.discussionId === discussionId && delivery.state !== 'responded');
    }
    function messageEditableClient(message: any) {
        return message.author === 'human' && (message.state === 'draft' || (state.review?.deliveries || []).some((delivery: any) => delivery.id === message.deliveryId && delivery.state === 'pending'));
    }
    function reviewGuard() { return { expectedRevision: state.review?.revision || 0, expectedStateDigest: state.review?.stateDigest || '' }; }
    async function reviewMutation(endpoint: any, action: any, method: any, body: any) {
        const send: any = async (payload: any) => {
            const response: any = await fetch(reviewURL(endpoint), { method, headers: { 'Content-Type': 'application/json', 'X-Toudocu-Action': action }, body: JSON.stringify(payload) });
            return { response, data: await response.json() };
        };
        let result: any = await send(body);
        if (!result.response.ok && result.data.diagnostics?.[0]?.code === 'AGENT_REVISION_CONFLICT') {
            await loadReview();
            result = await send({ ...body, ...reviewGuard() });
        }
        if (!result.response.ok)
            throw new Error(result.data.diagnostics?.[0]?.message || `HTTP ${result.response.status}`);
        return result.data;
    }
    function openDiscussions(returnElement: any = document.activeElement, focus = true) {
        clearTimeout(state.discussionsTimer);
        state.discussionsReturn = returnElement;
        elements.discussions.hidden = false;
        elements.discussionsScrim.hidden = false;
        elements.discussions.setAttribute('role', 'complementary');
        elements.discussions.removeAttribute('aria-modal');
        if (matchMedia('(max-width: 560px)').matches) {
            elements.discussions.setAttribute('role', 'dialog');
            elements.discussions.setAttribute('aria-modal', 'true');
        }
        $('[data-discussions-toggle]')?.setAttribute('aria-expanded', 'true');
        requestAnimationFrame(() => elements.discussions.classList.add('is-open'));
        if (focus)
            $('[data-discussions-close]')?.focus();
    }
    function closeDiscussions() {
        elements.discussions.classList.remove('is-open');
        elements.discussionsScrim.hidden = true;
        elements.discussions.setAttribute('role', 'complementary');
        elements.discussions.removeAttribute('aria-modal');
        $('[data-discussions-toggle]')?.setAttribute('aria-expanded', 'false');
        state.discussionsTimer = setTimeout(() => { elements.discussions.hidden = true; }, 180);
        state.discussionsReturn?.focus?.();
    }
    function openComposer(target: any, returnElement: any, mode: any = { operation: 'create' }) {
        if (!feedbackWritable()) {
            announce(text("features.changes.index.098"));
            return;
        }
        if (!target.path || mode.operation === 'create' && !state.selected)
            return;
        const normalized: any = { kind: target.kind || 'file', path: target.path };
        if (target.start && !(target.type === 'diff' && target.side === 'old'))
            normalized.range = { start: target.start, end: target.end };
        state.composerTarget = normalized;
        state.composerQuote = mode.operation === 'create' ? target.quote || '' : '';
        state.composerReturn = returnElement || document.activeElement;
        state.composerMode = mode;
        elements.composerMessage.value = mode.message || '';
        elements.composerIntent.value = mode.intent || 'question';
        elements.composerError.textContent = '';
        elements.composerTarget.textContent = `${normalized.path}${normalized.range ? ` · ${normalized.range.start.line}:${normalized.range.start.column}–${normalized.range.end.line}:${normalized.range.end.column}` : ''}`;
        elements.composer.showModal();
        requestAnimationFrame(() => elements.composerMessage.focus());
    }
    async function submitComposer() {
        const mode: any = state.composerMode || { operation: 'create' };
        const message: any = elements.composerMessage.value.trim();
        const intent: any = elements.composerIntent.value;
        if (!message)
            return;
        if (mode.operation === 'create' && !state.composerTarget) {
            elements.composerError.textContent = text("features.changes.index.101");
            return;
        }
        try {
            let updated: any;
            if (mode.operation === 'create') {
                const savedText: any = state.composerQuote ? `${message}\n\n${state.composerQuote}` : message;
                updated = await reviewMutation('/discussions', 'agent-discussion-create', 'POST', { ...reviewGuard(), target: state.composerTarget, intent, text: savedText });
            }
            else if (mode.operation === 'reply') {
                updated = await reviewMutation(`/discussions/${mode.discussionId}/messages`, 'agent-message-create', 'POST', { ...reviewGuard(), intent, text: message });
            }
            else {
                updated = await reviewMutation(`/discussions/${mode.discussionId}/messages/${mode.messageId}`, 'agent-message-update', 'PATCH', { ...reviewGuard(), intent, text: message });
            }
            state.review = updated;
            elements.composer.close();
            openDiscussions(state.composerReturn, false);
            renderReview();
            renderList();
            announce(text(mode.operation === 'create' ? "features.changes.index.102" : "features.changes.index.103"));
        }
        catch (error: any) { elements.composerError.textContent = error.message; }
    }
    async function updateDiscussion(discussionId: any, operation: any, extra: any = {}) {
        try {
            if (operation === 'deleteDiscussion')
                state.review = await reviewMutation(`/discussions/${discussionId}`, 'agent-discussion-delete', 'DELETE', reviewGuard());
            else if (operation === 'deleteMessage')
                state.review = await reviewMutation(`/discussions/${discussionId}/messages/${extra.messageId}`, 'agent-message-delete', 'DELETE', reviewGuard());
            else
                state.review = await reviewMutation(`/discussions/${discussionId}`, 'agent-discussion-update', 'PATCH', { ...reviewGuard(), state: operation === 'resolve' ? 'resolved' : 'open' });
            renderReview();
            renderList();
            return true;
        }
        catch (error: any) { announce(error.message); return false; }
    }
    function renderReview() {
        if (!REVIEW || !state.review)
            return;
        const discussions: any = [...(state.review.session?.discussions || [])].sort((left: any, right: any) => Number(left.state !== 'open') - Number(right.state !== 'open'));
        const open: any = discussions.filter((discussion: any) => discussion.state === 'open');
        elements.openDiscussionCount.textContent = String(open.length);
        elements.reviewSummary.textContent = text("features.changes.index.104", [open.length, discussions.length - open.length]);
        state.discussionScroll = elements.discussionList.scrollTop;
        elements.discussionList.replaceChildren();
        discussions.forEach((discussion: any) => {
            const article: any = document.createElement('article');
            article.className = `portal-review-thread review-thread is-${discussion.state}${state.activeDiscussion === discussion.id ? ' is-active' : ''}`;
            article.dataset.discussionId = discussion.id;
            const placement: any = discussion.placement || {};
            article.innerHTML = `<header><div><strong>${text("features.changes.index.111")}</strong><span>${escapeHTML(text(discussion.state === 'open' ? "features.changes.index.105" : "features.changes.index.106"))} · ${escapeHTML(placementLabel(placement.status || 'current'))}</span></div><div class="portal-review-thread-actions review-thread-actions"><button type="button" data-thread-state>${text(discussion.state === 'open' ? "features.changes.index.107" : "features.changes.index.108")}</button><button type="button" class="is-danger" data-delete-discussion>${text("features.changes.index.142")}</button></div></header><button type="button" class="portal-review-anchor review-anchor" data-open-anchor>${escapeHTML(placement.path || discussion.target.path)}${placement.range ? `:${placement.range.start.line}` : ''}</button>${discussion.anchor?.selectedText ? `<blockquote class="portal-review-quote">${escapeHTML(discussion.anchor.selectedText)}</blockquote>` : ''}<ol>${discussion.messages.map((message: any) => `<li class="review-message is-${message.author}"><div><strong>${message.author === 'agent' ? `${text("features.changes.index.110")} · ${escapeHTML(outcomeLabel(message.outcome || 'answered'))}` : `${text("features.changes.index.111")} · ${text(message.intent === 'change_request' ? "core.portal.094" : "core.portal.093")}`}</strong><time>${escapeHTML(new Date(message.createdAt).toLocaleString(locale))}</time></div><p>${escapeHTML(message.text)}</p>${message.changedPaths?.length ? `<button type="button" data-view-fix="${escapeHTML(message.id)}">${text("features.changes.index.112")}</button>` : ''}${messageEditableClient(message) ? `<div class="portal-review-thread-actions review-message-actions"><button type="button" data-edit-message="${escapeHTML(message.id)}">${text("features.changes.index.113")}</button><button type="button" class="is-danger" data-delete-message="${escapeHTML(message.id)}">${text("features.changes.index.114")}</button></div>` : ''}</li>`).join('')}</ol><button type="button" class="portal-review-reply review-reply" ${discussion.state !== 'open' || discussionInFlightClient(discussion.id) || discussion.messages.some((message: any) => message.state === 'draft') ? 'disabled' : ''}>${text("features.changes.index.115")}</button>`;
            article.querySelector('[data-thread-state]').addEventListener('click', () => updateDiscussion(discussion.id, discussion.state === 'open' ? 'resolve' : 'reopen'));
            article.querySelector('[data-delete-discussion]').addEventListener('click', () => {
                state.pendingDelete = discussion.id;
                elements.deleteConfirm.showModal();
            });
            article.querySelector('.review-reply').addEventListener('click', (event: any) => openComposer(discussion.target, event.currentTarget, { operation: 'reply', discussionId: discussion.id }));
            article.querySelector('[data-open-anchor]').addEventListener('click', () => openDiscussionAnchor(discussion));
            article.querySelectorAll('[data-edit-message]').forEach((button: any) => {
                const message: any = discussion.messages.find((item: any) => item.id === button.dataset.editMessage);
                button.addEventListener('click', () => openComposer(discussion.target, button, { operation: 'edit', discussionId: discussion.id, messageId: message.id, message: message.text, intent: message.intent }));
            });
            article.querySelectorAll('[data-delete-message]').forEach((button: any) => button.addEventListener('click', () => updateDiscussion(discussion.id, 'deleteMessage', { messageId: button.dataset.deleteMessage })));
            article.querySelectorAll('[data-view-fix]').forEach((button: any) => button.addEventListener('click', () => viewFix(discussion, discussion.messages.find((item: any) => item.id === button.dataset.viewFix))));
            article.addEventListener('focusin', () => state.activeDiscussion = discussion.id);
            elements.discussionList.append(article);
        });
        if (!discussions.length)
            elements.discussionList.innerHTML = `<div class="portal-review-empty"><p>${escapeHTML(text("changes.noDiscussions"))}</p></div>`;
        elements.discussionList.scrollTop = state.discussionScroll;
    }
    async function openDiscussionAnchor(discussion: any) {
        state.activeDiscussion = discussion.id;
        const path: any = discussion.placement?.path || discussion.target.path;
        const file: any = [...state.files, ...state.linked].find((item: any) => item.path === path || item.oldPath === path);
        if (file)
            await selectChange(file, 'source');
        highlightPlacement(discussion.placement);
        if (discussion.placement?.status === 'stale' || discussion.placement?.status === 'deleted')
            announce(placementLabel(discussion.placement.status));
    }
    function highlightPlacement(placement: any) {
        const range: any = placement?.range;
        if (!range)
            return false;
        const side: any = 'new';
        if (state.merge?.highlight) {
            state.merge.highlight(side, range.start, range.end);
            return true;
        }
        const rows: any = [...elements.detail.querySelectorAll(`[data-review-side="${side}"][data-review-line]`)].filter((row: any) => {
            const line: any = Number(row.dataset.reviewLine);
            return line >= range.start.line && line <= range.end.line;
        });
        if (!rows.length)
            return false;
        rows.forEach((row: any) => row.classList.add('review-placement-highlight'));
        rows[0].scrollIntoView({ block: 'center', behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
        setTimeout(() => rows.forEach((row: any) => row.classList.remove('review-placement-highlight')), 1800);
        return true;
    }
    async function viewFix(discussion: any, message: any) {
        const actual: any = new Set(state.files.map((file: any) => file.path));
        const preferred: any = discussion.placement?.path;
        const path: any = message.changedPaths.includes(preferred) && actual.has(preferred) ? preferred : message.changedPaths.find((item: any) => actual.has(item));
        if (!path) {
            await openDiscussionAnchor(discussion);
            announce(text("features.changes.index.118"));
            return;
        }
        const file: any = state.files.find((item: any) => item.path === path);
        await selectChange(file, 'source');
        if (!highlightPlacement(discussion.placement)) {
            elements.detail.classList.add('review-placement-highlight');
            setTimeout(() => elements.detail.classList.remove('review-placement-highlight'), 1800);
        }
    }
    async function loadReview() {
        if (!REVIEW || selectedTarget() !== 'working-tree') {
            state.review = null;
            state.reviewEtag = '';
            state.linked = [];
            document.querySelectorAll('[data-discussions-toggle]').forEach((element: any) => element.hidden = true);
            return;
        }
        document.querySelectorAll('[data-discussions-toggle]').forEach((element: any) => element.hidden = false);
        const response: any = await fetch(reviewURL('/discussions'), { cache: 'no-store' });
        const data: any = await response.json();
        if (!response.ok)
            throw new Error(data.diagnostics?.[0]?.message || `HTTP ${response.status}`);
        state.review = data;
        state.reviewEtag = response.headers.get('ETag') || '';
        reconcileLinkedFiles();
        renderReview();
        renderList();
    }
    async function loadFilePicker() {
        const response: any = await fetch(`${REPOSITORY_REVIEW}/repository/files?q=${encodeURIComponent(elements.filePickerQuery.value)}&limit=50`, { cache: 'no-store' });
        const data: any = await response.json();
        if (!response.ok)
            throw new Error(data.diagnostics?.[0]?.message || `HTTP ${response.status}`);
        elements.filePickerResults.innerHTML = data.files.map((file: any) => `<button type="button" data-link-path="${escapeHTML(file.path)}"><strong>${escapeHTML(file.path.split('/').pop())}</strong><span>${escapeHTML(file.path)}</span></button>`).join('') || `<p>${escapeHTML(text("changes.noMatches"))}</p>`;
        elements.filePickerResults.querySelectorAll('[data-link-path]').forEach((button: any) => button.addEventListener('click', async () => {
            const path: any = button.dataset.linkPath;
            state.linkedPaths.add(path);
            reconcileLinkedFiles();
            elements.filePicker.close();
            renderList();
            await selectChange([...state.files, ...state.linked].find((item: any) => item.path === path));
        }));
    }
    function announce(message: any) {
        elements.toastMessage.textContent = message;
        elements.toast.classList.add('is-visible');
        setTimeout(() => elements.toast.classList.remove('is-visible'), 2200);
    }
    function showEmptyDetail() {
        state.detailRequest++;
        state.selected = null;
        state.tab = 'source';
        state.merge?.destroy?.();
        state.merge = null;
        elements.detail.removeAttribute('aria-busy');
        elements.detail.innerHTML = `<div class="changes-empty" data-ui-state="empty"><h2>${escapeHTML(text("changes.none"))}</h2><p>${escapeHTML(text("changes.adjustFilters"))}</p></div>`;
        updateURL();
    }
    async function load(preserve: any = true) {
        const [response, repositoryResponse]: any = await Promise.all([fetch(apiURL(), { cache: 'no-store' }), REVIEW ? fetch(repositoryReviewURL('/repository/changes'), { cache: 'no-store' }) : Promise.resolve(null)]);
        const data: any = await response.json();
        if (!response.ok)
            throw new Error(data.diagnostics?.[0]?.message || `HTTP ${response.status}`);
        const repository: any = repositoryResponse ? await repositoryResponse.json() : null;
        if (repositoryResponse && !repositoryResponse.ok)
            throw new Error(repository.diagnostics?.[0]?.message || `HTTP ${repositoryResponse.status}`);
        const oldPath: any = preserve ? state.selected?.path : null;
        const oldScroll: any = preserve ? elements.detail.scrollTop : 0;
        state.report = data;
        state.repository = repository || data;
        state.files = repository ? repository.files.map(normalizedReviewFile) : data.changes;
        if (feedbackWritable())
            reconcileLinkedFiles();
        else
            state.linked = [];
        state.etag = response.headers.get('ETag') || '';
        state.repositoryEtag = repositoryResponse?.headers.get('ETag') || '';
        renderSummary();
        const visible: any = renderList();
        const params: any = new URLSearchParams(location.search);
        const requested: any = params.get('path') || oldPath;
        const selected: any = [...state.files, ...state.linked].find((change: any) => change.path === requested || change.oldPath === requested);
        if (selected)
            await selectChange(selected, params.get('tab') || (preserve ? state.tab : 'source'));
        else if (visible[0])
            await selectChange(visible[0], 'source');
        else
            showEmptyDetail();
        elements.detail.scrollTop = oldScroll;
    }
    async function init() {
        const params: any = new URLSearchParams(location.search);
        const target: any = params.get('target') || 'working-tree';
        elements.base.value = params.get('base') || 'HEAD';
        elements.branchBase.value = params.get('branchBase') || '';
        if (['working-tree', 'index', 'HEAD'].includes(target))
            elements.target.value = target;
        else {
            elements.target.value = 'revision';
            elements.targetRevision.value = target;
            elements.targetRevisionWrap.hidden = false;
        }
        elements.search.value = params.get('q') || '';
        elements.status.value = params.get('status') || '';
        elements.scope.value = params.get('scope') || '';
        const requestedPath: any = params.get('path');
        const requestedTab: any = params.get('tab') || 'source';
        const selectionBeforeLoad: any = state.detailRequest;
        [elements.search, elements.status, elements.scope].forEach((control: any) => control.addEventListener('input', async () => {
            const visible: any = renderList();
            if (!visible.some((change: any) => change.path === state.selected?.path)) {
                if (visible[0])
                    await selectChange(visible[0], 'source');
                else
                    showEmptyDetail();
            }
            else
                updateURL();
        }));
        elements.target.addEventListener('change', () => {
            elements.targetRevisionWrap.hidden = elements.target.value !== 'revision';
            if (elements.target.value === 'revision')
                elements.targetRevision.focus();
        });
        elements.apply.addEventListener('click', async () => {
            if (elements.target.value === 'revision' && !elements.targetRevision.value.trim()) {
                announce(text("features.changes.index.072"));
                return;
            }
            state.selected = null;
            elements.rangeDetails.open = false;
            elements.range.focus();
            updateURL();
            try {
                await Promise.all([load(false), loadReview()]);
            }
            catch (error: any) {
                announce(error.message);
            }
        });
        elements.composerForm.addEventListener('submit', (event: any) => {
            if (event.submitter?.value === 'cancel')
                return;
            event.preventDefault();
            submitComposer();
        });
        elements.composerMessage.addEventListener('keydown', (event: any) => {
            if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) { event.preventDefault(); submitComposer(); }
        });
        elements.composer.addEventListener('close', () => state.composerReturn?.focus?.());
        elements.deleteForm.addEventListener('submit', async (event: any) => {
            if (event.submitter?.value === 'cancel')
                return;
            event.preventDefault();
            event.submitter.disabled = true;
            const updated: any = await updateDiscussion(state.pendingDelete, 'deleteDiscussion');
            event.submitter.disabled = false;
            if (updated) {
                elements.deleteConfirm.close();
                announce(text("features.changes.index.144"));
            }
        });
        $('[data-discussions-toggle]')?.addEventListener('click', (event: any) => event.currentTarget.getAttribute('aria-expanded') === 'true' ? closeDiscussions() : openDiscussions(event.currentTarget));
        $('[data-discussions-close]')?.addEventListener('click', closeDiscussions);
        elements.discussionsScrim?.addEventListener('click', closeDiscussions);
        const closeFiles: any = () => { elements.filesPanel.classList.remove('is-open'); elements.filesPanel.setAttribute('role', 'navigation'); elements.filesPanel.removeAttribute('aria-modal'); $('[data-mobile-files]')?.setAttribute('aria-expanded', 'false'); $('[data-mobile-files]')?.focus(); };
        $('[data-mobile-files]')?.addEventListener('click', () => { elements.filesPanel.classList.add('is-open'); elements.filesPanel.setAttribute('role', 'dialog'); elements.filesPanel.setAttribute('aria-modal', 'true'); $('[data-mobile-files]')?.setAttribute('aria-expanded', 'true'); elements.filesPanel.querySelector('button')?.focus(); });
        $('[data-files-close]')?.addEventListener('click', closeFiles);
        const closeRange: any = () => { elements.rangeDetails.open = false; elements.range.focus(); };
        document.addEventListener('click', (event: any) => { if (elements.rangeDetails.open && !elements.rangeDetails.contains(event.target)) closeRange(); });
        document.addEventListener('keydown', (event: any) => { if (event.key !== 'Escape') return; if (elements.rangeDetails.open) closeRange(); else if (elements.discussions.classList.contains('is-open')) closeDiscussions(); else if (elements.filesPanel.classList.contains('is-open')) closeFiles(); });
        $('[data-linked-file-open]')?.addEventListener('click', async () => { elements.filePicker.showModal(); elements.filePickerQuery.focus(); await loadFilePicker(); });
        elements.filePickerQuery.addEventListener('input', () => loadFilePicker().catch((error: any) => announce(error.message)));
        elements.sendFeedback.addEventListener('click', async () => announce(await navigator.clipboard.writeText(text("features.changes.index.121")).then(() => text("core.portal.088"), () => text("core.portal.041"))));
        $('[data-refresh-open-file]')?.addEventListener('click', async () => { elements.openFileStale.hidden = true; await load(true); });
        try {
            await Promise.all([load(false), loadReview()]);
            const requested: any = [...state.files, ...state.linked].find((change: any) => change.path === requestedPath || change.oldPath === requestedPath);
            if (requested && state.detailRequest <= selectionBeforeLoad + 1 && (state.selected?.path !== requested.path || state.tab !== (requestedTab === 'summary' ? 'source' : requestedTab)))
                await selectChange(requested, requestedTab);
        }
        catch (error: any) {
            elements.detail.innerHTML = `<div class="changes-error"><h2>${escapeHTML(text("changes.unavailable"))}</h2><p>${escapeHTML(error.message)}</p><p>${escapeHTML(text("changes.unavailableHelp"))}</p></div>`;
        }
        const poll: any = async () => {
            try {
                if (document.hidden || elements.detail.hasAttribute('aria-busy'))
                    return;
                if (feedbackWritable()) {
                    const response: any = await fetch(repositoryReviewURL('/repository/changes'), { headers: state.repositoryEtag ? { 'If-None-Match': state.repositoryEtag } : {}, cache: 'no-store' });
                    if (response.status !== 304) {
                        const next: any = response.headers.get('ETag') || '';
                        if (state.repositoryEtag && next && next !== state.repositoryEtag) {
                            if (elements.composer.open && state.composerMode?.operation === 'create') {
                                state.composerTarget = null;
                                elements.composerError.textContent = text("features.changes.index.123");
                            }
                            if (state.selected)
                                elements.openFileStale.hidden = false;
                            else
                                await load(true);
                        }
                    }
                    const reviewResponse: any = await fetch(reviewURL('/discussions'), { headers: state.reviewEtag ? { 'If-None-Match': state.reviewEtag } : {}, cache: 'no-store' });
                    if (reviewResponse.status !== 304 && reviewResponse.ok) {
                        state.review = await reviewResponse.json();
                        state.reviewEtag = reviewResponse.headers.get('ETag') || '';
                        reconcileLinkedFiles();
                        renderReview();
                        renderList();
                    }
                }
                else {
                    const response: any = await fetch(apiURL(), { method: 'HEAD', cache: 'no-store' });
                    const next: any = response.headers.get('ETag') || '';
                    if (state.etag && next && next !== state.etag) {
                        elements.stale.hidden = false;
                        await load(true);
                        elements.stale.hidden = true;
                    }
                }
            }
            catch { /* current report remains readable */ }
            finally { setTimeout(poll, 2000); }
        };
        setTimeout(poll, 2000);
    }
    init();
})();
