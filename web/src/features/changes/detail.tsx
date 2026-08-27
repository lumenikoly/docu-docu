import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from "react";
import { text } from "../../core/locale";
import { detailCacheKey, lineDiff, type Change } from "./model";

type Detail = Change & {
  before?: string;
  current?: string;
  renderedBefore?: string;
  renderedCurrent?: string;
};
const cache = new Map<string, Detail>();
const rangeParams = () => {
  const source = new URLSearchParams(location.search);
  const params = new URLSearchParams();
  for (const key of ["base", "branchBase", "target"])
    if (source.has(key)) params.set(key, source.get(key)!);
  return params;
};
const endpoint = (
  base: string,
  path: string,
  change: Change,
  extra: Record<string, string> = {},
) => {
  const params = rangeParams();
  params.set("path", change.path);
  for (const [key, value] of Object.entries(extra)) params.set(key, value);
  return `${base}${path}?${params}`;
};
const languageFor = (path: string) =>
  path.endsWith(".json")
    ? "json"
    : /\.ya?ml$/i.test(path)
      ? "yaml"
      : path.endsWith(".go")
        ? "go"
        : path.endsWith(".java")
          ? "java"
          : /\.(js|jsx|mjs|cjs)$/i.test(path)
            ? "javascript"
            : /\.(ts|tsx|mts|cts)$/i.test(path)
              ? "typescript"
              : /\.md$/i.test(path)
                ? "markdown"
                : "text";
const statusLabel = (status: string) =>
  text(
    (
      {
        added: "features.changes.index.002",
        untracked: "features.changes.index.153",
        modified: "features.changes.index.003",
        deleted: "features.changes.index.004",
        renamed: "features.changes.index.005",
        copied: "features.changes.index.006",
        "type-changed": "features.changes.index.007",
        linked: "features.changes.index.091",
      } as Record<string, string>
    )[status] || status,
  );

function useDetail(change: Change) {
  const API = window.ToudocuPage?.endpoints?.changes || "";
  const REVIEW = window.ToudocuPage?.capabilities?.review;
  const key = detailCacheKey(rangeParams(), change);
  const [detail, setDetail] = useState<Detail | null>(
    () => cache.get(key) || null,
  );
  const [error, setError] = useState("");
  useEffect(() => {
    const cached = cache.get(key);
    if (cached) {
      setDetail(cached);
      setError("");
      return;
    }
    const controller = new AbortController();
    setDetail(null);
    setError("");
    const load = async () => {
      if (!REVIEW) {
        const result = { ...change } as Detail;
        cache.set(key, result);
        return setDetail(result);
      }
      const response = await fetch(
        endpoint(`${API}/review`, "/repository/file", change),
        { cache: "no-store", signal: controller.signal },
      );
      const data = await response.json();
      if (!response.ok)
        throw new Error(
          data.diagnostics?.[0]?.message || `HTTP ${response.status}`,
        );
      const result = {
        ...change,
        ...data.file,
        documentation: change.documentation,
        diagnostics: change.diagnostics,
        renderedDiffAvailable: change.renderedDiffAvailable,
        semanticDiffAvailable: change.semanticDiffAvailable,
        semanticChanges: change.semanticChanges,
        relationChanges: change.relationChanges,
        before: data.before,
        current: data.current,
        renderedBefore: data.renderedBefore,
        renderedCurrent: data.renderedCurrent,
        sourceDiff: data.patch,
        sourceDiffHunks: data.hunks,
      } as Detail;
      cache.set(key, result);
      setDetail(result);
    };
    void load().catch(
      (failure) => failure.name !== "AbortError" && setError(failure.message),
    );
    return () => controller.abort();
  }, [API, REVIEW, change, key]);
  return { detail, error };
}

