import { text } from "./locale";
import { createEmptyState, selectTab, setExpanded } from "../components";
import { createDiscussionPanel } from "../components/discussion-panel";
(() => {
    'use strict';
    const $: any = (selector: any, root: any = document) => root.querySelector(selector);
    const $$: any = (selector: any, root: any = document) => [...root.querySelectorAll(selector)];
    const normalize: any = (value: any) => String(value || '')
        .toLocaleLowerCase(window.ToudocuPage?.ui.locale || 'en')
        .replace(/\u0451/g, '\u0435')
        .replace(/[^\p{L}\p{N}]+/gu, ' ')
        .trim();
    const rootPrefix: any = () => document.body.dataset.rootPrefix || '';
    const pageContract: any = () => window.ToudocuPage || null;
    const assetBase: any = () => pageContract()?.portal?.assetBase || `${rootPrefix()}assets/`;
    const dataBase: any = () => pageContract()?.portal?.dataBase || `${rootPrefix()}data/`;
    const scriptLoads: any = new Map();
    function loadScript(name: any) {
        if (scriptLoads.has(name))
            return scriptLoads.get(name);
        const promise: any = new Promise((resolve: any, reject: any) => {
            const script: any = document.createElement('script');
            script.src = new URL(name, new URL(assetBase(), window.location.href)).href;
            if (name !== 'mermaid.tiny.js')
                script.type = 'module';
            script.onload = resolve;
            script.onerror = () => reject(new Error(text("core.portal.002", [name])));
            document.head.append(script);
        }).catch((error: any) => {
            scriptLoads.delete(name);
            throw error;
        });
        scriptLoads.set(name, promise);
        return promise;
    }
    function initializeHeroSummary() {
        $$('[data-hero-summary]').forEach((summary: any) => {
            const text: any = $('p', summary);
            const button: any = $('[data-hero-summary-toggle]', summary);
            if (!text || !button)
                return;
            const fullHeight: any = text.scrollHeight;
            summary.classList.add('is-clampable');
            if (text.clientHeight + 1 >= fullHeight) {
                summary.classList.remove('is-clampable');
                return;
            }
            button.hidden = false;
            button.addEventListener('click', () => {
                const expanded: any = summary.classList.toggle('is-expanded');
                button.setAttribute('aria-expanded', String(expanded));
                button.textContent = expanded ? text("core.portal.003") : text("core.portal.004");
            });
        });
    }
    function mermaidThemeConfig() {
        const styles: any = getComputedStyle(document.documentElement);
        const color: any = (name: any) => styles.getPropertyValue(name).trim();
        const dark: any = document.documentElement.dataset.theme === 'dark';
        return {
            theme: 'base',
            themeVariables: {
                darkMode: dark,
                background: color('--bg'),
                primaryColor: color('--surface-soft'),
                primaryTextColor: color('--text'),
                primaryBorderColor: color('--accent'),
                lineColor: color('--border-strong'),
                secondaryColor: color('--surface-strong'),
                tertiaryColor: color('--surface'),
                noteBkgColor: color('--accent-soft'),
                noteTextColor: color('--text'),
                fontFamily: color('--font-body'),
            },
        };
    }
    function initializeSidebar(signal: any) {
        const toggle: any = $('[data-sidebar-toggle]');
        let folderState: any = {};
        try {
            const storedFolderState: any = JSON.parse(localStorage.getItem('project-docs-navigation') || '{}');
            if (storedFolderState && typeof storedFolderState === 'object' && !Array.isArray(storedFolderState)) {
                folderState = storedFolderState;
            }
        }
        catch { /* file:// privacy mode */ }
        $$('[data-nav-folder]').forEach((folder: any) => {
            const folderToggle: any = $('[data-nav-folder-toggle]', folder);
            const label: any = $('.nav-folder-row a > span:nth-of-type(2)', folder)?.textContent.trim() || $('.nav-folder-row a', folder)?.textContent.trim() || text("core.portal.005");
            const key: any = folder.dataset.navFolder;
            const setCollapsed: any = (collapsed: any, persist: any = true) => {
                folder.classList.toggle('is-collapsed', collapsed);
                if (folderToggle)
                    setExpanded(folderToggle, !collapsed);
                const taskLabel: any = folder.classList.contains('nav-task-folder') ? text(collapsed ? "nav.expandTask" : "nav.collapseTask", [label]) : text("core.portal.006", [collapsed ? text("core.portal.027") : text("core.portal.028"), label]);
                folderToggle?.setAttribute('aria-label', taskLabel);
                if (folderToggle)
                    folderToggle.title = folderToggle.getAttribute('aria-label');
                if (!persist)
                    return;
                folderState[key] = collapsed;
                try {
                    localStorage.setItem('project-docs-navigation', JSON.stringify(folderState));
                }
                catch { /* file:// privacy mode */ }
            };
            const containsActivePage: any = Boolean($('.is-active', folder));
            const hasSavedState: any = Object.prototype.hasOwnProperty.call(folderState, key);
            setCollapsed(containsActivePage ? false : (hasSavedState ? folderState[key] === true : true), false);
            folderToggle?.addEventListener('click', (event: any) => {
                event.stopPropagation();
                setCollapsed(!folder.classList.contains('is-collapsed'));
            }, { signal });
        });
        toggle?.addEventListener('click', (event: any) => {
            event.stopPropagation();
            const open: any = document.body.classList.toggle('sidebar-open');
            toggle.setAttribute('aria-expanded', String(open));
        }, { signal });
        document.addEventListener('click', (event: any) => {
            if (!document.body.classList.contains('sidebar-open'))
                return;
            if (event.target.closest('.sidebar') || event.target.closest('[data-sidebar-toggle]'))
                return;
            document.body.classList.remove('sidebar-open');
            toggle?.setAttribute('aria-expanded', 'false');
        }, { signal });
        $$('.sidebar a').forEach((link: any) => link.addEventListener('click', () => {
            document.body.classList.remove('sidebar-open');
            toggle?.setAttribute('aria-expanded', 'false');
        }, { signal }));
        $('.nav-link.is-active, .nav-folder-link.is-active')?.scrollIntoView({ block: 'center' });
    }
    function initializeGlobalSearch() {
        const input: any = $('[data-global-search]');
        const results: any = $('[data-search-results]');
        if (!input || !results)
            return;
        let selected: any = -1;
        let currentItems: any = [];
        let index: any = [];
        async function ensureIndex() {
            if (index.length)
                return index;
            const response: any = await fetch(new URL('search-index.json', new URL(dataBase(), window.location.href)), { cache: 'no-store' });
            if (!response.ok)
                throw new Error(`Search index unavailable: HTTP ${response.status}`);
            const payload: any = await response.json();
            index = Array.isArray(payload) ? payload : [];
            return index;
        }
        function score(item: any, query: any, terms: any) {
            const title: any = normalize(item.title);
            const path: any = normalize(item.path);
            const haystack: any = item.text || normalize(`${item.title} ${item.path} ${item.description} ${item.status}`);
            if (!terms.every((term: any) => haystack.includes(term)))
                return -1;
            let value: any = 0;
            if (title === query)
                value += 120;
            if (title.startsWith(query))
                value += 80;
            else if (title.includes(query))
                value += 50;
            if (path.includes(query))
                value += 24;
            for (const term of terms) {
                if (title.includes(term))
                    value += 15;
                if (path.includes(term))
                    value += 6;
            }
            return value;
        }
        function close() {
            results.hidden = true;
            results.replaceChildren();
            currentItems = [];
            selected = -1;
            input.setAttribute('aria-expanded', 'false');
        }
        function showUnavailable(error: any) {
            results.replaceChildren();
            const message: any = document.createElement('div');
            message.className = 'search-empty';
            message.dataset.uiState = 'static-json-unavailable';
            message.setAttribute('role', 'alert');
            message.textContent = text("core.portal.007", [error?.message || text("core.portal.029")]);
            results.append(message);
            results.hidden = false;
            input.setAttribute('aria-expanded', 'true');
        }
        function select(indexToSelect: any) {
            selected = Math.max(-1, Math.min(indexToSelect, currentItems.length - 1));
            $$('.search-result', results).forEach((element: any, itemIndex: any) => {
                element.classList.toggle('is-selected', itemIndex === selected);
                element.setAttribute('aria-selected', String(itemIndex === selected));
            });
            if (selected >= 0)
                $$('.search-result', results)[selected]?.scrollIntoView({ block: 'nearest' });
        }
        function render() {
            const query: any = normalize(input.value);
            results.replaceChildren();
            selected = -1;
            if (!query) {
                close();
                return;
            }
            const terms: any = query.split(' ').filter(Boolean);
            currentItems = index
                .map((item: any) => ({ item, score: score(item, query, terms) }))
                .filter((entry: any) => entry.score >= 0)
                .sort((a: any, b: any) => b.score - a.score || a.item.title.localeCompare(b.item.title, window.ToudocuPage?.ui.locale || 'en'))
                .slice(0, 12)
                .map((entry: any) => entry.item);
            if (!currentItems.length) {
                const empty: any = createEmptyState(text("core.portal.008"));
                empty.classList.add('search-empty');
                results.append(empty);
            }
            else {
                currentItems.forEach((item: any, itemIndex: any) => {
                    const link: any = document.createElement('a');
                    link.className = 'search-result';
                    link.href = `${rootPrefix()}${item.url}`;
                    link.setAttribute('role', 'option');
                    link.setAttribute('aria-selected', 'false');
                    link.dataset.searchIndex = String(itemIndex);
                    const title: any = document.createElement('span');
                    title.className = 'search-result-title';
                    title.textContent = item.title;
                    const meta: any = document.createElement('span');
                    meta.className = 'search-result-meta';
                    meta.textContent = [item.typeLabel, item.status, item.path].filter(Boolean).join(' · ');
                    link.append(title, meta);
                    results.append(link);
                });
            }
            results.hidden = false;
            input.setAttribute('aria-expanded', 'true');
        }
        input.addEventListener('input', async () => {
            try {
                await ensureIndex();
                render();
            }
            catch (error: any) {
                showUnavailable(error);
            }
        });
        input.addEventListener('focus', async () => {
            try {
                await ensureIndex();
                if (input.value.trim())
                    render();
            }
            catch (error: any) {
                showUnavailable(error);
            }
        });
        input.addEventListener('keydown', (event: any) => {
            if (event.key === 'ArrowDown') {
                event.preventDefault();
                select(selected + 1);
            }
            else if (event.key === 'ArrowUp') {
                event.preventDefault();
                select(selected <= 0 ? currentItems.length - 1 : selected - 1);
            }
            else if (event.key === 'Enter' && selected >= 0) {
                event.preventDefault();
                $$('.search-result', results)[selected]?.click();
            }
            else if (event.key === 'Escape') {
                close();
                input.blur();
            }
        });
        results.addEventListener('mousemove', (event: any) => {
            const link: any = event.target.closest('[data-search-index]');
            if (link)
                select(Number(link.dataset.searchIndex));
        });
        document.addEventListener('click', (event: any) => {
            if (!event.target.closest('.global-search'))
                close();
        });
        document.addEventListener('keydown', (event: any) => {
            if (event.key === '/' && !event.ctrlKey && !event.metaKey && !event.altKey && !/INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName || '')) {
                event.preventDefault();
                input.focus();
            }
        });
    }
    function initializeCollectionFilters() {
        $$('[data-filter-scope]').forEach((scope: any) => {
            const items: any = $$('[data-filter-item]', scope);
            const controls: any = $$('[data-filter-control]', scope);
            const resetButtons: any = $$('[data-filter-reset]', scope);
            const count: any = $('[data-filter-count]', scope);
            const empty: any = $('[data-filter-empty]', scope);
            if (!items.length || !controls.length)
                return;
            function apply() {
                const filters: any = {};
                controls.forEach((control: any) => {
                    filters[control.dataset.filterControl] = normalize(control.value);
                });
                let visible: any = 0;
                items.forEach((item: any) => {
                    const matches: any = Object.entries(filters).every(([key, value]: any) => {
                        if (!value || value === 'all')
                            return true;
                        const itemValue: any = normalize(item.dataset[key] || '');
                        if (key === 'search' || key === 'route')
                            return itemValue.includes(value);
                        if (key === 'usecase') {
                            return String(item.dataset[key] || '').split('|').map(normalize).includes(value);
                        }
                        return itemValue === value;
                    });
                    item.hidden = !matches;
                    if (matches)
                        visible += 1;
                });
                if (count)
                    count.textContent = text("core.portal.009", [visible, items.length]);
                if (empty)
                    empty.hidden = visible !== 0;
            }
            controls.forEach((control: any) => {
                control.addEventListener(control.tagName === 'INPUT' ? 'input' : 'change', apply);
            });
            resetButtons.forEach((button: any) => {
                button.addEventListener('click', () => {
                    controls.forEach((control: any) => {
                        control.value = control.dataset.filterDefault || (control.tagName === 'SELECT' ? 'all' : '');
                    });
                    apply();
                    controls[0]?.focus();
                });
            });
            apply();
        });
    }
    function initializeTaskFilters() {
        const buttons: any = $$('[data-task-filter]');
        if (!buttons.length)
            return;
        function setFilter(value: any) {
            document.body.dataset.taskFilter = value;
            buttons.forEach((button: any) => {
                const active: any = button.dataset.taskFilter === value;
                button.classList.toggle('is-active', active);
                button.setAttribute('aria-pressed', String(active));
            });
        }
        buttons.forEach((button: any) => button.addEventListener('click', () => setFilter(button.dataset.taskFilter)));
        setFilter('all');
    }
    function initializeCollapsibleSections() {
        $$('.doc-content').forEach((content: any) => {
            const children: any = [...content.children];
            const h2Indexes: any = children.map((node: any, index: any) => node.tagName === 'H2' ? index : -1).filter((index: any) => index >= 0);
            if (!h2Indexes.length)
                return;
            for (let position: any = h2Indexes.length - 1; position >= 0; position -= 1) {
                const start: any = h2Indexes[position];
                const end: any = h2Indexes[position + 1] ?? children.length;
                const heading: any = children[start];
                if (!heading.parentNode || heading.closest('.doc-section'))
                    continue;
                const section: any = document.createElement('section');
                section.className = 'doc-section';
                const body: any = document.createElement('div');
                body.className = 'section-body';
                heading.parentNode.insertBefore(section, heading);
                section.append(heading);
                for (let itemIndex: any = start + 1; itemIndex < end; itemIndex += 1) {
                    const node: any = children[itemIndex];
                    if (node.parentNode === content)
                        body.append(node);
                }
                section.append(body);
                const toggle: any = document.createElement('button');
                toggle.type = 'button';
                toggle.className = 'section-toggle';
                toggle.textContent = '▾';
                const headingTitle: any = heading.textContent.replace(/^#/, '').trim();
                section.dataset.sectionTitle = headingTitle;
                toggle.setAttribute('aria-label', text("core.portal.010", [headingTitle]));
                toggle.setAttribute('aria-expanded', 'true');
                toggle.addEventListener('click', () => {
                    const collapsed: any = section.classList.toggle('is-collapsed');
                    toggle.setAttribute('aria-expanded', String(!collapsed));
                    toggle.setAttribute('aria-label', text("core.portal.011", [collapsed ? text("core.portal.030") : text("core.portal.031"), headingTitle]));
                    updateCollapseAllButton();
                });
                section.insertBefore(toggle, body);
            }
        });
        const collapseAllButton: any = $('[data-collapse-all]');
        const sections: any = $$('.doc-section');
        function updateCollapseAllButton() {
            if (!collapseAllButton)
                return;
            const hasExpandedSections: any = sections.some((section: any) => !section.classList.contains('is-collapsed'));
            collapseAllButton.dataset.collapseState = hasExpandedSections ? 'expanded' : 'collapsed';
            collapseAllButton.setAttribute('aria-expanded', String(hasExpandedSections));
            const label: any = $('[data-collapse-label]', collapseAllButton);
            if (label)
                label.textContent = hasExpandedSections ? text("core.portal.012") : text("core.portal.013");
        }
        if (collapseAllButton && !sections.length)
            collapseAllButton.hidden = true;
        collapseAllButton?.addEventListener('click', () => {
            const shouldCollapse: any = sections.some((section: any) => !section.classList.contains('is-collapsed'));
            sections.forEach((section: any) => {
                section.classList.toggle('is-collapsed', shouldCollapse);
                const toggle: any = $('.section-toggle', section);
                if (!toggle)
                    return;
                toggle.setAttribute('aria-expanded', String(!shouldCollapse));
                const headingTitle: any = section.dataset.sectionTitle || '';
                toggle.setAttribute('aria-label', text("core.portal.014", [shouldCollapse ? text("core.portal.032") : text("core.portal.033"), headingTitle]));
            });
            updateCollapseAllButton();
        });
        updateCollapseAllButton();
    }
    async function copyText(value: any) {
        if (navigator.clipboard?.writeText) {
            try {
                await navigator.clipboard.writeText(value);
                return true;
            }
            catch { /* file:// and browser permission fallback */ }
        }
        const activeElement: any = document.activeElement;
        const textarea: any = document.createElement('textarea');
        textarea.value = value;
        textarea.setAttribute('readonly', '');
        textarea.style.position = 'fixed';
        textarea.style.inset = '0 auto auto -9999px';
        textarea.style.opacity = '0';
        document.body.append(textarea);
        textarea.focus();
        textarea.select();
        textarea.setSelectionRange(0, textarea.value.length);
        let copied: any = false;
        try {
            copied = typeof document.execCommand === 'function' && document.execCommand('copy');
        }
        catch { /* unsupported fallback */ }
        textarea.remove();
        activeElement?.focus?.();
        return copied;
    }
    function initializeDocumentContextCopy() {
        $$('[data-copy-document-context]').forEach((button: any) => {
            const label: any = $('[data-copy-document-context-label]', button);
            const title: any = button.dataset.documentContextTitle || '';
            const path: any = button.dataset.documentContextPath || '';
            let resetTimer: any = 0;
            button.addEventListener('click', async () => {
                window.clearTimeout(resetTimer);
                const copied: any = await copyText(text("core.portal.015", [title, path]));
                button.dataset.copyState = copied ? 'success' : 'error';
                if (label)
                    label.textContent = copied ? text("core.portal.016") : text("core.portal.017");
                resetTimer = window.setTimeout(() => {
                    delete button.dataset.copyState;
                    if (label)
                        label.textContent = text("core.portal.018");
                }, copied ? 1300 : 2200);
            });
        });
    }
    function initializeDocumentReview(signal: any) {
        const page: any = pageContract();
        const review: any = page?.runtime === 'serve' && page.capabilities?.review ? page.endpoints?.review : '';
        const toggle: any = $('[data-discussions-toggle]');
        const contextButton: any = $('[data-copy-document-context]');
        const selectionArea: any = contextButton?.closest('.page-content');
        if (!review || !toggle)
            return;
        const selectionSources: any[] = [
            selectionArea && $('.page-header h1', selectionArea),
            selectionArea && $('.metadata-grid', selectionArea),
            selectionArea && $('.page-header .page-lead', selectionArea),
            ...(selectionArea ? $$('.doc-content', selectionArea) : []),
        ].filter(Boolean);
        const title: any = contextButton?.dataset.documentContextTitle || '';
        const path: any = contextButton?.dataset.documentContextPath || '';
        const canCreate: any = Boolean(path);
        const escapeHTML: any = (value: any) => String(value ?? '').replace(/[&<>"']/g, (character: any) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' } as Record<string, string>)[character]);
        const placementLabel: any = (status: any) => ({ current: text("core.portal.063"), moved: text("core.portal.064"), stale: text("core.portal.065"), deleted: text("core.portal.066") } as Record<string, string>)[status] || status;
        const outcomeLabel: any = (outcome: any) => ({ answered: text("core.portal.067"), changed: text("core.portal.090"), no_change: text("core.portal.068"), needs_clarification: text("core.portal.069"), failed: text("core.portal.091") } as Record<string, string>)[outcome] || outcome;
        const menu: any = document.createElement('div');
        menu.className = 'review-selection-menu';
        menu.hidden = true;
        menu.setAttribute('role', 'toolbar');
        menu.setAttribute('aria-label', text("core.portal.034"));
        menu.innerHTML = `<button type="button" data-selection-copy>${text("core.portal.035")}</button><button type="button" data-selection-context>${text("core.portal.036")}</button><button type="button" data-selection-question>${text("core.portal.037")}</button>`;
        menu.addEventListener('mousedown', (event: any) => event.preventDefault());
        const dialog: any = document.createElement('dialog');
        dialog.className = 'portal-review-dialog';
        dialog.innerHTML = `<form method="dialog"><header><h2 data-portal-review-dialog-title>${text("core.portal.042")}</h2><span data-portal-review-title></span></header><label data-portal-review-selection-wrap>${text("core.portal.043")}<pre data-portal-review-selection></pre></label><label>${text("core.portal.092")}<select data-portal-review-intent><option value="question">${text("core.portal.093")}</option><option value="change_request">${text("core.portal.094")}</option></select></label><label>${text("core.portal.044")}<textarea required maxlength="65536" rows="5" placeholder="${text("core.portal.045")}" data-portal-review-question></textarea></label><p class="portal-review-error" data-portal-review-error role="alert"></p><footer><button type="submit" value="cancel" formnovalidate>${text("core.portal.046")}</button><button type="submit" class="portal-review-submit">${text("core.portal.047")}</button></footer></form>`;
        $('[data-portal-review-title]', dialog).textContent = title;
        const confirmDialog: any = document.createElement('dialog');
        confirmDialog.className = 'portal-review-dialog portal-review-confirm';
        confirmDialog.innerHTML = `<form method="dialog"><header><h2>${text("core.portal.077")}</h2></header><p>${text("core.portal.078")}</p><p class="portal-review-error" data-portal-review-delete-error role="alert"></p><footer><button type="submit" value="cancel">${text("core.portal.046")}</button><button type="submit" class="portal-review-delete">${text("core.portal.072")}</button></footer></form>`;
        const { scrim, panel }: any = createDiscussionPanel({
            ariaLabel: text("core.portal.054"),
            closeAttribute: 'data-portal-review-close',
            closeLabel: text("core.portal.057"),
            copyAttribute: 'data-portal-review-copy-prompt',
            copyHelp: text("core.portal.076"),
            copyLabel: text("core.portal.075"),
            id: 'project-discussions-panel',
            listAttribute: 'data-portal-review-list',
            newAction: { attribute: 'data-portal-review-new', hidden: !canCreate, label: text("core.portal.056") },
            scrimAttribute: 'data-portal-review-scrim',
            scrimLabel: text("core.portal.057"),
            subtitle: path,
            summaryAttribute: 'data-portal-review-summary',
            title: text("core.portal.054"),
        });
        const toast: any = document.createElement('div');
        toast.className = 'portal-review-toast';
        toast.hidden = true;
        toast.setAttribute('role', 'status');
        toast.setAttribute('aria-live', 'polite');
        const toastMessage: any = document.createElement('span');
        const toastAction: any = document.createElement('button');
        toastAction.type = 'button';
        toastAction.hidden = true;
        toast.append(toastMessage, toastAction);
        document.body.append(menu, dialog, confirmDialog, scrim, panel, toast);
        let reviewState: any = null;
        let reviewEtag: any = '';
        let pending: any = null;
        let dialogSelection: any = null;
        let composerMode: any = { operation: 'create' };
        let pendingDelete: any = '';
        let panelReturn: any = null;
        let panelCloseTimer: any = 0;
        let toastCallback: any = null;
        let toastTimer: any = 0;
        const hideMenu: any = () => { menu.hidden = true; pending = null; };
        const announce: any = (message: any, actionLabel: any = '', callback: any = null) => {
            window.clearTimeout(toastTimer);
            toastMessage.textContent = message;
            toastAction.textContent = actionLabel;
            toastAction.hidden = !actionLabel;
            toastCallback = callback;
            toast.hidden = false;
            toastTimer = window.setTimeout(() => { toast.hidden = true; }, actionLabel ? 8000 : 2600);
        };
        const reviewGuard: any = () => ({ expectedRevision: reviewState?.revision || 0, expectedStateDigest: reviewState?.stateDigest || '' });
        const reviewMutation: any = async (endpoint: any, action: any, method: any, body: any) => {
            const send: any = async (payload: any) => {
                const response: any = await fetch(`${review}${endpoint}`, { method, headers: { 'Content-Type': 'application/json', 'X-Toudocu-Action': action }, body: JSON.stringify(payload) });
                return { response, result: await response.json() };
            };
            let attempt: any = await send(body);
            if (!attempt.response.ok && attempt.result.diagnostics?.[0]?.code === 'AGENT_REVISION_CONFLICT') {
                await loadReview();
                attempt = await send({ ...body, ...reviewGuard() });
            }
            if (!attempt.response.ok)
                throw new Error(attempt.result.diagnostics?.[0]?.message || `HTTP ${attempt.response.status}`);
            return attempt.result;
        };
        const projectDiscussions: any = () => [...(reviewState?.session?.discussions || [])]
            .sort((left: any, right: any) => Number(left.state !== 'open') - Number(right.state !== 'open') || String(left.createdAt).localeCompare(String(right.createdAt)));
        const discussionInFlight: any = (discussionId: any) => (reviewState?.deliveries || []).some((delivery: any) => delivery.discussionId === discussionId && delivery.state !== 'responded');
        const messageEditable: any = (message: any) => message.author === 'human' && (message.state === 'draft' || (reviewState?.deliveries || []).some((delivery: any) => delivery.id === message.deliveryId && delivery.state === 'pending'));
        const renderReview: any = () => {
            const discussions: any = projectDiscussions();
            const open: any = discussions.filter((discussion: any) => discussion.state === 'open');
            $('[data-open-discussion-count]', toggle).textContent = String(open.length);
            $('[data-portal-review-summary]', panel).textContent = text("core.portal.055", [open.length, discussions.length - open.length]);
            const list: any = $('[data-portal-review-list]', panel);
            const scrollTop: any = list.scrollTop;
            list.replaceChildren();
            discussions.forEach((discussion: any) => {
                const placement: any = discussion.placement || {};
                const article: any = document.createElement('article');
                article.className = `portal-review-thread is-${discussion.state}`;
                article.innerHTML = `<header><div><strong>${text("core.portal.070")}</strong><span>${text(discussion.state === 'open' ? "core.portal.059" : "core.portal.060")} · ${escapeHTML(placementLabel(placement.status || 'current'))}</span></div><div class="portal-review-thread-actions"><button type="button" data-portal-thread-state>${text(discussion.state === 'open' ? "core.portal.071" : "core.portal.073")}</button><button type="button" class="is-danger" data-portal-thread-delete>${text("core.portal.072")}</button></div></header><p class="portal-review-anchor">${escapeHTML(placement.path || discussion.target?.path || path)}${placement.range ? `:${placement.range.start.line}` : ''}</p>${discussion.anchor?.selectedText ? `<blockquote class="portal-review-quote">${escapeHTML(discussion.anchor.selectedText)}</blockquote>` : ''}<ol>${discussion.messages.map((message: any) => `<li class="is-${escapeHTML(message.author)}" data-portal-message="${escapeHTML(message.id)}"><div><strong>${message.author === 'agent' ? `${text("core.portal.061")} · ${escapeHTML(outcomeLabel(message.outcome || 'answered'))}` : `${text("core.portal.070")} · ${escapeHTML(message.intent === 'change_request' ? text("core.portal.094") : text("core.portal.093"))}`}</strong><time>${escapeHTML(new Date(message.createdAt).toLocaleString(window.ToudocuPage?.ui.locale || 'en'))}</time></div><p>${escapeHTML(message.text)}</p>${messageEditable(message) ? `<div class="portal-review-thread-actions"><button type="button" data-portal-message-edit>${text("core.portal.095")}</button><button type="button" class="is-danger" data-portal-message-delete>${text("core.portal.072")}</button></div>` : ''}</li>`).join('')}</ol><button type="button" class="portal-review-reply" data-portal-thread-reply ${discussion.state !== 'open' || discussionInFlight(discussion.id) || discussion.messages.some((message: any) => message.state === 'draft') ? 'disabled' : ''}>${text("core.portal.074")}</button>`;
                $('[data-portal-thread-state]', article).addEventListener('click', async (event: any) => {
                    event.currentTarget.disabled = true;
                    try {
                        reviewState = await reviewMutation(`/discussions/${discussion.id}`, 'agent-discussion-update', 'PATCH', { ...reviewGuard(), state: discussion.state === 'open' ? 'resolved' : 'open' });
                        renderReview();
                    }
                    catch (failure: any) { await loadReview().catch(() => { }); announce(failure.message); }
                }, { signal });
                $('[data-portal-thread-delete]', article).addEventListener('click', () => {
                    pendingDelete = discussion.id;
                    $('[data-portal-review-delete-error]', confirmDialog).textContent = '';
                    confirmDialog.showModal();
                }, { signal });
                $$('[data-portal-message]', article).forEach((item: any) => {
                    const message: any = discussion.messages.find((candidate: any) => candidate.id === item.dataset.portalMessage);
                    if (!message || !messageEditable(message))
                        return;
                    $('[data-portal-message-edit]', item)?.addEventListener('click', () => openComposer({ operation: 'edit', discussionId: discussion.id, message }), { signal });
                    $('[data-portal-message-delete]', item)?.addEventListener('click', async (event: any) => {
                        event.currentTarget.disabled = true;
                        try {
                            reviewState = await reviewMutation(`/discussions/${discussion.id}/messages/${message.id}`, 'agent-message-delete', 'DELETE', reviewGuard());
                            renderReview();
                        }
                        catch (failure: any) { announce(failure.message); }
                    }, { signal });
                });
                $('[data-portal-thread-reply]', article).addEventListener('click', () => openComposer({ operation: 'reply', discussionId: discussion.id }), { signal });
                list.append(article);
            });
            if (!discussions.length)
                list.innerHTML = `<div class="portal-review-empty"><p>${text("core.portal.058")}</p>${canCreate ? `<button type="button" data-portal-review-empty-new>${text("core.portal.056")}</button>` : ''}</div>`;
            $('[data-portal-review-empty-new]', list)?.addEventListener('click', () => openComposer({ operation: 'create' }), { signal });
            list.scrollTop = scrollTop;
        };
        const loadReview: any = async () => {
            const response: any = await fetch(`${review}/discussions`, { headers: reviewEtag ? { 'If-None-Match': reviewEtag } : {}, cache: 'no-store' });
            if (response.status === 304)
                return;
            const result: any = await response.json();
            if (!response.ok)
                throw new Error(result.diagnostics?.[0]?.message || `HTTP ${response.status}`);
            reviewState = result;
            reviewEtag = response.headers.get('ETag') || '';
            renderReview();
        };
        const openPanel: any = async (refresh = true) => {
            window.clearTimeout(panelCloseTimer);
            panelReturn = document.activeElement;
            panel.hidden = false;
            scrim.hidden = false;
            toggle.setAttribute('aria-expanded', 'true');
            if (matchMedia('(max-width: 560px)').matches) {
                panel.setAttribute('role', 'dialog');
                panel.setAttribute('aria-modal', 'true');
            }
            requestAnimationFrame(() => panel.classList.add('is-open'));
            $('[data-portal-review-close]', panel).focus();
            try { if (refresh) await loadReview(); }
            catch (failure: any) {
                $('[data-portal-review-list]', panel).innerHTML = `<div class="portal-review-empty is-error"><p>${escapeHTML(text("core.portal.080", [failure.message]))}</p><button type="button" data-portal-review-retry>${text("core.portal.081")}</button></div>`;
                $('[data-portal-review-retry]', panel)?.addEventListener('click', () => loadReview().catch((error: any) => announce(error.message)), { signal });
            }
        };
        const closePanel: any = () => {
            panel.classList.remove('is-open');
            scrim.hidden = true;
            toggle.setAttribute('aria-expanded', 'false');
            panel.setAttribute('role', 'complementary');
            panel.removeAttribute('aria-modal');
            panelCloseTimer = window.setTimeout(() => { panel.hidden = true; }, 180);
            panelReturn?.focus?.();
        };
        function openComposer(mode: any) {
            composerMode = mode;
            dialogSelection = mode.selection || null;
            $('[data-portal-review-dialog-title]', dialog).textContent = text(mode.operation === 'reply' ? "core.portal.082" : "core.portal.042");
            $('[data-portal-review-selection-wrap]', dialog).hidden = !dialogSelection;
            $('[data-portal-review-selection]', dialog).textContent = dialogSelection?.text || '';
            $('[data-portal-review-question]', dialog).value = mode.message?.text || '';
            $('[data-portal-review-intent]', dialog).value = mode.message?.intent || 'question';
            $('[data-portal-review-error]', dialog).textContent = '';
            $('.portal-review-submit', dialog).textContent = text(mode.operation === 'reply' ? "core.portal.083" : mode.operation === 'edit' ? "core.portal.099" : "core.portal.047");
            dialog.showModal();
            requestAnimationFrame(() => $('[data-portal-review-question]', dialog).focus());
        }
        const showMenu: any = () => {
            const selection: any = window.getSelection();
            if (!selection || selection.isCollapsed || !selection.rangeCount || !selection.toString().trim())
                return hideMenu();
            const range: any = selection.getRangeAt(0);
            const source: any = selectionSources.find((candidate: any) => candidate.contains(range.startContainer) && candidate.contains(range.endContainer));
            if (!source)
                return hideMenu();
            const rectangles: any = range.getClientRects();
            const rectangle: any = rectangles[rectangles.length - 1] || range.getBoundingClientRect();
            const selectedText: any = selection.toString();
            const prefixRange: any = document.createRange();
            prefixRange.selectNodeContents(source);
            prefixRange.setEnd(range.startContainer, range.startOffset);
            const prefix: any = selectionSources.slice(0, selectionSources.indexOf(source)).map((candidate: any) => candidate.textContent || '').join('') + prefixRange.toString();
            let occurrence: any = 1;
            for (let offset: any = 0; (offset = prefix.indexOf(selectedText, offset)) >= 0; offset += selectedText.length)
                occurrence++;
            pending = { text: selectedText, occurrence };
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
        };
        const copySelection: any = async (value: any, success: any) => {
            hideMenu();
            announce(await copyText(value) ? success : text("core.portal.041"));
        };
        $('[data-selection-copy]', menu).addEventListener('click', () => pending && copySelection(pending.text, text("core.portal.039")), { signal });
        $('[data-selection-context]', menu).addEventListener('click', () => pending && copySelection(text("core.portal.038", [title, path, pending.text]), text("core.portal.040")), { signal });
        $('[data-selection-question]', menu).addEventListener('click', () => {
            if (!pending)
                return;
            const selection: any = { text: pending.text, occurrence: pending.occurrence };
            hideMenu();
            openComposer({ operation: 'create', selection });
        }, { signal });
        $('form', dialog).addEventListener('submit', async (event: any) => {
            if (event.submitter?.value === 'cancel')
                return;
            event.preventDefault();
            const question: any = $('[data-portal-review-question]', dialog).value.trim();
            const intent: any = $('[data-portal-review-intent]', dialog).value;
            if (!question)
                return;
            const submit: any = event.submitter;
            const error: any = $('[data-portal-review-error]', dialog);
            submit.disabled = true;
            submit.textContent = text("core.portal.048");
            error.textContent = '';
            try {
                if (!reviewState)
                    await loadReview();
                if (composerMode.operation === 'reply') {
                    reviewState = await reviewMutation(`/discussions/${composerMode.discussionId}/messages`, 'agent-message-create', 'POST', { ...reviewGuard(), intent, text: question });
                }
                else if (composerMode.operation === 'edit') {
                    reviewState = await reviewMutation(`/discussions/${composerMode.discussionId}/messages/${composerMode.message.id}`, 'agent-message-update', 'PATCH', { ...reviewGuard(), intent, text: question });
                }
                else {
                    if (new TextEncoder().encode(question).length > 65536)
                        throw new Error(text("core.portal.053"));
                    reviewState = await reviewMutation('/discussions', 'agent-discussion-create', 'POST', { ...reviewGuard(), target: { kind: 'document', path, documentId: page.page?.id || '' }, selection: dialogSelection ? { selectedText: dialogSelection.text, occurrence: dialogSelection.occurrence } : undefined, intent, text: question });
                }
                dialog.close();
                renderReview();
                await openPanel(false);
                announce(text(composerMode.operation === 'reply' ? "core.portal.084" : composerMode.operation === 'edit' ? "core.portal.098" : "core.portal.050"));
            }
            catch (failure: any) {
                error.textContent = text("core.portal.049", [failure.message]);
            }
            finally {
                submit.disabled = false;
                submit.textContent = text(composerMode.operation === 'reply' ? "core.portal.083" : composerMode.operation === 'edit' ? "core.portal.099" : "core.portal.047");
            }
        }, { signal });
        $('form', confirmDialog).addEventListener('submit', async (event: any) => {
            if (event.submitter?.value === 'cancel')
                return;
            event.preventDefault();
            const submit: any = event.submitter;
            const error: any = $('[data-portal-review-delete-error]', confirmDialog);
            submit.disabled = true;
            error.textContent = '';
            try {
                reviewState = await reviewMutation(`/discussions/${pendingDelete}`, 'agent-discussion-delete', 'DELETE', reviewGuard());
                confirmDialog.close();
                renderReview();
                announce(text("core.portal.079"));
            }
            catch (failure: any) { error.textContent = failure.message; }
            finally { submit.disabled = false; }
        }, { signal });
        toggle.addEventListener('click', () => toggle.getAttribute('aria-expanded') === 'true' ? closePanel() : openPanel(), { signal });
        $('[data-portal-review-new]', panel).addEventListener('click', () => openComposer({ operation: 'create' }), { signal });
        $('[data-portal-review-close]', panel).addEventListener('click', closePanel, { signal });
        scrim.addEventListener('click', closePanel, { signal });
        $('[data-portal-review-copy-prompt]', panel).addEventListener('click', async () => announce(await copyText(text("core.portal.087")) ? text("core.portal.088") : text("core.portal.041")), { signal });
        toastAction.addEventListener('click', () => toastCallback?.(), { signal });
        selectionArea?.addEventListener('pointerup', showMenu, { signal });
        selectionArea?.addEventListener('keyup', (event: any) => { if (event.key === 'Shift' || !event.shiftKey) showMenu(); }, { signal });
        document.addEventListener('pointerdown', (event: any) => { if (!menu.contains(event.target)) hideMenu(); }, { signal });
        document.addEventListener('scroll', hideMenu, { capture: true, signal });
        window.addEventListener('resize', hideMenu, { signal });
        document.addEventListener('keydown', (event: any) => { if (event.key !== 'Escape') return; if (!menu.hidden) { event.preventDefault(); hideMenu(); } else if (toggle.getAttribute('aria-expanded') === 'true' && !dialog.open && !confirmDialog.open) closePanel(); }, { capture: true, signal });
        const pollTimer: any = window.setInterval(() => { if (panel.classList.contains('is-open') && !document.hidden && !dialog.open && !confirmDialog.open) loadReview().catch(() => { }); }, 2000);
        loadReview().catch(() => { });
        signal.addEventListener('abort', () => { window.clearTimeout(toastTimer); window.clearTimeout(panelCloseTimer); window.clearInterval(pollTimer); menu.remove(); dialog.remove(); confirmDialog.remove(); scrim.remove(); panel.remove(); toast.remove(); }, { once: true });
    }
    function initializeCodeCopy() {
        $$('.code-block').forEach((block: any) => {
            const code: any = $('code', block);
            if (!code)
                return;
            const button: any = document.createElement('button');
            button.type = 'button';
            button.className = 'copy-code';
            button.textContent = text("core.portal.019");
            button.addEventListener('click', async () => {
                try {
                    await navigator.clipboard.writeText(code.textContent || '');
                    button.textContent = text("core.portal.020");
                    window.setTimeout(() => { button.textContent = text("core.portal.021"); }, 1300);
                }
                catch {
                    const range: any = document.createRange();
                    range.selectNodeContents(code);
                    const selection: any = window.getSelection();
                    selection.removeAllRanges();
                    selection.addRange(range);
                    button.textContent = text("core.portal.022");
                }
            });
            block.append(button);
        });
    }
    function initializeDiagramViewport({ stage, target, zoomIn, zoomOut, fitButton, fullscreenButton, signal }: any) {
        if (!stage || !target)
            return null;
        let view: any = { scale: 1, x: 0, y: 0 };
        let dragging: any = null;
        let nativeFullscreen: any = false;
        function applyTransform() {
            const svg: any = $('svg', target);
            if (!svg)
                return;
            svg.style.transformOrigin = 'center center';
            svg.style.transform = `translate(${view.x}px, ${view.y}px) scale(${view.scale})`;
        }
        function fit() {
            view = { scale: 1, x: 0, y: 0 };
            applyTransform();
        }
        function updateFullscreenButton() {
            if (!fullscreenButton)
                return;
            const expanded: any = document.fullscreenElement === stage || stage.classList.contains('is-fullscreen-fallback');
            fullscreenButton.textContent = expanded ? text("core.portal.023") : text("core.portal.024");
            fullscreenButton.setAttribute('aria-label', expanded ? text("core.portal.025") : text("core.portal.026"));
        }
        zoomIn?.addEventListener('click', () => {
            view.scale = Math.min(3, view.scale + 0.15);
            applyTransform();
        });
        zoomOut?.addEventListener('click', () => {
            view.scale = Math.max(0.45, view.scale - 0.15);
            applyTransform();
        });
        fitButton?.addEventListener('click', fit);
        fullscreenButton?.addEventListener('click', async () => {
            if (document.fullscreenElement === stage) {
                await document.exitFullscreen();
            }
            else if (stage.classList.contains('is-fullscreen-fallback')) {
                stage.classList.remove('is-fullscreen-fallback');
            }
            else if (stage.requestFullscreen) {
                try {
                    await stage.requestFullscreen();
                }
                catch {
                    stage.classList.add('is-fullscreen-fallback');
                }
            }
            else {
                stage.classList.add('is-fullscreen-fallback');
            }
            updateFullscreenButton();
            fit();
        });
        document.addEventListener('fullscreenchange', () => {
            const expanded: any = document.fullscreenElement === stage;
            if (!expanded && !nativeFullscreen)
                return;
            nativeFullscreen = expanded;
            updateFullscreenButton();
            fit();
        }, { signal });
        document.addEventListener('keydown', (event: any) => {
            if (event.key !== 'Escape' || !stage.classList.contains('is-fullscreen-fallback'))
                return;
            stage.classList.remove('is-fullscreen-fallback');
            updateFullscreenButton();
            fit();
        }, { signal });
        stage.addEventListener('wheel', (event: any) => {
            if (!event.target.closest('svg'))
                return;
            event.preventDefault();
            view.scale = Math.max(0.45, Math.min(3, view.scale + (event.deltaY < 0 ? 0.1 : -0.1)));
            applyTransform();
        }, { passive: false });
        stage.addEventListener('pointerdown', (event: any) => {
            if (event.button !== 0 || !event.target.closest('svg'))
                return;
            dragging = { x: event.clientX, y: event.clientY, originX: view.x, originY: view.y };
            stage.setPointerCapture(event.pointerId);
            stage.classList.add('is-panning');
        });
        stage.addEventListener('pointermove', (event: any) => {
            if (!dragging)
                return;
            view.x = dragging.originX + event.clientX - dragging.x;
            view.y = dragging.originY + event.clientY - dragging.y;
            applyTransform();
        });
        const stopPan: any = () => {
            dragging = null;
            stage.classList.remove('is-panning');
        };
        stage.addEventListener('pointerup', stopPan);
        stage.addEventListener('pointercancel', stopPan);
        updateFullscreenButton();
        return { fit };
    }
    function initializeMermaid(signal: any) {
        const containers: any = $$('[data-mermaid-container]').filter((container: any) => $('[data-mermaid-diagram]', container));
        if (!containers.length)
            return;
        const viewports: any = new Map();
        containers.forEach((container: any) => {
            const viewport: any = initializeDiagramViewport({
                stage: $('[data-mermaid-stage]', container),
                target: $('[data-mermaid-diagram]', container),
                zoomIn: $('[data-mermaid-zoom-in]', container),
                zoomOut: $('[data-mermaid-zoom-out]', container),
                fitButton: $('[data-mermaid-fit]', container),
                fullscreenButton: $('[data-mermaid-fullscreen]', container),
                signal,
            });
            if (viewport)
                viewports.set(container, viewport);
        });
        const showError: any = (container: any) => {
            container.classList.add('has-error');
            const target: any = $('[data-mermaid-diagram]', container);
            const error: any = $('[data-mermaid-error]', container);
            if (target)
                target.replaceChildren();
            if (error)
                error.hidden = false;
        };
        let renderQueue: any = Promise.resolve();
        let started: any = false;
        const renderAll: any = async () => {
            if (!window.mermaid) {
                containers.forEach(showError);
                return;
            }
            window.mermaid.initialize({
                startOnLoad: false,
                securityLevel: 'strict',
                secure: ['secure', 'securityLevel', 'startOnLoad', 'maxTextSize', 'suppressErrorRendering'],
                maxTextSize: 50000,
                suppressErrorRendering: true,
                ...mermaidThemeConfig(),
            });
            for (const container of containers) {
                const target: any = $('[data-mermaid-diagram]', container);
                const source: any = $('.mermaid-source code', container)?.textContent || '';
                const error: any = $('[data-mermaid-error]', container);
                if (!target)
                    continue;
                container.classList.remove('has-error');
                if (error)
                    error.hidden = true;
                target.removeAttribute('data-processed');
                target.textContent = source;
                try {
                    await window.mermaid.run({ nodes: [target] });
                    viewports.get(container)?.fit();
                }
                catch {
                    showError(container);
                }
            }
        };
        const scheduleRender: any = () => {
            started = true;
            const render: any = async () => {
                try {
                    if (!window.mermaid)
                        await loadScript('mermaid.tiny.js');
                    await renderAll();
                }
                catch {
                    containers.forEach(showError);
                }
            };
            renderQueue = renderQueue.then(render, render);
        };
        document.addEventListener('toudocu:themechange', () => {
            if (started)
                scheduleRender();
        }, { signal });
        if ('IntersectionObserver' in window) {
            const observer: any = new IntersectionObserver((entries: any) => {
                if (!entries.some((entry: any) => entry.isIntersecting))
                    return;
                observer.disconnect();
                scheduleRender();
            }, { rootMargin: '320px 0px' });
            containers.forEach((container: any) => observer.observe(container));
            signal.addEventListener('abort', () => observer.disconnect(), { once: true });
        }
        else {
            globalThis.setTimeout(scheduleRender, 0);
        }
    }
    function initializeTocTracking(signal: any) {
        const links: any = $$('.page-toc a[href^="#"]');
        if (!links.length || !('IntersectionObserver' in window))
            return;
        const byId: any = new Map(links.map((link: any) => [decodeURIComponent(link.hash.slice(1)), link]));
        const headings: any = [...byId.keys()].map((id: any) => document.getElementById(id)).filter(Boolean);
        const observer: any = new IntersectionObserver((entries: any) => {
            const visible: any = entries.filter((entry: any) => entry.isIntersecting).sort((a: any, b: any) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
            if (!visible)
                return;
            links.forEach((link: any) => link.classList.toggle('is-active', link === byId.get(visible.target.id)));
        }, { rootMargin: '-80px 0px -70% 0px', threshold: 0 });
        headings.forEach((heading: any) => observer.observe(heading));
        signal.addEventListener('abort', () => observer.disconnect(), { once: true });
    }
    function initializeUseCaseTabs(signal: any) {
        $$('[data-usecase-tabs]').forEach((container: any) => {
            const tabs: any = $$('[data-usecase-tab]', container);
            const panels: any = $$('[data-usecase-panel]', container);
            if (!tabs.length || !panels.length)
                return;
            const ids: any = new Set(panels.map((panel: any) => panel.id));
            container.classList.add('is-enhanced');
            function activate(id: any, updateHistory: any = false) {
                const targetID: any = ids.has(id) ? id : 'overview';
                tabs.forEach((tab: any) => {
                    const active: any = tab.dataset.usecaseTab === targetID;
                    tab.classList.toggle('is-active', active);
                    tab.setAttribute('aria-selected', String(active));
                    tab.tabIndex = active ? 0 : -1;
                });
                panels.forEach((panel: any) => {
                    panel.hidden = panel.id !== targetID;
                });
                const activePanel: any = panels.find((panel: any) => panel.id === targetID);
                activePanel?.dispatchEvent(new CustomEvent('toudocu:panelshown', { bubbles: true }));
                if (updateHistory && window.location.hash !== `#${targetID}`) {
                    window.history.pushState(null, '', `#${targetID}`);
                }
            }
            tabs.forEach((tab: any, index: any) => {
                tab.addEventListener('click', (event: any) => {
                    event.preventDefault();
                    activate(tab.dataset.usecaseTab, true);
                });
                tab.addEventListener('keydown', (event: any) => {
                    let next: any = index;
                    if (event.key === 'ArrowRight')
                        next = (index + 1) % tabs.length;
                    else if (event.key === 'ArrowLeft')
                        next = (index - 1 + tabs.length) % tabs.length;
                    else if (event.key === 'Home')
                        next = 0;
                    else if (event.key === 'End')
                        next = tabs.length - 1;
                    else
                        return;
                    event.preventDefault();
                    selectTab(tabs, next);
                    activate(tabs[next].dataset.usecaseTab, true);
                });
            });
            window.addEventListener('hashchange', () => activate(window.location.hash.slice(1)), { signal });
            window.addEventListener('popstate', () => activate(window.location.hash.slice(1)), { signal });
            activate(window.location.hash.slice(1));
        });
    }
    function initializePrint() {
        $('[data-print]')?.addEventListener('click', () => window.print());
    }
    async function initializePage(signal: any) {
        initializeHeroSummary();
        initializeSidebar(signal);
        initializeCollectionFilters();
        initializeTaskFilters();
        initializeCollapsibleSections();
        initializeDocumentContextCopy();
        initializeDocumentReview(signal);
        initializeCodeCopy();
        initializeMermaid(signal);
        initializeUseCaseTabs(signal);
        initializeTocTracking(signal);
        if ($('[data-screen-map]')) {
            await loadScript('screen-map.js').catch(() => { });
            if (!signal.aborted)
                window.ToudocuInitializeScreenMap?.(document, signal);
        }
        if ($('[data-playable-flow]')) {
            await loadScript('playable-flow.js').catch(() => { });
            if (!signal.aborted)
                window.ToudocuInitializePlayableFlow?.(document, signal);
        }
    }
    let pageController: any = new AbortController();
    initializeGlobalSearch();
    initializePrint();
    initializePage(pageController.signal);
    document.addEventListener('toudocu:pagechange', () => {
        pageController.abort();
        pageController = new AbortController();
        initializePage(pageController.signal);
    });
})();
