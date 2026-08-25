import { text } from "./locale";
(() => {
    'use strict';
    const page: any = window.ToudocuPage;
    const rebuildEndpoint: any = page?.runtime === 'serve' && page.capabilities?.rebuild ? page.endpoints?.rebuild : '';
    const editorEndpoint: any = page?.runtime === 'serve' && page.capabilities?.editor ? page.endpoints?.editor : '';
    const versionEndpoint: any = page?.runtime === 'serve' && page.capabilities?.updateCheck ? page.endpoints?.version : '';
    const button: any = document.querySelector('[data-server-rebuild]');
    const status: any = document.querySelector('[data-server-rebuild-status]');
    const label: any = document.querySelector('[data-server-rebuild-label]');
    const baseline: any = (document.querySelector('meta[name="toudocu-revision"]') as HTMLMetaElement | null)?.content || '';
    let etag: any = baseline ? `"${baseline}"` : '';
    function releaseURL(latestVersion: string, value: unknown): URL | null {
        if (typeof value !== 'string')
            return null;
        try {
            const parsed: any = new URL(value);
            const expected: any = `https://github.com/lumenikoly/toudocu/releases/tag/${encodeURIComponent(latestVersion)}`;
            return parsed.href === expected ? parsed : null;
        }
        catch {
            return null;
        }
    }
    async function initializeUpdateNotice() {
        if (!versionEndpoint || document.querySelector('[data-update-notice]'))
            return;
        try {
            const response: any = await fetch(versionEndpoint, {
                cache: 'no-store',
                credentials: 'same-origin',
                headers: { Accept: 'application/json' },
            });
            if (!response.ok)
                return;
            const result: any = await response.json();
            if (result?.schemaVersion !== 1 || result.status !== 'update-available' || typeof result.currentVersion !== 'string' || typeof result.latestVersion !== 'string')
                return;
            const target: any = releaseURL(result.latestVersion, result.releaseURL);
            if (!target)
                return;
            const storageKey: any = 'toudocu-dismissed-update';
            try {
                if (localStorage.getItem(storageKey) === result.latestVersion)
                    return;
            }
            catch { /* storage may be unavailable */ }
            const notice: any = document.createElement('aside');
            notice.className = 'update-notice';
            notice.dataset.updateNotice = '';
            notice.setAttribute('role', 'status');
            notice.setAttribute('aria-live', 'polite');
            const copy: any = document.createElement('div');
            copy.className = 'update-notice-copy';
            const title: any = document.createElement('strong');
            title.textContent = text('core.serve-runtime.008', [result.latestVersion]);
            const current: any = document.createElement('span');
            current.textContent = text('core.serve-runtime.009', [result.currentVersion]);
            copy.append(title, current);
            const actions: any = document.createElement('div');
            actions.className = 'update-notice-actions';
            const link: any = document.createElement('a');
            link.href = target.href;
            link.target = '_blank';
            link.rel = 'noopener noreferrer';
            link.textContent = text('core.serve-runtime.010');
            const dismiss: any = document.createElement('button');
            dismiss.type = 'button';
            dismiss.textContent = text('core.serve-runtime.011');
            dismiss.setAttribute('aria-label', text('core.serve-runtime.012', [result.latestVersion]));
            dismiss.addEventListener('click', () => {
                try {
                    localStorage.setItem(storageKey, result.latestVersion);
                }
                catch { /* hide for this page even without storage */ }
                notice.remove();
            });
            actions.append(link, dismiss);
            notice.append(copy, actions);
            document.querySelector('.site-header')?.insertAdjacentElement('afterend', notice);
        }
        catch {
            // Version discovery is optional; portal content remains fully usable.
        }
    }
    async function pollRevision() {
        try {
            if (!editorEndpoint)
                return;
            const response: any = await fetch(`${editorEndpoint}/files`, {
                cache: 'no-store',
                headers: etag ? { 'If-None-Match': etag } : {},
            });
            if (response.status === 304)
                return;
            if (!response.ok)
                return;
            const next: any = response.headers.get('ETag') || '';
            if (etag && next && next !== etag) {
                if (document.querySelector('[data-roadmap-dialog][open], [data-roadmap-dialog][data-open]'))
                    return;
                window.location.reload();
                return;
            }
            etag = next;
        }
        catch {
            // The server may be stopping; the current rendered page remains usable.
        }
    }
    button?.addEventListener('click', async () => {
        if (button.disabled)
            return;
        button.disabled = true;
        button.classList.remove('has-error', 'has-success');
        button.classList.add('is-rebuilding');
        if (label)
            label.textContent = text("core.serve-runtime.001");
        if (status)
            status.textContent = text("core.serve-runtime.002");
        try {
            if (!rebuildEndpoint)
                throw new Error(text("core.serve-runtime.003"));
            const response: any = await fetch(rebuildEndpoint, {
                method: 'POST',
                cache: 'no-store',
                headers: { Accept: 'application/json', 'X-Toudocu-Action': 'rebuild' },
            });
            if (!response.ok) {
                const detail: any = await response.text();
                throw new Error(detail.trim() || `HTTP ${response.status}`);
            }
            const result: any = await response.json();
            button.classList.remove('is-rebuilding');
            button.classList.add('has-success');
            if (label)
                label.textContent = text("core.serve-runtime.004");
            if (status)
                status.textContent = text("core.serve-runtime.005", [result.documents || 0, result.pages || 0]);
            window.setTimeout(() => window.location.reload(), 700);
        }
        catch (error: any) {
            button.disabled = false;
            button.classList.remove('is-rebuilding');
            button.classList.add('has-error');
            if (label)
                label.textContent = text("core.serve-runtime.006");
            if (status)
                status.textContent = text("core.serve-runtime.007", [error.message]);
        }
    });
    pollRevision();
    initializeUpdateNotice();
    window.setInterval(pollRevision, 2000);
})();