export const tabsFor = (change: Change) => [
  ["source", text("features.changes.index.022")],
  ...(!change.binary && !change.asset
    ? [["file", text("features.changes.index.133")]]
    : []),
  ...(change.renderedDiffAvailable
    ? [["rendered", text("features.changes.index.023")]]
    : []),
  ...(change.semanticDiffAvailable
    ? [["semantic", text("features.changes.index.024")]]
    : []),
  ...(change.documentation || change.classification !== "repository-file"
    ? [["relations", text("features.changes.index.025")]]
    : []),
  ...(change.classification === "contract" ? [["openapi", "OpenAPI"]] : []),
  ...(change.mermaidBlocks?.length ? [["mermaid", "Mermaid"]] : []),
  ...(change.classification === "asset"
    ? [["assets", text("features.changes.index.154")]]
    : []),
  ...([...(change.entitiesBefore || []), ...(change.entitiesAfter || [])].some(
    (item) => item.type === "screen" || item.type === "transition",
  )
    ? [["map", text("features.changes.index.026")]]
    : []),
];

function Viewer({
  content,
  language,
  onSelect,
}: {
  content: string;
  language: string;
  onSelect?: (selection: any) => void;
}) {
  const host = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!host.current) return;
    const viewer = window.ToudocuCodeMirror?.createViewer?.({
      parent: host.current,
      doc: content,
      language,
      onSelect,
    });
    const theme = (event: Event) =>
      viewer?.setTheme?.((event as CustomEvent).detail.theme);
    document.addEventListener("toudocu:themechange", theme);
    if (!viewer) host.current.textContent = content;
    return () => {
      document.removeEventListener("toudocu:themechange", theme);
      viewer?.destroy?.();
      host.current?.replaceChildren();
    };
  }, [content, language, onSelect]);
  return <div data-file-view ref={host} />;
}

function SelectionMenu({
  selection,
  path,
  onQuestion,
  agentQuestion,
}: {
  selection: string;
  path: string;
  onQuestion?: () => void;
  agentQuestion?: boolean;
}) {
  if (!selection) return null;
  return (
    <div
      className="review-selection-menu"
      role="toolbar"
      aria-label={text("features.changes.index.137")}
    >
      <button
        type="button"
        data-selection-copy
        onClick={() => void navigator.clipboard.writeText(selection)}
      >
        {text("core.portal.035")}
      </button>
      <button
        type="button"
        data-selection-context
        onClick={() =>
          void navigator.clipboard.writeText(
            text("core.portal.038", [path.split("/").pop(), path, selection]),
          )
        }
      >
        {text("core.portal.036")}
      </button>
      <button
        type="button"
        data-selection-question
        disabled={!agentQuestion && !onQuestion}
        data-discussion-compose-available={(Boolean(onQuestion) && !agentQuestion) || undefined}
        onClick={agentQuestion
          ? () => document.dispatchEvent(new CustomEvent("toudocu:agent-compose", { detail: { text: `Explain this selection from ${path}:\n\n${selection}`, policy: "filesystem-read-only" } }))
          : onQuestion}
      >
        {text("core.portal.037")}
      </button>
    </div>
  );
}

function FileView({
  detail,
  onCompose,
}: {
  detail: Detail;
  onCompose?: (target: any, returnElement?: HTMLElement | null) => void;
}) {
  const deleted = detail.status === "deleted";
  const content = deleted ? detail.before || "" : detail.current || "";
  const [selection, setSelection] = useState<{
    text: string;
    range: any;
  } | null>(null);
  const selected = useCallback((range: any) => {
    const value = window.getSelection()?.toString() || "";
    setSelection(range && value ? { text: value, range } : null);
  }, []);
  return (
    <>
      {deleted && (
        <p className="changes-absence">
          {text("changes.showingDeletedVersion")}
        </p>
      )}
      <Viewer
        content={content}
        language={detail.language || languageFor(detail.path)}
        onSelect={selected}
      />
      <SelectionMenu
        selection={selection?.text || ""}
        path={detail.path}
        agentQuestion
        onQuestion={
          onCompose && selection
            ? () =>
                onCompose(
                  {
                    type: "fileRange",
                    path: detail.path,
                    start: selection.range.start,
                    end: selection.range.end,
                  },
                  document.querySelector<HTMLElement>(
                    "[data-file-view] .cm-content",
                  ),
                )
            : undefined
        }
      />
    </>
  );
}

