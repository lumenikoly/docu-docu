import { text } from "../../core/locale";
window.ToudocuInitializeScreenMap = (scope: any, signal: any) => {
    'use strict';
    scope = scope || document;
    const workspace: any = scope.querySelector('[data-screen-map]');
    if (!workspace)
        return;
    if (workspace.dataset.pageInitialized === 'true')
        return;
    workspace.dataset.pageInitialized = 'true';
    const data: any = JSON.parse(workspace.querySelector('[data-screen-map-data]')?.textContent || '{}');
    const screens: any = data.screens || [];
    const transitions: any = data.transitions || [];
    const flows: any = data.flows || [];
    const CARD_WIDTH: any = 278;
    const MIN_CARD_HEIGHT: any = 248;
    const MODULE_COLUMN_GAP: any = 144;
    const MODULE_ROW_GAP: any = 112;
    const MODULE_GROUP_GAP: any = 96;
    const FLOW_COLUMN_GAP: any = 180;
    const FLOW_ROW_GAP: any = 120;
    const SITEMAP_COLUMN_GAP: any = 120;
    const SITEMAP_ROW_GAP: any = 160;
    const NODE_CLEARANCE: any = 18;
    const LABEL_CLEARANCE: any = 10;
    const ROUTE_CORNER_RADIUS: any = 14;
    const ROUTE_LANE_OFFSET: any = 12;
    const byId: any = new Map(screens.map((screen: any) => [screen.id, screen]));
    const nodeById: any = new Map([...workspace.querySelectorAll('[data-screen-node]')].map((node: any) => [node.dataset.screenNode, node]));
    const stage: any = workspace.querySelector('[data-map-stage]');
    const viewport: any = workspace.querySelector('[data-map-viewport]');
    const groupsLayer: any = workspace.querySelector('[data-map-groups]');
    const nodesLayer: any = workspace.querySelector('[data-map-nodes]');
    const edgeLayer: any = workspace.querySelector('[data-map-edges]');
    const labelsLayer: any = workspace.querySelector('[data-map-labels]');
    const inspector: any = workspace.querySelector('[data-map-inspector]');
    const empty: any = workspace.querySelector('[data-map-empty]');
    const status: any = workspace.querySelector('[data-map-summary]');
    const moduleSelect: any = workspace.querySelector('[data-map-module]');
    const useCaseSelect: any = workspace.querySelector('[data-map-usecase]');
    const statusSelect: any = workspace.querySelector('[data-map-status]');
    const search: any = workspace.querySelector('[data-map-search]');
    const changesToggle: any = workspace.querySelector('[data-map-changes]');
    const page: any = window.ToudocuPage;
    const locale: any = page?.ui.locale || 'en';
    const changesAPI: any = page?.runtime === 'serve' && page.capabilities?.changes ? page.endpoints?.changes : '';
    if (changesToggle && !changesAPI)
        changesToggle.hidden = true;
    const changeStatusSelect: any = workspace.querySelector('[data-map-change-status]');
    const initialUseCase: any = workspace.dataset.mapInitialUsecase || '';
    let mode: any = initialUseCase ? 'usecase' : 'all';
    let selected: any = '';
    let selectedTransition: any = '';
    let scale: any = 1;
    let panX: any = 24;
    let panY: any = 24;
    let dragging: any = false;
    let dragStart: any = null;
    let visible: any = new Set(screens.map((screen: any) => screen.id));
    let activeEdges: any = transitions;
    const positions: any = new Map();
    let groupBounds: any = [];
    let cardHeight: any = MIN_CARD_HEIGHT;
    let canvasBounds: any = { width: 900, height: 620 };
    let changesLoaded: any = false;
    let changesActive: any = false;
    const changedNodes: any = new Map();
    const changedEdges: any = new Map();
    function selectedFlow() {
        return flows.find((flow: any) => flow.useCase === useCaseSelect?.value);
    }
    function computeVisible() {
        const query: any = (search?.value || '').trim().toLocaleLowerCase(locale);
        let values: any = screens.filter((screen: any) => !screen._changeGhost || changesActive);
        if (changesActive)
            values = values.filter((screen: any) => changedNodes.has(screen.id));
        if (changesActive && changeStatusSelect?.value)
            values = values.filter((screen: any) => changedNodes.get(screen.id)?.status === changeStatusSelect.value);
        if (mode === 'module' && moduleSelect?.value)
            values = values.filter((screen: any) => screen.module === moduleSelect.value);
        if (mode === 'usecase' && selectedFlow()) {
            const allowed: any = new Set(selectedFlow().reachableScreens || []);
            values = values.filter((screen: any) => allowed.has(screen.id));
        }
        if (mode === 'unfinished')
            values = values.filter((screen: any) => ['in-progress', 'planned', 'blocked'].includes(screen.status?.kind));
        if (statusSelect?.value)
            values = values.filter((screen: any) => screen.status?.kind === statusSelect.value);
        if (query) {
            values = values.filter((screen: any) => [screen.id, screen.title, screen.route, screen.module].join(' ').toLocaleLowerCase(locale).includes(query));
        }
        visible = new Set(values.map((screen: any) => screen.id));
        if (mode === 'sitemap') {
            activeEdges = screens.filter((screen: any) => screen.parent && visible.has(screen.parent) && visible.has(screen.id))
                .map((screen: any) => ({ id: `parent-${screen.id}`, source: screen.parent, target: screen.id, action: text("features.screen-map.index.001"), condition: '', type: 'navigation' }));
        }
        else {
            activeEdges = transitions.filter((transition: any) => visible.has(transition.source) && visible.has(transition.target));
            if (changesActive)
                activeEdges = activeEdges.filter((transition: any) => changedEdges.has(transition.id));
            if (changesActive && changeStatusSelect?.value)
                activeEdges = activeEdges.filter((transition: any) => changedEdges.get(transition.id)?.status === changeStatusSelect.value);
            if (mode === 'usecase' && useCaseSelect?.value) {
                activeEdges = activeEdges.filter((transition: any) => !transition.useCase || transition.useCase === useCaseSelect.value);
            }
        }
    }
    function layout() {
        positions.clear();
        groupBounds = [];
        const ids: any = [...visible].sort();
        if (mode === 'all' || mode === 'module' || mode === 'unfinished') {
            const groups: any = new Map();
            ids.forEach((id: any) => {
                const module: any = byId.get(id)?.module || text("features.screen-map.index.002");
                if (!groups.has(module))
                    groups.set(module, []);
                groups.get(module).push(id);
            });
            let cursorX: any = 48;
            [...groups.entries()].sort((a: any, b: any) => a[0].localeCompare(b[0], locale, { numeric: true })).forEach(([module, members]: any) => {
                members.sort((a: any, b: any) => a.localeCompare(b, locale, { numeric: true }));
                const columns: any = Math.min(2, members.length);
                const rows: any = Math.ceil(members.length / columns);
                const width: any = 64 + columns * CARD_WIDTH + Math.max(0, columns - 1) * MODULE_COLUMN_GAP;
                const height: any = 96 + rows * cardHeight + Math.max(0, rows - 1) * MODULE_ROW_GAP;
                members.forEach((id: any, index: any) => {
                    positions.set(id, {
                        x: cursorX + 32 + (index % columns) * (CARD_WIDTH + MODULE_COLUMN_GAP),
                        y: 96 + Math.floor(index / columns) * (cardHeight + MODULE_ROW_GAP),
                    });
                });
                groupBounds.push({ module, x: cursorX, y: 32, width, height });
                cursorX += width + MODULE_GROUP_GAP;
            });
            canvasBounds = {
                width: Math.max(900, cursorX),
                height: Math.max(620, ...groupBounds.map((group: any) => group.y + group.height + 128)),
            };
            return;
        }
        const incoming: any = new Map(ids.map((id: any) => [id, 0]));
        const adjacency: any = new Map(ids.map((id: any) => [id, []]));
        activeEdges.forEach((edge: any) => {
            if (!visible.has(edge.source) || !visible.has(edge.target) || edge.source === edge.target)
                return;
            incoming.set(edge.target, (incoming.get(edge.target) || 0) + 1);
            adjacency.get(edge.source)?.push(edge.target);
        });
        const roots: any = ids.filter((id: any) => (incoming.get(id) || 0) === 0);
        if (!roots.length && ids.length)
            roots.push(ids[0]);
        const levels: any = new Map();
        const queue: any = roots.map((id: any) => [id, 0]);
        while (queue.length) {
            const [id, depth]: any = queue.shift();
            if ((levels.get(id) ?? -1) >= depth)
                continue;
            levels.set(id, depth);
            (adjacency.get(id) || []).forEach((target: any) => queue.push([target, Math.min(depth + 1, ids.length)]));
        }
        ids.forEach((id: any) => {
            if (!levels.has(id))
                levels.set(id, 0);
        });
        const groups: any = new Map();
        ids.forEach((id: any) => {
            const level: any = levels.get(id);
            if (!groups.has(level))
                groups.set(level, []);
            groups.get(level).push(id);
        });
        const vertical: any = mode === 'sitemap';
        [...groups.entries()].sort((a: any, b: any) => a[0] - b[0]).forEach(([level, members]: any) => {
            members.sort((a: any, b: any) => a.localeCompare(b, locale, { numeric: true }));
            members.forEach((id: any, row: any) => {
                positions.set(id, vertical
                    ? {
                        x: 48 + row * (CARD_WIDTH + SITEMAP_COLUMN_GAP),
                        y: 56 + level * (cardHeight + SITEMAP_ROW_GAP),
                    }
                    : {
                        x: 48 + level * (CARD_WIDTH + FLOW_COLUMN_GAP),
                        y: 56 + row * (cardHeight + FLOW_ROW_GAP),
                    });
            });
        });
        canvasBounds = {
            width: Math.max(900, ...[...positions.values()].map((position: any) => position.x + CARD_WIDTH + 128)),
            height: Math.max(620, ...[...positions.values()].map((position: any) => position.y + cardHeight + 128)),
        };
    }
    function setCanvasSize(width: any, height: any) {
        nodesLayer.style.width = `${width}px`;
        nodesLayer.style.height = `${height}px`;
        groupsLayer.style.width = `${width}px`;
        groupsLayer.style.height = `${height}px`;
        labelsLayer.style.width = `${width}px`;
        labelsLayer.style.height = `${height}px`;
        edgeLayer.setAttribute('width', width);
        edgeLayer.setAttribute('height', height);
        viewport.style.width = `${width}px`;
        viewport.style.height = `${height}px`;
    }
    function drawGroups() {
        groupsLayer.replaceChildren();
        groupBounds.forEach((group: any) => {
            const element: any = document.createElement('div');
            element.className = 'screen-module-group';
            element.style.transform = `translate(${group.x}px, ${group.y}px)`;
            element.style.width = `${group.width}px`;
            element.style.height = `${group.height}px`;
            const label: any = document.createElement('strong');
            label.textContent = group.module;
            element.append(label);
            groupsLayer.append(element);
        });
    }
    function screenRect(id: any, padding: any = 0) {
        const position: any = positions.get(id);
        if (!position)
            return null;
        return {
            id,
            left: position.x - padding,
            top: position.y - padding,
            right: position.x + CARD_WIDTH + padding,
            bottom: position.y + cardHeight + padding,
        };
    }
    function rectsOverlap(first: any, second: any, padding: any = 0) {
        return first.left < second.right + padding
            && first.right > second.left - padding
            && first.top < second.bottom + padding
            && first.bottom > second.top - padding;
    }
    function segmentIntersectsRect(first: any, second: any, rect: any) {
        const epsilon: any = .1;
        if (Math.abs(first.x - second.x) < epsilon) {
            const low: any = Math.min(first.y, second.y);
            const high: any = Math.max(first.y, second.y);
            return first.x > rect.left + epsilon && first.x < rect.right - epsilon
                && high > rect.top + epsilon && low < rect.bottom - epsilon;
        }
        if (Math.abs(first.y - second.y) < epsilon) {
            const low: any = Math.min(first.x, second.x);
            const high: any = Math.max(first.x, second.x);
            return first.y > rect.top + epsilon && first.y < rect.bottom - epsilon
                && high > rect.left + epsilon && low < rect.right - epsilon;
        }
        return true;
    }
    function simplifyRoute(points: any) {
        const result: any = [];
        points.forEach((point: any) => {
            const previous: any = result[result.length - 1];
            if (!previous || previous.x !== point.x || previous.y !== point.y)
                result.push(point);
            while (result.length >= 3) {
                const first: any = result[result.length - 3];
                const middle: any = result[result.length - 2];
                const last: any = result[result.length - 1];
                if ((first.x === middle.x && middle.x === last.x) || (first.y === middle.y && middle.y === last.y)) {
                    result.splice(result.length - 2, 1);
                }
                else
                    break;
            }
        });
        return result;
    }
    function routeSegments(points: any) {
        return points.slice(1).map((point: any, index: any) => ({ first: points[index], second: point }));
    }
    function routeLength(points: any) {
        return routeSegments(points).reduce((total: any, segment: any) => total
            + Math.abs(segment.second.x - segment.first.x)
            + Math.abs(segment.second.y - segment.first.y), 0);
    }
    function roundedRoute(points: any) {
        if (points.length < 2)
            return '';
        let path: any = `M ${points[0].x} ${points[0].y}`;
        for (let index: any = 1; index < points.length - 1; index += 1) {
            const previous: any = points[index - 1];
            const current: any = points[index];
            const next: any = points[index + 1];
            const incoming: any = Math.abs(current.x - previous.x) + Math.abs(current.y - previous.y);
            const outgoing: any = Math.abs(next.x - current.x) + Math.abs(next.y - current.y);
            const radius: any = Math.min(ROUTE_CORNER_RADIUS, incoming / 2, outgoing / 2);
            const before: any = {
                x: current.x + Math.sign(previous.x - current.x) * radius,
                y: current.y + Math.sign(previous.y - current.y) * radius,
            };
            const after: any = {
                x: current.x + Math.sign(next.x - current.x) * radius,
                y: current.y + Math.sign(next.y - current.y) * radius,
            };
            path += ` L ${before.x} ${before.y} Q ${current.x} ${current.y}, ${after.x} ${after.y}`;
        }
        const last: any = points[points.length - 1];
        return `${path} L ${last.x} ${last.y}`;
    }
    function portFor(rect: any, side: any) {
        const centerX: any = (rect.left + rect.right) / 2;
        const centerY: any = (rect.top + rect.bottom) / 2;
        const values: any = {
            left: { actual: { x: rect.left, y: centerY }, outside: { x: rect.left - NODE_CLEARANCE, y: centerY }, axis: 'horizontal', direction: -1 },
            right: { actual: { x: rect.right, y: centerY }, outside: { x: rect.right + NODE_CLEARANCE, y: centerY }, axis: 'horizontal', direction: 1 },
            top: { actual: { x: centerX, y: rect.top }, outside: { x: centerX, y: rect.top - NODE_CLEARANCE }, axis: 'vertical', direction: -1 },
            bottom: { actual: { x: centerX, y: rect.bottom }, outside: { x: centerX, y: rect.bottom + NODE_CLEARANCE }, axis: 'vertical', direction: 1 },
        };
        return values[side];
    }
    function routeObstacleHits(points: any, obstacles: any, sourceID: any, targetID: any) {
        const hits: any = new Set();
        routeSegments(points).forEach((segment: any, segmentIndex: any, segments: any) => obstacles.forEach((obstacle: any) => {
            if (obstacle.id === sourceID && segmentIndex === 0)
                return;
            if (obstacle.id === targetID && segmentIndex === segments.length - 1)
                return;
            if (segmentIntersectsRect(segment.first, segment.second, obstacle))
                hits.add(obstacle.id);
        }));
        return hits.size;
    }
    function guideValues(obstacles: any, axis: any, middle: any, outerLow: any, outerHigh: any) {
        const values: any = new Set([middle, outerLow, outerHigh]);
        const boundaries: any = [];
        obstacles.forEach((rect: any) => {
            const low: any = axis === 'x' ? rect.left : rect.top;
            const high: any = axis === 'x' ? rect.right : rect.bottom;
            values.add(low);
            values.add(high);
            boundaries.push(low, high);
        });
        boundaries.sort((first: any, second: any) => first - second);
        for (let index: any = 1; index < boundaries.length; index += 1) {
            if (boundaries[index] - boundaries[index - 1] > NODE_CLEARANCE) {
                values.add((boundaries[index] + boundaries[index - 1]) / 2);
            }
        }
        const ranked: any = [...values].sort((first: any, second: any) => Math.abs(first - middle) - Math.abs(second - middle) || first - second);
        return [...new Set([...ranked.slice(0, 16), outerLow, outerHigh])];
    }
    function routeConflictCost(points: any, usedSegments: any) {
        let cost: any = 0;
        routeSegments(points).forEach((candidate: any) => {
            usedSegments.forEach((used: any) => {
                const candidateHorizontal: any = candidate.first.y === candidate.second.y;
                const usedHorizontal: any = used.first.y === used.second.y;
                if (candidateHorizontal === usedHorizontal) {
                    const sameLane: any = candidateHorizontal
                        ? Math.abs(candidate.first.y - used.first.y) < 1
                        : Math.abs(candidate.first.x - used.first.x) < 1;
                    if (!sameLane)
                        return;
                    const candidateLow: any = candidateHorizontal ? Math.min(candidate.first.x, candidate.second.x) : Math.min(candidate.first.y, candidate.second.y);
                    const candidateHigh: any = candidateHorizontal ? Math.max(candidate.first.x, candidate.second.x) : Math.max(candidate.first.y, candidate.second.y);
                    const usedLow: any = usedHorizontal ? Math.min(used.first.x, used.second.x) : Math.min(used.first.y, used.second.y);
                    const usedHigh: any = usedHorizontal ? Math.max(used.first.x, used.second.x) : Math.max(used.first.y, used.second.y);
                    cost += Math.max(0, Math.min(candidateHigh, usedHigh) - Math.max(candidateLow, usedLow)) * 1.5;
                    return;
                }
                const horizontal: any = candidateHorizontal ? candidate : used;
                const vertical: any = candidateHorizontal ? used : candidate;
                const horizontalLow: any = Math.min(horizontal.first.x, horizontal.second.x);
                const horizontalHigh: any = Math.max(horizontal.first.x, horizontal.second.x);
                const verticalLow: any = Math.min(vertical.first.y, vertical.second.y);
                const verticalHigh: any = Math.max(vertical.first.y, vertical.second.y);
                if (vertical.first.x > horizontalLow && vertical.first.x < horizontalHigh
                    && horizontal.first.y > verticalLow && horizontal.first.y < verticalHigh)
                    cost += 24;
            });
        });
        return cost;
    }
    function routeAroundObstacles(edge: any, obstacles: any, usedSegments: any, laneOffset: any = 0) {
        const sourceRect: any = screenRect(edge.source);
        const targetRect: any = screenRect(edge.target);
        if (!sourceRect || !targetRect)
            return [];
        const allLeft: any = Math.min(...obstacles.map((rect: any) => rect.left));
        const allRight: any = Math.max(...obstacles.map((rect: any) => rect.right));
        const allTop: any = Math.min(...obstacles.map((rect: any) => rect.top));
        const allBottom: any = Math.max(...obstacles.map((rect: any) => rect.bottom));
        const outerLeft: any = Math.max(16, allLeft - 64);
        const outerRight: any = Math.max(canvasBounds.width - 32, allRight + 64);
        const outerTop: any = Math.max(16, allTop - 64);
        const outerBottom: any = Math.max(canvasBounds.height - 32, allBottom + 64);
        const sourceCenter: any = { x: (sourceRect.left + sourceRect.right) / 2, y: (sourceRect.top + sourceRect.bottom) / 2 };
        const targetCenter: any = { x: (targetRect.left + targetRect.right) / 2, y: (targetRect.top + targetRect.bottom) / 2 };
        const preferredAxis: any = mode === 'sitemap' || Math.abs(targetCenter.y - sourceCenter.y) > Math.abs(targetCenter.x - sourceCenter.x)
            ? 'vertical' : 'horizontal';
        const xGuides: any = guideValues(obstacles, 'x', (sourceCenter.x + targetCenter.x) / 2, outerLeft, outerRight);
        const yGuides: any = guideValues(obstacles, 'y', (sourceCenter.y + targetCenter.y) / 2, outerTop, outerBottom);
        const candidates: any = [];
        const blockedCandidates: any = [];
        const addCandidate: any = (points: any, axis: any, outer: any = false) => {
            const simplified: any = simplifyRoute(points);
            let preference: any = axis === preferredAxis ? 0 : 120;
            if (edge.type === 'return')
                preference += outer ? -100 : 180;
            else if (outer)
                preference += 120;
            const obstacleHits: any = routeObstacleHits(simplified, obstacles, edge.source, edge.target);
            const candidate: any = {
                points: simplified,
                score: routeLength(simplified) + Math.max(0, simplified.length - 2) * 36 + preference
                    + routeConflictCost(simplified, usedSegments) + obstacleHits * 10000,
            };
            if (obstacleHits)
                blockedCandidates.push(candidate);
            else
                candidates.push(candidate);
        };
        const horizontalPairs: any = [['right', 'left'], ['left', 'right'], ['right', 'right'], ['left', 'left']];
        horizontalPairs.forEach(([sourceSide, targetSide]: any) => {
            const source: any = portFor(sourceRect, sourceSide);
            const target: any = portFor(targetRect, targetSide);
            if (source.outside.y === target.outside.y && Math.sign(target.outside.x - source.outside.x) === source.direction
                && Math.sign(source.outside.x - target.outside.x) === target.direction) {
                addCandidate([source.actual, source.outside, target.outside, target.actual], 'horizontal');
            }
            xGuides.forEach((guide: any) => {
                const lane: any = guide + laneOffset;
                if (Math.sign(lane - source.outside.x) !== source.direction || Math.sign(lane - target.outside.x) !== target.direction)
                    return;
                addCandidate([source.actual, source.outside, { x: lane, y: source.outside.y }, { x: lane, y: target.outside.y }, target.outside, target.actual], 'horizontal', guide === outerLeft || guide === outerRight);
            });
        });
        const verticalPairs: any = [['bottom', 'top'], ['top', 'bottom'], ['bottom', 'bottom'], ['top', 'top']];
        verticalPairs.forEach(([sourceSide, targetSide]: any) => {
            const source: any = portFor(sourceRect, sourceSide);
            const target: any = portFor(targetRect, targetSide);
            if (source.outside.x === target.outside.x && Math.sign(target.outside.y - source.outside.y) === source.direction
                && Math.sign(source.outside.y - target.outside.y) === target.direction) {
                addCandidate([source.actual, source.outside, target.outside, target.actual], 'vertical');
            }
            yGuides.forEach((guide: any) => {
                const lane: any = guide + laneOffset;
                if (Math.sign(lane - source.outside.y) !== source.direction || Math.sign(lane - target.outside.y) !== target.direction)
                    return;
                addCandidate([source.actual, source.outside, { x: source.outside.x, y: lane }, { x: target.outside.x, y: lane }, target.outside, target.actual], 'vertical', guide === outerTop || guide === outerBottom);
            });
        });
        return (candidates.length ? candidates : blockedCandidates)
            .sort((first: any, second: any) => first.score - second.score || roundedRoute(first.points).localeCompare(roundedRoute(second.points), locale));
    }
    function selfRouteCandidates(edge: any, obstacles: any, usedSegments: any, laneOffset: any = 0) {
        const rect: any = screenRect(edge.source);
        if (!rect)
            return [];
        const centerX: any = (rect.left + rect.right) / 2;
        const centerY: any = (rect.top + rect.bottom) / 2;
        const distance: any = 84 + Math.abs(laneOffset);
        const candidates: any = [
            [{ x: rect.right, y: centerY - 42 }, { x: rect.right + NODE_CLEARANCE, y: centerY - 42 }, { x: rect.right + distance, y: centerY - 42 }, { x: rect.right + distance, y: centerY + 42 }, { x: rect.right + NODE_CLEARANCE, y: centerY + 42 }, { x: rect.right, y: centerY + 42 }],
            [{ x: centerX - 54, y: rect.bottom }, { x: centerX - 54, y: rect.bottom + NODE_CLEARANCE }, { x: centerX - 54, y: rect.bottom + distance }, { x: centerX + 54, y: rect.bottom + distance }, { x: centerX + 54, y: rect.bottom + NODE_CLEARANCE }, { x: centerX + 54, y: rect.bottom }],
            [{ x: rect.left, y: centerY + 42 }, { x: rect.left - NODE_CLEARANCE, y: centerY + 42 }, { x: rect.left - distance, y: centerY + 42 }, { x: rect.left - distance, y: centerY - 42 }, { x: rect.left - NODE_CLEARANCE, y: centerY - 42 }, { x: rect.left, y: centerY - 42 }],
            [{ x: centerX + 54, y: rect.top }, { x: centerX + 54, y: rect.top - NODE_CLEARANCE }, { x: centerX + 54, y: rect.top - distance }, { x: centerX - 54, y: rect.top - distance }, { x: centerX - 54, y: rect.top - NODE_CLEARANCE }, { x: centerX - 54, y: rect.top }],
        ];
        const ranked: any = candidates
            .map(simplifyRoute)
            .filter((points: any) => points.every((point: any) => point.x >= 16 && point.y >= 16))
            .map((points: any) => {
            const obstacleHits: any = routeObstacleHits(points, obstacles, edge.source, edge.target);
            return {
                points,
                obstacleHits,
                score: routeLength(points) + routeConflictCost(points, usedSegments) + obstacleHits * 10000,
            };
        })
            .sort((first: any, second: any) => first.score - second.score);
        const clear: any = ranked.filter((candidate: any) => candidate.obstacleHits === 0);
        return clear.length ? clear : ranked;
    }
    function labelRectangle(center: any, size: any) {
        return {
            left: center.x - size.width / 2,
            top: center.y - size.height / 2,
            right: center.x + size.width / 2,
            bottom: center.y + size.height / 2,
        };
    }
    function labelCandidates(points: any, size: any) {
        const candidates: any = [];
        routeSegments(points)
            .map((segment: any) => ({ ...segment, length: Math.abs(segment.second.x - segment.first.x) + Math.abs(segment.second.y - segment.first.y) }))
            .sort((first: any, second: any) => second.length - first.length)
            .forEach((segment: any) => {
            [.5, .34, .66].forEach((ratio: any) => {
                const point: any = {
                    x: segment.first.x + (segment.second.x - segment.first.x) * ratio,
                    y: segment.first.y + (segment.second.y - segment.first.y) * ratio,
                };
                if (segment.first.y === segment.second.y) {
                    candidates.push({ x: point.x, y: point.y - size.height / 2 - LABEL_CLEARANCE });
                    candidates.push({ x: point.x, y: point.y + size.height / 2 + LABEL_CLEARANCE });
                }
                else {
                    candidates.push({ x: point.x + size.width / 2 + LABEL_CLEARANCE, y: point.y });
                    candidates.push({ x: point.x - size.width / 2 - LABEL_CLEARANCE, y: point.y });
                }
            });
        });
        return candidates;
    }
    function findLabelPlacement(points: any, size: any, nodeRects: any, occupiedLabels: any) {
        return labelCandidates(points, size).find((center: any) => {
            const rect: any = labelRectangle(center, size);
            if (rect.left < 16 || rect.top < 16)
                return false;
            if (nodeRects.some((node: any) => rectsOverlap(rect, node, LABEL_CLEARANCE)))
                return false;
            return !occupiedLabels.some((label: any) => rectsOverlap(rect, label, 6));
        });
    }
    function createTransitionLabel(edge: any) {
        const label: any = document.createElement('button');
        label.type = 'button';
        label.className = 'screen-edge-label';
        label.dataset.transitionLabel = edge.id;
        label.dataset.transitionId = edge.id;
        label.setAttribute('aria-label', `${edge.id}: ${edge.action}${edge.condition ? text("features.screen-map.index.003", [edge.condition]) : ''}`);
        label.title = `${edge.id} · ${edge.action}${edge.condition ? ` · ${edge.condition}` : ''}`;
        const action: any = document.createElement('span');
        action.className = 'screen-edge-label-action';
        action.textContent = edge.action;
        label.append(action);
        if (edge.condition) {
            label.classList.add('has-condition');
            const condition: any = document.createElement('span');
            condition.className = 'screen-edge-label-condition';
            condition.textContent = edge.condition;
            label.append(condition);
        }
        label.addEventListener('click', (event: any) => {
            event.stopPropagation();
            selectTransition(edge.id);
        });
        labelsLayer.append(label);
        return label;
    }
    function routeLaneOffsets(edges: any) {
        const groups: any = new Map();
        edges.forEach((edge: any) => {
            const key: any = `${edge.source}\u0000${edge.target}`;
            if (!groups.has(key))
                groups.set(key, []);
            groups.get(key).push(edge.id);
        });
        const offsets: any = new Map();
        groups.forEach((ids: any) => {
            ids.sort((first: any, second: any) => first.localeCompare(second, locale, { numeric: true }));
            ids.forEach((id: any, index: any) => offsets.set(id, (index - (ids.length - 1) / 2) * ROUTE_LANE_OFFSET));
        });
        return offsets;
    }
    function drawEdges() {
        edgeLayer.innerHTML = `<defs><marker id="screen-map-arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto"><path d="M0,0 L0,6 L9,3 z"></path></marker></defs>`;
        labelsLayer.replaceChildren();
        const edges: any = [...activeEdges].sort((first: any, second: any) => first.id.localeCompare(second.id, locale, { numeric: true }));
        const obstacles: any = [...visible].map((id: any) => screenRect(id, NODE_CLEARANCE)).filter(Boolean);
        const nodeRects: any = [...visible].map((id: any) => screenRect(id)).filter(Boolean);
        const occupiedLabels: any = [];
        const usedSegments: any = [];
        const offsets: any = routeLaneOffsets(edges);
        let fallbackIndex: any = 0;
        let requiredWidth: any = canvasBounds.width;
        let requiredHeight: any = canvasBounds.height;
        edges.forEach((edge: any) => {
            const label: any = createTransitionLabel(edge);
            const labelSize: any = { width: label.offsetWidth, height: label.offsetHeight };
            const routeOptions: any = edge.source === edge.target
                ? selfRouteCandidates(edge, obstacles, usedSegments, offsets.get(edge.id) || 0)
                : routeAroundObstacles(edge, obstacles, usedSegments, offsets.get(edge.id) || 0);
            if (!routeOptions.length) {
                label.remove();
                return;
            }
            let chosenRoute: any = routeOptions[0];
            let labelCenter: any = null;
            for (const candidate of routeOptions) {
                const placement: any = findLabelPlacement(candidate.points, labelSize, nodeRects, occupiedLabels);
                if (placement) {
                    chosenRoute = candidate;
                    labelCenter = placement;
                    break;
                }
            }
            let leader: any = null;
            if (!labelCenter) {
                fallbackIndex += 1;
                const anchor: any = chosenRoute.points[Math.floor(chosenRoute.points.length / 2)];
                labelCenter = {
                    x: Math.max(canvasBounds.width, ...nodeRects.map((rect: any) => rect.right)) + 48 + labelSize.width / 2,
                    y: 48 + fallbackIndex * (labelSize.height + 16),
                };
                leader = { first: anchor, second: { x: labelCenter.x - labelSize.width / 2, y: labelCenter.y } };
            }
            const labelRect: any = labelRectangle(labelCenter, labelSize);
            occupiedLabels.push(labelRect);
            routeSegments(chosenRoute.points).forEach((segment: any) => usedSegments.push(segment));
            requiredWidth = Math.max(requiredWidth, labelRect.right + 48, ...chosenRoute.points.map((point: any) => point.x + 48));
            requiredHeight = Math.max(requiredHeight, labelRect.bottom + 48, ...chosenRoute.points.map((point: any) => point.y + 48));
            const group: any = document.createElementNS('http://www.w3.org/2000/svg', 'g');
            group.classList.add('screen-edge', `screen-edge-${edge.type || 'navigation'}`);
            const change: any = changedEdges.get(edge.id);
            if (change) {
                group.classList.add(`is-change-${change.status}`);
                group.setAttribute('aria-label', text("features.screen-map.index.004", [changeStatusLabel(change.status), edge.id]));
            }
            group.dataset.transitionId = edge.id;
            const path: any = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            path.classList.add('screen-edge-path');
            path.setAttribute('d', roundedRoute(chosenRoute.points));
            path.setAttribute('marker-end', 'url(#screen-map-arrow)');
            if (edge.type === 'external') {
                const outerPath: any = document.createElementNS('http://www.w3.org/2000/svg', 'path');
                outerPath.classList.add('screen-edge-path', 'screen-edge-external-outer');
                outerPath.setAttribute('d', path.getAttribute('d'));
                path.classList.add('screen-edge-external-inner');
                group.append(outerPath);
            }
            if (leader) {
                const leaderPath: any = document.createElementNS('http://www.w3.org/2000/svg', 'path');
                leaderPath.classList.add('screen-edge-leader');
                leaderPath.setAttribute('d', `M ${leader.first.x} ${leader.first.y} L ${leader.second.x} ${leader.second.y}`);
                group.append(leaderPath);
            }
            group.append(path);
            group.addEventListener('click', (event: any) => {
                event.stopPropagation();
                selectTransition(edge.id);
            });
            edgeLayer.append(group);
            label.style.left = `${labelCenter.x}px`;
            label.style.top = `${labelCenter.y}px`;
        });
        setCanvasSize(Math.ceil(requiredWidth), Math.ceil(requiredHeight));
        applySelectionStyles();
    }
    function applyTransform() {
        viewport.style.transform = `translate(${panX}px, ${panY}px) scale(${scale})`;
        if (status)
            status.textContent = text("features.screen-map.index.005", [visible.size, activeEdges.length, Math.round(scale * 100)]);
    }
    function renderInspector(id: any) {
        const screen: any = byId.get(id);
        if (!screen) {
            inspector.innerHTML = `<div class="screen-inspector-empty"><strong>${escapeText(text("screenMap.select"))}</strong><span>${escapeText(text("screenMap.selectHelp"))}</span></div>`;
            return;
        }
        const incomingRows: any = transitions.filter((transition: any) => transition.target === id)
            .map((transition: any) => `<li><code>${escapeText(transition.id)}</code><span>${escapeText(transition.source)} · ${escapeText(transition.action)} · ${escapeText(transition.condition)}</span></li>`).join('');
        const outgoingRows: any = transitions.filter((transition: any) => transition.source === id)
            .map((transition: any) => `<li><code>${escapeText(transition.id)}</code><span>${escapeText(transition.action)} · ${escapeText(transition.condition)} → ${escapeText(transition.target)}${transition.state ? ` @${escapeText(transition.state)}` : ''}${transition.error ? ` · ${escapeText(transition.error)}` : ''}${transition.useCase ? ` · ${escapeText(transition.useCase)}` : ''}</span></li>`).join('');
        const states: any = (screen.states || []).map((state: any) => `<span class="screen-state-chip">${escapeText(state.id)}</span>`).join('');
        const preview: any = screen.preview
            ? `<img class="screen-inspector-preview" src="${escapeAttribute(screen.preview)}" alt="${escapeAttribute(text("screenMap.previewAlt", [screen.title]))}">`
            : `<div class="screen-inspector-preview screen-preview-placeholder"><strong>${escapeText(screen.id)}</strong><span>${escapeText(text("screenMap.noPreview"))}</span></div>`;
        const nodeChange: any = changedNodes.get(id);
        const changeNotice: any = nodeChange ? `<p class="screen-change-notice is-${escapeAttribute(nodeChange.status)}">${escapeText(text("screenMap.changed", [changeStatusLabel(nodeChange.status)]))}</p>` : '';
        const noTransitions: any = `<li>${escapeText(text("screenMap.noTransitions"))}</li>`;
        inspector.innerHTML = `<div class="screen-inspector-head"><span>${escapeText(screen.module)}</span><button type="button" data-inspector-close>${escapeText(text("changes.close"))}</button></div>
<p class="screen-eyebrow">${escapeText(screen.id)}</p><h2>${escapeText(screen.title)}</h2>${preview}<p>${escapeText(screen.description || '')}</p>
<dl><div><dt>${escapeText(text("field.status"))}</dt><dd>${escapeText(screen.status?.label || '')}</dd></div><div><dt>${escapeText(text("field.route"))}</dt><dd><code>${escapeText(screen.route || '—')}</code></dd></div>
<div><dt>${escapeText(text("field.component"))}</dt><dd><code>${escapeText(screen.component || '—')}</code></dd></div></dl>
<div class="screen-inspector-states">${states}</div>
<h3>${escapeText(text("screenMap.useCasesTasks"))}</h3><p>${escapeText([...(screen.useCases || []), ...(screen.workItems || [])].join(' · ') || text("features.screen-map.index.023"))}</p>
<h3>${escapeText(text("screenMap.contracts"))}</h3><p>${escapeText((screen.contracts || []).join(' · ') || text("features.screen-map.index.024"))}</p>
<h3>${escapeText(text("screenMap.outgoingTransitions"))}</h3><ul>${outgoingRows || noTransitions}</ul>
<h3>${escapeText(text("screenMap.incomingTransitions"))}</h3><ul>${incomingRows || noTransitions}</ul>
${changeNotice}<a class="primary-link" href="${escapeAttribute(data.screenUrls?.[id] || '#')}">${escapeText(text("screenMap.openDocument"))}</a>`;
        inspector.querySelector('[data-inspector-close]')?.addEventListener('click', () => selectScreen(''));
    }
    function escapeText(value: any) {
        const element: any = document.createElement('span');
        element.textContent = value == null ? '' : String(value);
        return element.innerHTML;
    }
    function escapeAttribute(value: any) {
        return escapeText(value).replaceAll('"', '&quot;');
    }
    function transitionByID(id: any) {
        return activeEdges.find((item: any) => item.id === id) || transitions.find((item: any) => item.id === id);
    }
    function applySelectionStyles() {
        nodeById.forEach((node: any, nodeId: any) => {
            const transition: any = transitionByID(selectedTransition);
            node.classList.toggle('is-selected', nodeId === selected || Boolean(transition && (nodeId === transition.source || nodeId === transition.target)));
        });
        [...edgeLayer.querySelectorAll('.screen-edge'), ...labelsLayer.querySelectorAll('.screen-edge-label')].forEach((element: any) => {
            const transition: any = transitionByID(element.dataset.transitionId);
            const relatedToScreen: any = Boolean(selected && transition && (transition.source === selected || transition.target === selected));
            const relatedToTransition: any = Boolean(selectedTransition && element.dataset.transitionId === selectedTransition);
            element.classList.toggle('is-related', relatedToScreen || relatedToTransition);
            element.classList.toggle('is-muted', Boolean((selected && !relatedToScreen) || (selectedTransition && !relatedToTransition)));
        });
    }
    function selectScreen(id: any) {
        selected = id;
        selectedTransition = '';
        applySelectionStyles();
        renderInspector(id);
    }
    function selectTransition(id: any) {
        const transition: any = transitionByID(id);
        if (!transition)
            return;
        selected = '';
        selectedTransition = id;
        applySelectionStyles();
        inspector.innerHTML = `<div class="screen-inspector-head"><span>${escapeText(transition.type || 'navigation')}</span><button type="button" data-inspector-close>${escapeText(text("changes.close"))}</button></div>
<p class="screen-eyebrow">${escapeText(transition.id)}</p><h2>${escapeText(transition.action)}</h2>
<dl><div><dt>${escapeText(text("screenMap.condition"))}</dt><dd>${escapeText(transition.condition)}</dd></div><div><dt>${escapeText(text("screenMap.from"))}</dt><dd><code>${escapeText(transition.source)}</code></dd></div>
<div><dt>${escapeText(text("screenMap.to"))}</dt><dd><code>${escapeText(transition.target)}</code></dd></div><div><dt>${escapeText(text("field.useCase"))}</dt><dd>${escapeText(transition.useCase || text("features.screen-map.index.027"))}</dd></div>
<div><dt>${escapeText(text("screenMap.state"))}</dt><dd>${escapeText(transition.state || 'DEFAULT')}</dd></div><div><dt>${escapeText(text("screenMap.error"))}</dt><dd>${escapeText(transition.error || '—')}</dd></div></dl>
${transition.message ? `<div class="screen-transition-message"><strong>${escapeText(transition.error || text("features.screen-map.index.028"))}</strong><span>${escapeText(transition.message)}</span></div>` : ''}`;
        inspector.querySelector('[data-inspector-close]')?.addEventListener('click', () => selectScreen(''));
    }
    function measureVisibleCards() {
        nodeById.forEach((node: any, id: any) => {
            node.hidden = !visible.has(id);
            node.style.height = '';
            const change: any = changedNodes.get(id);
            node.classList.remove('is-change-added', 'is-change-modified', 'is-change-removed');
            if (change) {
                node.classList.add(`is-change-${change.status}`);
                node.dataset.changeStatus = changeStatusLabel(change.status);
                node.setAttribute('aria-label', `${node.dataset.screenNode}: ${changeStatusLabel(change.status)}`);
            }
            else {
                delete node.dataset.changeStatus;
            }
        });
        cardHeight = Math.max(MIN_CARD_HEIGHT, ...[...nodeById.entries()]
            .filter(([id]: any) => visible.has(id))
            .map(([, node]: any) => node.offsetHeight));
        nodeById.forEach((node: any, id: any) => {
            if (visible.has(id))
                node.style.height = `${cardHeight}px`;
        });
    }
    function render({ fit = false }: any = {}) {
        computeVisible();
        measureVisibleCards();
        layout();
        nodeById.forEach((node: any, id: any) => {
            const position: any = positions.get(id);
            if (position)
                node.style.transform = `translate(${position.x}px, ${position.y}px)`;
        });
        drawGroups();
        drawEdges();
        empty.hidden = visible.size > 0;
        if (selected && !visible.has(selected))
            selectScreen('');
        if (selectedTransition && !activeEdges.some((edge: any) => edge.id === selectedTransition))
            selectScreen('');
        if (fit)
            fitToStage();
        else
            applyTransform();
    }
    function fitToStage() {
        const width: any = parseFloat(viewport.style.width) || 900;
        const height: any = parseFloat(viewport.style.height) || 620;
        scale = Math.min(1, Math.max(.2, Math.min((stage.clientWidth - 48) / width, (stage.clientHeight - 48) / height)));
        panX = Math.max(24, (stage.clientWidth - width * scale) / 2);
        panY = Math.max(24, (stage.clientHeight - height * scale) / 2);
        applyTransform();
    }
    function setScale(next: any, originX: any = stage.clientWidth / 2, originY: any = stage.clientHeight / 2) {
        const previous: any = scale;
        scale = Math.min(2.4, Math.max(.2, next));
        panX = originX - (originX - panX) * (scale / previous);
        panY = originY - (originY - panY) * (scale / previous);
        applyTransform();
    }
    nodeById.forEach((node: any, id: any) => {
        node.addEventListener('click', (event: any) => {
            event.stopPropagation();
            selectScreen(id);
        });
        node.addEventListener('dblclick', () => {
            if (data.screenUrls?.[id])
                window.location.href = data.screenUrls[id];
        });
        node.addEventListener('keydown', (event: any) => {
            if (event.key === 'Enter') {
                event.preventDefault();
                if (selected === id && data.screenUrls?.[id])
                    window.location.href = data.screenUrls[id];
                else
                    selectScreen(id);
            }
        });
    });
    stage.addEventListener('click', (event: any) => {
        if (event.target === stage || event.target === viewport || event.target === nodesLayer)
            selectScreen('');
    });
    stage.addEventListener('wheel', (event: any) => {
        if (event.ctrlKey || event.metaKey)
            return;
        event.preventDefault();
        const box: any = stage.getBoundingClientRect();
        setScale(scale * (event.deltaY > 0 ? .9 : 1.1), event.clientX - box.left, event.clientY - box.top);
    }, { passive: false });
    stage.addEventListener('pointerdown', (event: any) => {
        if (event.target.closest('[data-screen-node], [data-transition-label]'))
            return;
        dragging = true;
        dragStart = { x: event.clientX, y: event.clientY, panX, panY };
        stage.setPointerCapture(event.pointerId);
        stage.classList.add('is-panning');
    });
    stage.addEventListener('pointermove', (event: any) => {
        if (!dragging)
            return;
        panX = dragStart.panX + event.clientX - dragStart.x;
        panY = dragStart.panY + event.clientY - dragStart.y;
        applyTransform();
    });
    stage.addEventListener('pointerup', () => {
        dragging = false;
        stage.classList.remove('is-panning');
    });
    stage.addEventListener('keydown', (event: any) => {
        if (event.target.matches('input,select'))
            return;
        if (event.key === '+' || event.key === '=')
            setScale(scale * 1.1);
        if (event.key === '-')
            setScale(scale * .9);
        if (event.key === '0')
            fitToStage();
        if (event.key === 'Escape')
            selectScreen('');
    });
    workspace.querySelectorAll('[data-map-mode]').forEach((button: any) => button.addEventListener('click', () => {
        mode = button.dataset.mapMode;
        workspace.querySelectorAll('[data-map-mode]').forEach((candidate: any) => {
            const active: any = candidate === button;
            candidate.classList.toggle('is-active', active);
            candidate.setAttribute('aria-pressed', String(active));
        });
        moduleSelect.hidden = mode !== 'module';
        useCaseSelect.hidden = mode !== 'usecase';
        render({ fit: true });
    }));
    moduleSelect?.addEventListener('change', () => render({ fit: true }));
    useCaseSelect?.addEventListener('change', () => render({ fit: true }));
    statusSelect?.addEventListener('change', () => render({ fit: true }));
    search?.addEventListener('input', () => render({ fit: true }));
    workspace.querySelector('[data-map-zoom-in]')?.addEventListener('click', () => setScale(scale * 1.1));
    workspace.querySelector('[data-map-zoom-out]')?.addEventListener('click', () => setScale(scale * .9));
    workspace.querySelector('[data-map-fit]')?.addEventListener('click', fitToStage);
    workspace.querySelector('[data-map-reset]')?.addEventListener('click', () => {
        scale = 1;
        panX = 24;
        panY = 24;
        applyTransform();
    });
    workspace.querySelector('[data-map-fullscreen]')?.addEventListener('click', () => {
        if (document.fullscreenElement)
            document.exitFullscreen?.();
        else
            stage.requestFullscreen?.();
    });
    function changeStatusLabel(value: any) {
        return ({ added: text("features.screen-map.index.012"), modified: text("features.screen-map.index.013"), removed: text("features.screen-map.index.014") } as Record<string, string>)[value] || text("features.screen-map.index.015");
    }
    function changeSnapshot(value: any) {
        return value?.after || value?.before || {};
    }
    function addGhostNode(node: any) {
        if (!node?.id || byId.has(node.id))
            return;
        const screen: any = {
            id: node.id, title: node.title || node.id, route: node.route || text("features.screen-map.index.016"), module: node.module || text("features.screen-map.index.017"),
            status: { kind: node.status || 'removed', label: text("features.screen-map.index.018") }, states: [], useCases: [], workItems: [], contracts: [], _changeGhost: true,
        };
        screens.push(screen);
        byId.set(screen.id, screen);
        const card: any = document.createElement('article');
        card.className = 'screen-node is-change-removed';
        card.dataset.screenNode = screen.id;
        card.tabIndex = 0;
        card.setAttribute('role', 'button');
        card.innerHTML = `<div class="screen-preview-placeholder"><strong>${escapeText(screen.id)}</strong><span>${escapeText(text("screenMap.deletedScreen"))}</span></div><div class="screen-node-copy"><strong>${escapeText(screen.id)}</strong><span>${escapeText(screen.title)}</span><small>${escapeText(screen.route)}</small><div class="screen-node-meta"><span class="screen-module-label">${escapeText(screen.module)}</span></div></div>`;
        nodeById.set(screen.id, card);
        nodesLayer.append(card);
        card.addEventListener('click', (event: any) => { event.stopPropagation(); selectScreen(screen.id); });
        card.addEventListener('keydown', (event: any) => {
            if (event.key === 'Enter') {
                event.preventDefault();
                selectScreen(screen.id);
            }
        });
    }
    async function loadChanges() {
        if (changesLoaded)
            return true;
        if (!changesAPI)
            return false;
        try {
            const response: any = await fetch(`${changesAPI}/screen-map`, { headers: { Accept: 'application/json' } });
            if (!response.ok)
                throw new Error(`HTTP ${response.status}`);
            const payload: any = await response.json();
            (payload.nodes || []).forEach((change: any) => {
                if (!change.id)
                    return;
                const status: any = change.status === 'deleted' ? 'removed' : (change.status === 'added' || change.status === 'untracked' ? 'added' : 'modified');
                changedNodes.set(change.id, { ...change, status });
                if (status === 'removed')
                    addGhostNode(changeSnapshot(change));
            });
            (payload.edges || []).forEach((change: any) => {
                const edge: any = changeSnapshot(change);
                const status: any = change.status === 'deleted' ? 'removed' : (change.status === 'added' || change.status === 'untracked' ? 'added' : 'modified');
                if (!edge.id)
                    return;
                changedEdges.set(edge.id, { ...change, status });
                if (status === 'removed' && !transitions.some((item: any) => item.id === edge.id))
                    transitions.push({ ...edge, type: 'navigation', _changeGhost: true });
            });
            changesLoaded = true;
            return true;
        }
        catch (error: any) {
            if (status)
                status.textContent = text("features.screen-map.index.020", [error.message]);
            return false;
        }
    }
    changesToggle?.addEventListener('click', async () => {
        if (!changesActive && !(await loadChanges()))
            return;
        changesActive = !changesActive;
        changesToggle.classList.toggle('is-active', changesActive);
        changesToggle.setAttribute('aria-pressed', String(changesActive));
        changesToggle.textContent = changesActive ? text("features.screen-map.index.021") : text("features.screen-map.index.022");
        changeStatusSelect.hidden = !changesActive;
        render({ fit: true });
    });
    changeStatusSelect?.addEventListener('change', () => render({ fit: true }));
    document.addEventListener('toudocu:panelshown', (event: any) => {
        if (event.target?.contains(workspace)) {
            window.requestAnimationFrame(() => render({ fit: true }));
        }
    }, signal ? { signal } : undefined);
    if (initialUseCase && useCaseSelect) {
        useCaseSelect.value = initialUseCase;
    }
    const hash: any = new URLSearchParams(window.location.hash.replace(/^#/, ''));
    if (!initialUseCase && hash.get('usecase')) {
        mode = 'usecase';
        useCaseSelect.value = hash.get('usecase');
        useCaseSelect.hidden = false;
        workspace.querySelectorAll('[data-map-mode]').forEach((button: any) => {
            const active: any = button.dataset.mapMode === 'usecase';
            button.classList.toggle('is-active', active);
            button.setAttribute('aria-pressed', String(active));
        });
    }
    render({ fit: true });
    if (hash.get('screen'))
        selectScreen(hash.get('screen'));
};
