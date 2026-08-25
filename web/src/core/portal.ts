import { text } from "./locale";
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
                background: color('--td-surface-canvas'),
                primaryColor: color('--td-surface-muted'),
                primaryTextColor: color('--td-text'),
                primaryBorderColor: color('--td-accent'),
                lineColor: color('--td-border-strong'),
                secondaryColor: color('--td-surface-raised'),
                tertiaryColor: color('--td-surface'),
                noteBkgColor: color('--td-accent-soft'),
                noteTextColor: color('--td-text'),
                fontFamily: color('--td-font-body'),
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
                    folderToggle.setAttribute('aria-expanded', String(!collapsed));
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
                const empty: any = document.createElement('div');
                empty.className = 'ui-empty-state';
                empty.dataset.uiState = 'empty';
                empty.textContent = text("core.portal.008");
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
                toggle.textContent = '';
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
                    tabs.forEach((candidate: any, candidateIndex: any) => {
                        candidate.setAttribute('aria-selected', String(candidateIndex === next));
                        candidate.tabIndex = candidateIndex === next ? 0 : -1;
                    });
                    tabs[next]?.focus();
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