function DiffLine({
  line,
  counters,
  writable,
  path,
  onCompose,
}: {
  line: string;
  counters: { old: number; next: number };
  writable: boolean;
  path: string;
  onCompose?: (target: any, element?: HTMLElement | null) => void;
}) {
  const marker = line[0] || " ";
  if (![" ", "-", "+"].includes(marker))
    return (
      <span className="diff-line diff-line-context">
        <span />
        <span className="diff-line-number" />
        <span className="diff-line-number" />
        <span className="diff-line-marker">{marker}</span>
        <span className="diff-line-content">{line.slice(1)}</span>
      </span>
    );
  let oldLine: number | "" = "",
    newLine: number | "" = "";
  if (marker === " ") {
    oldLine = counters.old++;
    newLine = counters.next++;
  } else if (marker === "-") oldLine = counters.old++;
  else newLine = counters.next++;
  const side = marker === "-" ? "old" : "new";
  const selectedLine = side === "old" ? oldLine : newLine;
  return (
    <span
      className={`diff-line diff-line-${marker === "+" ? "added" : marker === "-" ? "removed" : "context"}`}
      data-review-side={side}
      data-review-line={selectedLine}
    >
      {writable ? (
        <button
          type="button"
          className="diff-comment"
          aria-label={text(
            side === "old"
              ? "features.changes.index.096"
              : "features.changes.index.097",
            [selectedLine],
          )}
          onClick={(event) =>
            onCompose?.(
              side === "old"
                ? {
                    kind: "file",
                    path,
                    quote: `> ${path}:${selectedLine}\n> ${line.slice(1)}`,
                  }
                : {
                    type: "diff",
                    path,
                    side,
                    start: { line: selectedLine, column: 1 },
                    end: {
                      line: selectedLine,
                      column: Array.from(line.slice(1)).length + 1,
                    },
                  },
              event.currentTarget,
            )
          }
        >
          +
        </button>
      ) : (
        <span />
      )}
      <span className="diff-line-number">{oldLine}</span>
      <span className="diff-line-number">{newLine}</span>
      <span className="diff-line-marker">{marker}</span>
      <span className="diff-line-content">{line.slice(1)}</span>
    </span>
  );
}

