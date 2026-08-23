import { parseBootstrap } from "./bootstrap";

(() => {
    'use strict';
    const revisionSelector: any = 'meta[name="toudocu-revision"]';
    const cache: any = new Map();
    const maxCacheEntries: any = 8;
    let prefetchTimer: any = 0;
    let navigating: any = false;
    const currentRevision: any = () => (document.querySelector(revisionSelector) as HTMLMetaElement | null)?.content || '';
    const canonicalURL: any = (value: any) => {
        const url: any = new URL(value, window.location.href);
        url.hash = '';
        return url.href;
    };
    function isCanonicalLink(link: any, event: any = null) {
        if (!link || !link.href || link.hasAttribute('download') || link.target)
            return false;
        if (event && (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey))
            return false;
        const raw: any = link.getAttribute('href') || '';
        if (!raw || raw.startsWith('#') || /^(mailto|tel|javascript):/i.test(raw))
            return false;
        const url: any = new URL(link.href, window.location.href);
        if (url.origin !== window.location.origin || !/^https?:$/.test(url.protocol))
            return false;
        if (url.pathname.startsWith('/_toudocu/') || url.pathname.startsWith('/changes/') || url.pathname.startsWith('/api/'))
            return false;
        if (/^\/[a-z]{2}(?:-[A-Z]{2})?\//.test(url.pathname))
            return false;
        if (!url.pathname.endsWith('/') && !url.pathname.endsWith('.html'))
            return false;
        return true;
    }
    function touchCache(key: any, value: any) {
        cache.delete(key);
        cache.set(key, value);
        while (cache.size > maxCacheEntries)
            cache.delete(cache.keys().next().value);
    }
    async function fetchPage(url: any) {
        const key: any = canonicalURL(url);
        if (cache.has(key)) {
            const cached: any = cache.get(key);
            touchCache(key, cached);
            return cached;
        }
        const request: any = fetch(key, { headers: { Accept: 'text/html' }, credentials: 'same-origin' })
            .then(async (response: any) => {
            if (!response.ok || !response.headers.get('content-type')?.includes('text/html'))
                throw new Error(`HTTP ${response.status}`);
            const parsed: any = new DOMParser().parseFromString(await response.text(), 'text/html');
                const revision: any = (parsed.querySelector(revisionSelector) as HTMLMetaElement | null)?.content || '';
                if (!revision || !parsed.querySelector('[data-toudocu-serve-navigation]') || !parsed.querySelector('.site-layout') || !parsed.querySelector('#toudocu-page')) {
                throw new Error('not a canonical serve page');
            }
            if (revision !== currentRevision()) {
                cache.clear();
                throw new Error('serve revision changed');
            }
            return parsed;
        })
            .catch((error: any) => {
            cache.delete(key);
            throw error;
        });
        touchCache(key, request);
        return request;
    }
    function setBusy(busy: any) {
        document.documentElement.classList.toggle('is-soft-navigating', busy);
        document.querySelector('main')?.setAttribute('aria-busy', String(busy));
    }
    function replaceOptional(selector: any, nextDocument: any) {
        const current: any = document.querySelector(selector);
        const next: any = nextDocument.querySelector(selector);
        if (current && next)
            current.replaceWith(next.cloneNode(true));
        else if (current && !next)
            current.remove();
    }
    function syncPageStyles(nextDocument: any, pageURL: any) {
        document.querySelectorAll('link[data-page-style]').forEach((link: any) => link.remove());
        nextDocument.querySelectorAll('link[data-page-style]').forEach((link: any) => {
            const clone: any = link.cloneNode(true);
            clone.href = new URL(link.getAttribute('href'), pageURL).href;
            document.head.append(clone);
        });
    }
    function syncBootstrap(nextDocument: any) {
        const next: any = nextDocument.querySelector('#toudocu-page');
        const current: any = document.querySelector('#toudocu-page');
        if (!next?.textContent || !current)
            throw new Error('page bootstrap unavailable');
        let raw: any;
        try {
            raw = JSON.parse(next.textContent);
        }
        catch {
            throw new Error('page bootstrap invalid');
        }
        const parsed: any = parseBootstrap(raw);
        if (!parsed.ok || parsed.value.runtime !== 'serve')
            throw new Error(parsed.ok ? 'unexpected page runtime' : parsed.reason);
        current.textContent = next.textContent;
        window.ToudocuPage = parsed.value;
    }
    function scrollToTarget(url: any, restoreScroll: any) {
        if (url.hash) {
            const id: any = decodeURIComponent(url.hash.slice(1));
            const target: any = document.getElementById(id) || document.querySelector(`[name="${CSS.escape(id)}"]`);
            target?.scrollIntoView();
            return;
        }
        if (restoreScroll)
            window.scrollTo(restoreScroll.x || 0, restoreScroll.y || 0);
        else
            window.scrollTo(0, 0);
    }
    async function navigate(value: any, { historyMode = 'push', restoreScroll = null }: any = {}) {
        const url: any = new URL(value, window.location.href);
        if (navigating)
            return;
        navigating = true;
        setBusy(true);
        try {
            const nextDocument: any = await fetchPage(url);
            const nextLayout: any = nextDocument.querySelector('.site-layout');
            const currentLayout: any = document.querySelector('.site-layout');
            if (!currentLayout || !nextLayout)
                throw new Error('page layout unavailable');
            const sidebarScrollTop: any = document.querySelector('.sidebar')?.scrollTop || 0;
            currentLayout.replaceWith(nextLayout.cloneNode(true));
            document.title = nextDocument.title;
            document.documentElement.lang = nextDocument.documentElement.lang;
            document.body.dataset.rootPrefix = nextDocument.body.dataset.rootPrefix || '';
            const description: any = nextDocument.querySelector('meta[name="description"]')?.content || '';
            document.querySelector('meta[name="description"]')?.setAttribute('content', description);
            const favicon: any = nextDocument.querySelector('link[rel="icon"]')?.getAttribute('href');
            if (favicon)
                document.querySelector('link[rel="icon"]')?.setAttribute('href', new URL(favicon, url).href);
            const nextBrand: any = nextDocument.querySelector('.brand');
            const brand: any = document.querySelector('.brand');
            if (brand && nextBrand) {
                brand.setAttribute('href', new URL(nextBrand.getAttribute('href'), url).href);
                brand.innerHTML = nextBrand.innerHTML;
            }
            replaceOptional('.language-select', nextDocument);
            syncBootstrap(nextDocument);
            syncPageStyles(nextDocument, url);
            document.body.classList.remove('sidebar-open');
            if (historyMode === 'push')
                history.pushState({ toudocu: true, scrollX: 0, scrollY: 0 }, '', url);
            document.dispatchEvent(new CustomEvent('toudocu:pagechange', { detail: { url: url.href } }));
            const sidebar: any = document.querySelector('.sidebar');
            if (sidebar)
                sidebar.scrollTop = sidebarScrollTop;
            scrollToTarget(url, restoreScroll);
            const main: any = document.querySelector('main');
            if (historyMode !== 'pop' && main) {
                main.tabIndex = -1;
                main.focus({ preventScroll: true });
                main.addEventListener('blur', () => main.removeAttribute('tabindex'), { once: true });
            }
        }
        catch {
            window.location.assign(url.href);
        }
        finally {
            navigating = false;
            setBusy(false);
        }
    }
    function schedulePrefetch(link: any) {
        window.clearTimeout(prefetchTimer);
        if (!isCanonicalLink(link))
            return;
        prefetchTimer = window.setTimeout(() => { fetchPage(link.href).catch(() => { }); }, 80);
    }
    history.replaceState({ ...(history.state || {}), toudocu: true, scrollX: window.scrollX, scrollY: window.scrollY }, '');
    history.scrollRestoration = 'manual';
    let scrollFrame: any = 0;
    window.addEventListener('scroll', () => {
        if (scrollFrame)
            return;
        scrollFrame = window.requestAnimationFrame(() => {
            scrollFrame = 0;
            history.replaceState({
                ...(history.state || {}),
                toudocu: true,
                scrollX: window.scrollX,
                scrollY: window.scrollY,
            }, '');
        });
    }, { passive: true });
    document.addEventListener('click', (event: any) => {
        const link: any = event.target.closest('a[href]');
        if (!isCanonicalLink(link, event))
            return;
        const target: any = new URL(link.href, window.location.href);
        if (canonicalURL(target) === canonicalURL(window.location.href))
            return;
        event.preventDefault();
        history.replaceState({ ...(history.state || {}), toudocu: true, scrollX: window.scrollX, scrollY: window.scrollY }, '');
        navigate(target);
    });
    document.addEventListener('pointerover', (event: any) => schedulePrefetch(event.target.closest('a[href]')));
    document.addEventListener('focusin', (event: any) => schedulePrefetch(event.target.closest('a[href]')));
    window.addEventListener('popstate', (event: any) => {
        navigate(window.location.href, {
            historyMode: 'pop',
            restoreScroll: event.state?.toudocu ? { x: event.state.scrollX, y: event.state.scrollY } : null,
        });
    });
})();