function SourceView({
  detail,
  writable,
  onCompose,
}: {
  detail: Detail;
  writable: boolean;
  onCompose?: (target: any, element?: HTMLElement | null) => void;
}) {
  const [mode, setMode] = useState("unified");
  const host = useRef<HTMLDivElement>(null);
  const merge = useRef<any>(null);
  const [selection, setSelection] = useState<{
    text: string;
    target?: any;
    returnElement?: HTMLElement | null;
  }>({ text: "" });
  useEffect(() => {
    if (mode !== "merge" || !host.current) return;
    merge.current = window.ToudocuCodeMirror?.createMerge?.({
      parent: host.current,
      before: detail.before || "",
      after: detail.current || "",
      language: detail.language || languageFor(detail.path),
      onSelect: (range: any) =>
        setSelection({
          text: window.getSelection()?.toString() || "",
          target: range && {
            type: "diff",
            path: detail.path,
            side: range.side,
            start: range.start,
            end: range.end,
          },
          returnElement: host.current,
        }),
    });
    const theme = (event: Event) =>
      merge.current?.setTheme?.((event as CustomEvent).detail.theme);
    document.addEventListener("toudocu:themechange", theme);
    return () => {
      document.removeEventListener("toudocu:themechange", theme);
      merge.current?.destroy?.();
      merge.current = null;
      host.current?.replaceChildren();
    };
  }, [detail, mode]);
  const hunks = (detail.sourceDiffHunks || []) as any[];
  const selectUnified = () => {
    const current = window.getSelection();
    if (!current || current.isCollapsed || !current.rangeCount)
      return setSelection({ text: "" });
    const range = current.getRangeAt(0);
    const element = (node: Node) =>
      node.nodeType === Node.TEXT_NODE
        ? (node.parentElement as HTMLElement | null)
        : (node as HTMLElement);
    const start = element(range.startContainer)?.closest<HTMLElement>(
      "[data-review-line]",
    );
    const end = element(range.endContainer)?.closest<HTMLElement>(
      "[data-review-line]",
    );
    if (!start || !end) return setSelection({ text: current.toString() });
    const rows = [
      ...start
        .closest("[data-source-view]")!
        .querySelectorAll<HTMLElement>("[data-review-line]"),
    ];
    const firstIndex = rows.indexOf(start);
    const lastIndex = rows.indexOf(end);
    const value = rows
      .slice(firstIndex, lastIndex + 1)
      .map((row) => row.querySelector(".diff-line-content")?.textContent || "")
      .join("\n");
    if (!value) return setSelection({ text: "" });
    const side =
      start.dataset.reviewSide === end.dataset.reviewSide
        ? start.dataset.reviewSide
        : "mixed";
    const first = Number(start.dataset.reviewLine);
    const last = Number(end.dataset.reviewLine);
    const target =
      writable && side === "new"
        ? {
            type: "diff",
            path: detail.path,
            side,
            start: { line: first, column: 1 },
            end: {
              line: last,
              column:
                Array.from(
                  end.querySelector(".diff-line-content")?.textContent || "",
                ).length + 1,
            },
          }
        : writable && side === "old"
          ? {
              kind: "file",
              path: detail.path,
              quote: `> ${detail.path}:${first}${first === last ? "" : `–${last}`}\n> ${value.replace(/\n/g, "\n> ")}`,
            }
          : undefined;
    setSelection({
      text: value,
      target,
      returnElement: start.querySelector<HTMLElement>(".diff-line-content"),
    });
  };
  return (
    <>
      <div className="source-actions">
        <div
          className="source-mode-toggle"
          role="group"
          aria-label={text("changes.comparisonMode")}
        >
          <button
            type="button"
            data-source-mode="unified"
            aria-pressed={mode === "unified"}
            onClick={() => setMode("unified")}
          >
            {text("changes.unified")}
          </button>
          <button
            type="button"
            data-source-mode="merge"
            aria-pressed={mode === "merge"}
            onClick={() => setMode("merge")}
          >
            {text("changes.sideBySide")}
          </button>
        </div>
        <button
          type="button"
          className="changes-button tertiary"
          data-copy-diff
          onClick={() =>
            void navigator.clipboard.writeText(detail.sourceDiff || "")
          }
        >
          {text("changes.copyDiff")}
        </button>
      </div>
      <div
        data-source-view
        ref={mode === "merge" ? host : undefined}
        onPointerUp={selectUnified}
      >
        {mode === "unified" &&
          (hunks.length ? (
            hunks.map((hunk, index) => {
              const lines = hunk.patch.split("\n");
              const header = lines.shift();
              if (lines.at(-1) === "") lines.pop();
              const counters = { old: hunk.oldStart, next: hunk.newStart };
              return (
                <article
                  className="changes-hunk"
                  id={hunk.id}
                  tabIndex={-1}
                  key={hunk.id}
                >
                  <header>
                    <a
                      href={`#${hunk.id}`}
                      aria-label={text("changes.hunkLink", [index + 1])}
                    >
                      {header}
                    </a>
                    <span>
                      −{hunk.oldStart},{hunk.oldLines} +{hunk.newStart},
                      {hunk.newLines}
                    </span>
                    <button
                      type="button"
                      className="changes-button secondary"
                      data-copy-hunk
                      onClick={() =>
                        void navigator.clipboard.writeText(hunk.patch)
                      }
                    >
                      {text("changes.copyHunk")}
                    </button>
                  </header>
                  <pre>
                    {lines.map((line: string, lineIndex: number) => (
                      <DiffLine
                        key={lineIndex}
                        line={line}
                        counters={counters}
                        writable={writable}
                        path={detail.path}
                        onCompose={onCompose}
                      />
                    ))}
                  </pre>
                </article>
              );
            })
          ) : (
            <pre className="changes-diff">
              {detail.sourceDiff || text("features.changes.index.031")}
            </pre>
          ))}
      </div>
      <SelectionMenu
        selection={selection.text}
        path={detail.path}
        onQuestion={
          onCompose && selection.target
            ? () => onCompose(selection.target, selection.returnElement)
            : undefined
        }
      />
    </>
  );
}

function Rendered({ detail }: { detail: Detail }) {
  const host = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const documents =
      host.current?.querySelectorAll<HTMLElement>(".rendered-document");
    if (!documents?.length) return;
    const content = (root: HTMLElement | undefined, anchor: string) => { const heading = anchor ? root?.querySelector<HTMLElement>(`#${CSS.escape(anchor)}`) : null; const blocks: Element[] = []; for (let node = heading?.nextElementSibling; node && node.tagName !== "H2"; node = node.nextElementSibling) blocks.push(node); return { heading, blocks }; };
    const unmatched = (source: Element[], target: Element[]) => { const remaining = new Map<string, number>(); for (const item of target) remaining.set(item.outerHTML, (remaining.get(item.outerHTML) || 0) + 1); return source.filter((item) => { const count = remaining.get(item.outerHTML) || 0; if (!count) return true; remaining.set(item.outerHTML, count - 1); return false; }); };
    for (const section of (detail.renderedSections || []) as any[]) {
      if (section.status === "unchanged-section") continue;
      const before = content(documents[0], section.anchorBefore); const after = content(documents[1], section.anchorAfter);
      [before, after].forEach((part) => {
          const heading = part.heading;
          if (!heading) return;
          heading.classList.add("rendered-section-heading", section.status);
          heading.dataset.changeLabel = section.status
            .replace("-section", "")
            .replace("-", " ");
      });
      const beforeChanged = section.status === "modified-section" ? unmatched(before.blocks, after.blocks) : before.blocks;
      const afterChanged = section.status === "modified-section" ? unmatched(after.blocks, before.blocks) : after.blocks;
      for (const node of [...beforeChanged, ...afterChanged]) node.classList.add("rendered-section-content", section.status);
    }
  }, [detail]);
  if (detail.renderedBefore == null && detail.renderedCurrent == null)
    return <div className="changes-error">{text("changes.renderedDiffFailed", [""])}</div>;
  return (
    <div className="rendered-columns" ref={host}>
      <section>
        <h3>{text("changes.before")}</h3>
        {detail.renderedBefore ? (
          <div
            className="rendered-document"
            dangerouslySetInnerHTML={{ __html: detail.renderedBefore }}
          />
        ) : (
          <p className="changes-absence">{text("changes.documentMissing")}</p>
        )}
      </section>
      <section>
        <h3>{text("changes.after")}</h3>
        {detail.renderedCurrent ? (
          <div
            className="rendered-document"
            dangerouslySetInnerHTML={{ __html: detail.renderedCurrent }}
          />
        ) : (
          <p className="changes-absence">{text("changes.documentDeleted")}</p>
        )}
      </section>
    </div>
  );
}

function Semantic({ detail }: { detail: Detail }) {
  const items = (detail.semanticChanges || []) as any[];
  return items.length ? (
    <ol className="semantic-list">
      {items.map((item, index) => (
        <li key={index}>
          <div>
            <strong>{item.kind}</strong>
            {item.compatibility && (
              <span className={`compatibility ${item.compatibility}`}>
                {item.compatibility}
              </span>
            )}
          </div>
          <p>{item.summary}</p>
          {(item.before !== undefined || item.after !== undefined) && (
            <div className="semantic-values">
              <pre>{JSON.stringify(item.before, null, 2) || "—"}</pre>
              <pre>{JSON.stringify(item.after, null, 2) || "—"}</pre>
            </div>
          )}
        </li>
      ))}
    </ol>
  ) : (
    <div className="changes-empty">
      <h3>{text("changes.noSemanticChanges")}</h3>
      <p>{text("changes.noSemanticChangesHelp")}</p>
    </div>
  );
}
function Relations({ detail }: { detail: Detail }) {
  const items = (detail.relationChanges || []) as any[];
  return items.length ? (
    <ul>
      {items.map((item, index) => (
        <li key={index}>
          {item.kind}: {item.source?.id} → {item.target?.id}
        </li>
      ))}
    </ul>
  ) : (
    <div className="changes-empty">
      <h3>{text("changes.noRelationChanges")}</h3>
      <p>{text("changes.noRelationChangesHelp")}</p>
    </div>
  );
}
function Mermaid({ detail }: { detail: Detail }) {
  const host = useRef<HTMLDivElement>(null);
  const pointer = useRef<{ id: number; x: number; y: number; startX: number; startY: number; block: string } | null>(null);
  const [views, setViews] = useState<Record<string, { zoom: number; x: number; y: number }>>({});
  const blocks = (detail.mermaidBlocks || []) as any[];
  useEffect(() => {
    if (!window.mermaid || !host.current) return;
    let active = true;
    window.mermaid.initialize({
      startOnLoad: false,
      securityLevel: "strict",
      theme:
        document.documentElement.dataset.theme === "dark" ? "dark" : "default",
    });
    for (const diagram of host.current.querySelectorAll(".mermaid"))
      void window.mermaid
        .run({ nodes: [diagram] })
        .catch(() => active && diagram.isConnected &&
          diagram.insertAdjacentHTML(
            "afterend",
            `<p class="changes-error">${text("changes.mermaidVersionFailed")}</p>`,
          ),
        );
    return () => { active = false; pointer.current = null; };
  }, [detail]);
  if (!blocks.length)
    return (
      <div className="changes-empty">
        <h3>{text("changes.noMermaidChanges")}</h3>
        <p>{text("changes.noMermaidChangesHelp")}</p>
      </div>
    );
  return (
    <div ref={host}>
      {blocks.map((block) => { const view = views[block.id] || { zoom: 1, x: 0, y: 0 }; const update = (next: Partial<typeof view>) => setViews((current) => ({ ...current, [block.id]: { ...view, ...next } })); const canvas = { style: { "--mermaid-zoom": view.zoom, "--mermaid-x": `${view.x}px`, "--mermaid-y": `${view.y}px` } as CSSProperties, onPointerDown: (event: ReactPointerEvent<HTMLDivElement>) => { pointer.current = { id: event.pointerId, x: event.clientX, y: event.clientY, startX: view.x, startY: view.y, block: block.id }; event.currentTarget.setPointerCapture?.(event.pointerId); event.currentTarget.classList.add("is-panning"); }, onPointerMove: (event: ReactPointerEvent<HTMLDivElement>) => { const drag = pointer.current; if (!drag || drag.id !== event.pointerId || drag.block !== block.id) return; update({ x: drag.startX + event.clientX - drag.x, y: drag.startY + event.clientY - drag.y }); }, onPointerUp: (event: ReactPointerEvent<HTMLDivElement>) => { pointer.current = null; event.currentTarget.classList.remove("is-panning"); } }; return (
        <section className="mermaid-change" key={block.id}>
          <header>
            <strong>{block.caption || block.id}</strong>
            <span className={`changes-file-status status-${block.status}`}>
              {statusLabel(block.status)}
            </span>
          </header>
          <div
            className="mermaid-controls"
            role="group"
            aria-label={text("changes.diagramControls")}
          >
            <button type="button" className="changes-button secondary" data-mermaid-zoom="out" onClick={() => update({ zoom: Math.max(.5, view.zoom - .2) })}>{text("screen.zoomOut")}</button>
            <button type="button" className="changes-button secondary" data-mermaid-zoom="reset" onClick={() => update({ zoom: 1, x: 0, y: 0 })}>100%</button>
            <button type="button" className="changes-button secondary" data-mermaid-zoom="in" onClick={() => update({ zoom: Math.min(2.5, view.zoom + .2) })}>{text("screen.zoomIn")}</button>
            <button
              type="button"
              className="changes-button secondary"
              onClick={(event) =>
                event.currentTarget.closest("section")?.requestFullscreen?.()
              }
            >
              {text("changes.fullscreen")}
            </button>
          </div>
          <div className="rendered-columns">
            <section>
              <h3>{text("changes.diagramBefore")}</h3>
              {block.before ? (
                <div className="mermaid-canvas" {...canvas}>
                  <pre className="mermaid">{block.before}</pre>
                </div>
              ) : (
                <p className="changes-absence">
                  {text("changes.diagramMissing")}
                </p>
              )}
            </section>
            <section>
              <h3>{text("changes.diagramAfter")}</h3>
              {block.after ? (
                <div className="mermaid-canvas" {...canvas}>
                  <pre className="mermaid">{block.after}</pre>
                </div>
              ) : (
                <p className="changes-absence">
                  {text("changes.diagramDeleted")}
                </p>
              )}
            </section>
          </div>
          <h3>{text("changes.mermaidSourceDiff")}</h3>
          <pre className="changes-diff mermaid-source-diff">{lineDiff(block.before || "", block.after || "") || text("features.changes.index.044")}</pre>
        </section>
      );})}
    </div>
  );
}
function Assets({ detail }: { detail: Detail }) {
  const [position, setPosition] = useState(50);
  const asset = detail.asset as any;
  const base = window.ToudocuPage?.endpoints?.changes || "";
  const url = (side: string) => endpoint(base, "/content", detail, { side });
  const before = detail.status !== "added" && detail.status !== "untracked";
  const after = detail.status !== "deleted";
  const overlay =
    before &&
    after &&
    asset?.before?.mediaType !== "image/svg+xml" &&
    asset?.after?.mediaType !== "image/svg+xml";
  const meta = (side: any, bytes: unknown) => side ? text("changes.assetMeta", [side.width || "?", side.height || "?", side.aspectRatio || "?", bytes || 0, side.transparency == null ? "" : ` · ${text(side.transparency ? "changes.alpha" : "changes.opaque")}`]) : text("changes.assetBytes", [bytes || 0]);
  return (
    <>
      <div className="rendered-columns asset-columns">
        <section>
          <h3>{text("changes.before")} · {meta(asset?.before, detail.oldSize)}</h3>
          {before ? (
            <img
              src={url("before")}
              alt={text("changes.oldVersion", [detail.path])}
            />
          ) : (
            <p className="changes-absence">{text("changes.assetMissing")}</p>
          )}
        </section>
        <section>
          <h3>{text("changes.after")} · {meta(asset?.after, detail.newSize)}</h3>
          {after ? (
            <img
              src={url("after")}
              alt={text("changes.newVersion", [detail.path])}
            />
          ) : (
            <p className="changes-absence">{text("changes.assetDeleted")}</p>
          )}
        </section>
      </div>
      {overlay && (
        <section className="asset-overlay">
          <h3>{text("changes.overlay")}</h3>
          <div className="asset-overlay-stage">
            <img src={url("before")} alt={text("changes.oldVersionShort")} />
            <div
              data-overlay-after
              style={{ clipPath: `inset(0 0 0 ${position}%)` }}
            >
              <img src={url("after")} alt={text("changes.newVersionShort")} />
            </div>
            <span data-overlay-divider style={{ left: `${position}%` }} />
          </div>
          <label>
            <span>{text("changes.dividerPosition")}</span>
            <input
              type="range"
              min="0"
              max="100"
              value={position}
              data-overlay-range
              onChange={(event) => setPosition(Number(event.target.value))}
            />
          </label>
        </section>
      )}
    </>
  );
}
function MapView({ detail }: { detail: Detail }) {
  const screen = detail.screen as any;
  if (!screen)
    return (
      <div className="changes-empty">
        <h3>{text("changes.screenDiffUnavailable")}</h3>
        <p>{text("changes.screenDiffUnavailableHelp")}</p>
      </div>
    );
  const node = screen.after || screen.before;
  return (
    <div className="map-change-preview">
      <h3>{text("changes.screenMapChanges")}</h3>
      <div
        className={`map-change-node status-${detail.status}${screen.after ? "" : " is-ghost"}`}
      >
        <strong>{node?.id}</strong>
        <span>{statusLabel(detail.status)}</span>
        <small>
          {node?.title}
          {node?.route ? ` · ${node.route}` : ""}
        </small>
      </div>
      <h4>{text("changes.transitions")}</h4>
      {screen.transitions?.length ? (
        <ol className="map-change-edges">
          {screen.transitions.map((item: any) => (
            <li
              className={`map-change-edge status-${item.status}`}
              key={item.id}
            >
              <header>
                <strong>{item.id}</strong>
                <span>{statusLabel(item.status)}</span>
              </header>
              <div className="map-edge-values">
                <span>
                  {item.before
                    ? `${item.before.source} → ${item.before.target}`
                    : "—"}
                </span>
                <span>
                  {item.after
                    ? `${item.after.source} → ${item.after.target}`
                    : "—"}
                </span>
              </div>
              {(item.after?.action || item.before?.action) && <p>{item.after?.action || item.before?.action} · {item.after?.condition || item.before?.condition || ""}</p>}
            </li>
          ))}
        </ol>
      ) : (
        <p>{text("changes.transitionsUnchanged")}</p>
      )}
      <p><a href={endpoint(window.ToudocuPage?.endpoints?.changes || "", "/screen-map", detail)}>{text("changes.openChangedMapJSON")}</a></p>
    </div>
  );
}

export function ChangesDetail({
  change,
  tab,
  onTab,
  onCompose,
}: {
  change: Change;
  tab: string;
  onTab: (tab: string) => void;
  onCompose?: (target: any, element?: HTMLElement | null) => void;
}) {
  const { detail, error } = useDetail(change);
  const writable = Boolean(
    window.ToudocuPage?.capabilities?.review &&
    (new URLSearchParams(location.search).get("target") || "working-tree") ===
      "working-tree",
  );
  const tabs = tabsFor(detail || change);
  const active = tabs.some(([id]) => id === tab) ? tab : "source";
  useEffect(() => {
    const host = document.querySelector<HTMLElement>("[data-detail]");
    if (!host) return;
    if (!detail && !error) host.setAttribute("aria-busy", "true");
    else host.removeAttribute("aria-busy");
    return () => host.removeAttribute("aria-busy");
  }, [detail, error]);
  return (
    <>
      {
        <header className="changes-detail-header">
          <div>
            <span className={`changes-file-status status-${change.status}`}>
              {statusLabel(change.status)}
            </span>
            <h2>{change.path.split("/").pop()}</h2>
            <p>
              {change.oldPath
                ? `${change.oldPath} → ${change.path}`
                : change.path}{" "}
              · +{change.lines.added} −{change.lines.deleted}
            </p>
          </div>
          <div className="changes-detail-actions">
            {writable && change.status !== "linked" && (
              <button
                type="button"
                className="changes-button secondary"
                data-file-comment
                onClick={(event) =>
                  onCompose?.(
                    { kind: "file", path: change.path },
                    event.currentTarget,
                  )
                }
              >
                {text("features.changes.index.095")}
              </button>
            )}
          </div>
        </header>
      }
      {change.diagnostics?.length ? (
        <details
          className="changes-diagnostics"
          open={change.diagnostics.some((item) => item.severity === "error")}
        >
          <summary>
            {text("features.changes.index.155")} · {change.diagnostics.length}
          </summary>
          <ul>
            {change.diagnostics.map((item) => (
              <li className={`is-${item.severity}`} key={item.code}>
                <strong className="diagnostic-severity">{text(item.severity === "error" ? "health.error" : item.severity === "warning" ? "health.warning" : item.severity)}</strong> <code>{item.code}</code> {item.message}
              </li>
            ))}
          </ul>
        </details>
      ) : null}
      <nav
        className="changes-tabs"
        role="tablist"
        aria-label={text("changes.changeViews")}
      >
        {tabs.map(([id, label]) => (
          <button
            type="button"
            role="tab"
            aria-selected={active === id}
            data-tab={id}
            key={id}
            onClick={() => onTab(id)}
          >
            {label}
          </button>
        ))}
      </nav>
      <div className="changes-tab-panel" data-tab-panel>
        {error ? (
          <div className="changes-error">{error}</div>
        ) : !detail ? (
          <div
            className="changes-loading"
            data-ui-state="loading"
            role="status"
          >
            {text("changes.loadingFile")}
          </div>
        ) : active === "source" ? (
          <SourceView
            detail={detail}
            writable={writable}
            onCompose={onCompose}
          />
        ) : active === "file" ? (
          <FileView
            detail={detail}
            onCompose={writable ? onCompose : undefined}
          />
        ) : active === "rendered" ? (
          <Rendered detail={detail} />
        ) : active === "semantic" || active === "openapi" ? (
          <Semantic detail={detail} />
        ) : active === "relations" ? (
          <Relations detail={detail} />
        ) : active === "mermaid" ? (
          <Mermaid detail={detail} />
        ) : active === "assets" ? (
          <Assets detail={detail} />
        ) : (
          <MapView detail={detail} />
        )}
      </div>
    </>
  );
}
