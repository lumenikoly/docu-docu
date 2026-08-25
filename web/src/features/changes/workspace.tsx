import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { text } from "../../core/locale";
import {
  buildChangeTree,
  mergeLinkedFiles,
  rangeFromURL,
  rangeQuery,
  visibleChanges,
  type Change,
  type ChangeTree,
  type Filters,
  type RangeState,
} from "./model";

type Report = {
  comparison: {
    base: { displayRef: string; resolved: string };
    target: { displayRef: string; resolved?: string };
  };
  repository: { branch?: string; dirty?: boolean };
  summary: { lines: { added: number; deleted: number } };
  changes?: Change[];
  files?: Change[];
};
type LoadState = { report: Report; files: Change[]; etag: string };

const normalizeReviewFile = (file: Change): Change => {
  const documentation = (file.documentation || {}) as Record<string, any>;
  return {
    ...file,
    documentation: file.documentation || null,
    classification: documentation.classification || "repository-file",
    renderedDiffAvailable: Boolean(documentation.renderedDiffAvailable),
    semanticDiffAvailable: Boolean(documentation.semanticDiffAvailable),
    sourceDiff: documentation.sourceDiff || "",
    sourceDiffHunks: documentation.sourceDiffHunks || [],
    semanticChanges: documentation.semanticChanges || [],
    relationChanges: documentation.relationChanges || [],
    mermaidBlocks: documentation.mermaidBlocks || [],
    diagnostics: documentation.diagnostics || [],
    entitiesBefore: documentation.entitiesBefore || [],
    entitiesAfter: documentation.entitiesAfter || [],
  };
};

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
const statusCode = (status: string) =>
  (
    ({
      added: "A",
      untracked: "U",
      modified: "M",
      deleted: "D",
      renamed: "R",
      copied: "C",
      "type-changed": "T",
      linked: "L",
    }) as Record<string, string>
  )[status] || "?";

function Tree({
  node,
  selected,
  locale,
  onSelect,
}: {
  node: ChangeTree;
  selected?: string;
  locale: string;
  onSelect: (change: Change) => void;
}): ReactNode {
  return (
    <>
      {[...node.directories]
        .sort(([a], [b]) => a.localeCompare(b, locale))
        .map(([name, child]) => (
          <details className="changes-tree-folder" open key={name}>
            <summary>
              <span aria-hidden />
              <strong>{name}</strong>
            </summary>
            <Tree
              node={child}
              selected={selected}
              locale={locale}
              onSelect={onSelect}
            />
          </details>
        ))}
      {node.files.map((change) => (
        <button
          type="button"
          className={`changes-file${selected === change.path ? " is-active" : ""}`}
          data-path={change.path}
          title={change.path}
          key={change.path}
          onClick={() => onSelect(change)}
        >
          <span className="changes-file-icon" aria-hidden />
          <strong>{change.path.split("/").pop()}</strong>
          <span className="changes-line-stats">
            +{change.lines.added} −{change.lines.deleted}
          </span>
          <span
            className={`changes-file-status status-${change.status}`}
            aria-label={statusLabel(change.status)}
            title={statusLabel(change.status)}
          >
            {statusCode(change.status)}
          </span>
        </button>
      ))}
    </>
  );
}

function FilePicker({
  endpoint,
  onSelect,
}: {
  endpoint: string;
  onSelect: (change: Change) => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const controller = useRef<AbortController | null>(null);
  const [query, setQuery] = useState("");
  const [files, setFiles] = useState<
    Array<{ path: string; language?: string }>
  >([]);
  const [error, setError] = useState("");
  const load = async (value = query) => {
    controller.current?.abort();
    controller.current = new AbortController();
    const response = await fetch(
      `${endpoint}/repository/files?q=${encodeURIComponent(value)}&limit=50`,
      { cache: "no-store", signal: controller.current.signal },
    );
    const data = await response.json();
    if (!response.ok)
      throw new Error(
        data.diagnostics?.[0]?.message || `HTTP ${response.status}`,
      );
    setFiles(data.files || []);
  };
  const failed = (failure: Error) => { if (failure.name !== "AbortError") setError(failure.message); };
  useEffect(() => () => controller.current?.abort(), []);
  return (
    <>
      <button
        type="button"
        className="changes-linked-add"
        data-linked-file-open
        onClick={() => {
          dialog.current?.showModal();
          void load().catch(failed);
        }}
      >
        {text("changes.linkFile")}
      </button>
      <dialog ref={dialog} className="review-file-picker" data-file-picker onClose={() => controller.current?.abort()}>
        <form method="dialog">
          <header>
            <strong>{text("changes.linkFile")}</strong>
            <button value="cancel" aria-label={text("changes.close")}>
              {text("changes.close")}
            </button>
          </header>
          <label>
            {text("changes.pathOrTitle")}
            <input
              type="search"
              data-file-picker-query
              autoComplete="off"
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                void load(event.target.value).catch(failed);
              }}
            />
          </label>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <div data-file-picker-results>
            {files.map((file) => (
              <button
                type="button"
                data-link-path={file.path}
                key={file.path}
                onClick={() => {
                  onSelect({
                    path: file.path,
                    status: "linked",
                    language: file.language,
                    lines: { added: 0, deleted: 0 },
                  });
                  dialog.current?.close();
                }}
              >
                <strong>{file.path.split("/").pop()}</strong>
                <span>{file.path}</span>
              </button>
            ))}
          </div>
        </form>
      </dialog>
    </>
  );
}

export function ChangesWorkspace({
  renderDetail,
  onSelectionChange,
  linkedPaths = [],
}: {
  renderDetail: (
    change: Change | null,
    tab: string,
    onTab: (tab: string) => void,
  ) => ReactNode;
  onSelectionChange?: (change: Change | null) => void;
  linkedPaths?: string[];
}) {
  const page = window.ToudocuPage;
  const locale = page?.ui.locale || "en";
  const API =
    page?.runtime === "serve" && page.capabilities?.changes
      ? page.endpoints?.changes || ""
      : "";
  const REVIEW =
    page?.runtime === "serve" && page.capabilities?.review
      ? page.endpoints?.review || ""
      : "";
  const initial = new URLSearchParams(location.search);
  const [range, setRange] = useState(rangeFromURL);
  const [applied, setApplied] = useState(range);
  const [filters, setFilters] = useState<Filters>({
    query: initial.get("q") || "",
    status: initial.get("status") || "",
    scope: initial.get("scope") || "",
  });
  const [data, setData] = useState<LoadState | null>(null);
  const [manualLinked, setManualLinked] = useState<Change[]>([]);
  const [selected, setSelected] = useState<Change | null>(null);
  const selectedRef = useRef<Change | null>(null);
  const [tab, setTab] = useState(
    initial.get("tab") === "summary"
      ? "source"
      : initial.get("tab") || "source",
  );
  const [error, setError] = useState("");
  const [filesOpen, setFilesOpen] = useState(false);
  const [stale, setStale] = useState(false);
  const [openFileStale, setOpenFileStale] = useState(false);
  const request = useRef(0);
  const rangeSummary = useRef<HTMLElement>(null);
  const rangeDetails = useRef<HTMLDetailsElement>(null);
  const query = useCallback(
    (extra: Record<string, string> = {}) => {
      const params = rangeQuery(applied);
      for (const [key, value] of Object.entries(extra)) params.set(key, value);
      return params;
    },
    [applied],
  );
  const load = useCallback(
    async (signal: AbortSignal, preserve = true) => {
      const generation = ++request.current;
      const suffix = query().toString();
      const [baseResponse, reviewResponse] = await Promise.all([
        fetch(`${API}?${suffix}`, { cache: "no-store", signal }),
        REVIEW
          ? fetch(`${API}/review/repository/changes?${suffix}`, {
              cache: "no-store",
              signal,
            })
          : Promise.resolve(null),
      ]);
      const base = await baseResponse.json();
      if (!baseResponse.ok)
        throw new Error(
          base.diagnostics?.[0]?.message || `HTTP ${baseResponse.status}`,
        );
      const repository = reviewResponse ? await reviewResponse.json() : null;
      if (reviewResponse && !reviewResponse.ok)
        throw new Error(
          repository.diagnostics?.[0]?.message ||
            `HTTP ${reviewResponse.status}`,
        );
      if (generation !== request.current) return;
      const report = repository || base;
      const etag = reviewResponse?.headers.get("ETag") || baseResponse.headers.get("ETag") || "";
      const files: Change[] = (repository
        ? repository.files.map(normalizeReviewFile)
        : base.changes).map((item: Change) => ({ ...item, _revision: etag }));
      setData({
        report,
        files,
        etag,
      });
      setSelected((current) => {
        const requested =
          new URLSearchParams(location.search).get("path") ||
          (preserve ? current?.path : "");
        return (
          files.find(
            (item: Change) =>
              item.path === requested || item.oldPath === requested,
          ) ||
          files[0] ||
          null
        );
      });
      setError("");
    },
    [API, REVIEW, query],
  );
  useEffect(() => {
    if (!API) return;
    const controller = new AbortController();
    void load(controller.signal, false).catch(
      (failure) => failure.name !== "AbortError" && setError(failure.message),
    );
    return () => controller.abort();
  }, [API, load]);
  useEffect(() => {
    selectedRef.current = selected;
  }, [selected]);
  useEffect(() => {
    if (!API || !data) return;
    const controller = new AbortController();
    let timer = 0;
    const poll = async () => {
      try {
        if (!document.hidden) {
          const suffix = query().toString();
          const response = await fetch(
            `${REVIEW ? `${API}/review/repository/changes` : API}?${suffix}`,
            {
              method: REVIEW ? "GET" : "HEAD",
              headers: data.etag ? { "If-None-Match": data.etag } : {},
              cache: "no-store",
              signal: controller.signal,
            },
          );
          if (response.status !== 304) {
            setStale(true);
            if (selectedRef.current) setOpenFileStale(true);
            else await load(controller.signal);
            setStale(false);
          }
        }
      } catch (failure) {
        if ((failure as Error).name !== "AbortError") setStale(false);
      } finally {
        if (!controller.signal.aborted) timer = window.setTimeout(poll, 2000);
      }
    };
    timer = window.setTimeout(poll, 2000);
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [API, REVIEW, data?.etag, load, query]);
  const files = useMemo(() => mergeLinkedFiles(data?.files || [], manualLinked, linkedPaths), [data?.files, linkedPaths, manualLinked]);
  const visible = useMemo(
    () => visibleChanges(files, filters, locale),
    [files, filters, locale],
  );
  useEffect(() => {
    if (selected && visible.some((item) => item.path === selected.path)) return;
    setSelected(visible[0] || null);
    setTab("source");
  }, [visible, selected]);
  useEffect(() => onSelectionChange?.(selected), [onSelectionChange, selected]);
  useEffect(() => {
    const params = rangeQuery(applied);
    if (selected) params.set("path", selected.path);
    if (tab !== "source") params.set("tab", tab);
    if (filters.query) params.set("q", filters.query);
    if (filters.status) params.set("status", filters.status);
    if (filters.scope) params.set("scope", filters.scope);
    history.replaceState(
      null,
      "",
      `${location.pathname}${params.size ? `?${params}` : ""}`,
    );
  }, [applied, filters, selected, tab]);
  useEffect(() => {
    if (!filesOpen) return;
    const key = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setFilesOpen(false);
        document
          .querySelector<HTMLButtonElement>("[data-mobile-files]")
          ?.focus();
      }
    };
    document.addEventListener("keydown", key);
    return () => document.removeEventListener("keydown", key);
  }, [filesOpen]);
  useEffect(() => {
    const close = () => {
      if (!rangeDetails.current?.open) return;
      rangeDetails.current.open = false;
      rangeSummary.current?.focus();
    };
    const click = (event: MouseEvent) => {
      if (
        rangeDetails.current?.open &&
        !rangeDetails.current.contains(event.target as Node)
      )
        close();
    };
    const key = (event: KeyboardEvent) => {
      if (event.key === "Escape" && rangeDetails.current?.open) close();
    };
    document.addEventListener("click", click);
    document.addEventListener("keydown", key);
    return () => {
      document.removeEventListener("click", click);
      document.removeEventListener("keydown", key);
    };
  }, []);
  if (!API)
    return (
      <main id="changes-main" className="changes-workspace">
        <div className="changes-error" data-ui-state="capability-unavailable">
          {text("changes.viewerUnavailable")}
        </div>
      </main>
    );
  const summary = data?.report.summary.lines;
  const comparison = data?.report.comparison;
  return (
    <main id="changes-main" className="changes-workspace">
      <section className="changes-overview" aria-labelledby="changes-title">
        <h1 id="changes-title">{text("changes.title")}</h1>
        <details
          ref={rangeDetails}
          className="changes-range-details"
          data-range-details
          onToggle={(event) => {
            if (event.currentTarget.open)
              rangeSummary.current =
                event.currentTarget.querySelector("summary");
          }}
        >
          <summary data-range-summary>
            {comparison
              ? `${comparison.base.displayRef} → ${comparison.target.displayRef}`
              : text("changes.rangeSummary")}
          </summary>
          <div className="changes-range">
            <p data-range-meta>
              {comparison
                ? text("features.changes.index.149", [
                    comparison.base.resolved.slice(0, 7),
                    comparison.target.resolved?.slice(0, 7) ||
                      comparison.target.displayRef,
                    data?.report.repository.branch ||
                      text("features.changes.index.150"),
                    text(
                      data?.report.repository.dirty
                        ? "features.changes.index.151"
                        : "features.changes.index.152",
                    ),
                  ])
                : text("changes.preparingGit")}
            </p>
            <label>
              {text("changes.base")}
              <input
                value={range.base}
                onChange={(event) =>
                  setRange({ ...range, base: event.target.value })
                }
                data-base
                spellCheck={false}
              />
            </label>
            <label>
              {text("changes.branchBase")}
              <input
                value={range.branchBase}
                onChange={(event) =>
                  setRange({ ...range, branchBase: event.target.value })
                }
                data-branch-base
                placeholder="main"
                spellCheck={false}
              />
            </label>
            <label>
              {text("changes.target")}
              <select
                value={range.target}
                onChange={(event) =>
                  setRange({ ...range, target: event.target.value })
                }
                data-target
              >
                <option value="working-tree">
                  {text("changes.workingTree")}
                </option>
                <option value="index">index</option>
                <option value="HEAD">HEAD</option>
                <option value="revision">{text("changes.gitRevision")}</option>
              </select>
            </label>
            {range.target === "revision" && (
              <label data-target-revision-wrap>
                {text("changes.targetRevision")}
                <input
                  value={range.revision}
                  onChange={(event) =>
                    setRange({ ...range, revision: event.target.value })
                  }
                  data-target-revision
                  spellCheck={false}
                  placeholder="abc1234"
                />
              </label>
            )}
            <button
              type="button"
              className="changes-button"
              data-apply-range
              onClick={() => {
                if (range.target === "revision" && !range.revision.trim())
                  return setError(text("features.changes.index.072"));
                setApplied(range);
                if (rangeDetails.current) rangeDetails.current.open = false;
                rangeSummary.current?.focus();
              }}
            >
              {text("changes.apply")}
            </button>
          </div>
        </details>
        <p className="changes-summary" data-summary>
          {summary
            ? text("features.changes.index.131", [
                data?.files.length || 0,
                summary.added,
                summary.deleted,
              ])
            : text("changes.preparing")}
        </p>
        <div className="changes-review-actions">
          <button
            type="button"
            className="changes-button secondary"
            data-mobile-files
            aria-expanded={filesOpen}
            onClick={() => setFilesOpen(true)}
          >
            {text("changes.files")}
          </button>
        </div>
      </section>
      <div className="changes-notice" data-stale hidden={!stale} role="status">
        {text("changes.workspaceChanged")}
      </div>
      <div
        className="changes-notice"
        data-open-file-stale
        hidden={!openFileStale}
        role="alert"
      >
        {text("changes.workspaceChanged")}{" "}
        <button
          type="button"
          className="changes-button secondary"
          data-refresh-open-file
          onClick={() => {
            const controller = new AbortController();
            void load(controller.signal)
              .then(() => setOpenFileStale(false))
              .catch((failure) => setError(failure.message));
          }}
        >
          {text("changes.refresh")}
        </button>
      </div>
      {error && (
        <div className="changes-notice" role="alert">
          {error}
        </div>
      )}
      <div className="changes-split">
        <aside
          className={`changes-list-panel${filesOpen ? " is-open" : ""}`}
          data-files-panel
          aria-label={text("changes.files")}
          role={filesOpen ? "dialog" : "navigation"}
          aria-modal={filesOpen || undefined}
        >
          <div className="changes-list-heading">
            <h2>{text("changes.files")}</h2>
            <span data-result-count>
              {text("features.changes.index.013", [
                visible.length,
                data?.files.length || 0,
              ])}
            </span>
            <button
              type="button"
              data-files-close
              aria-label={text("changes.closeFiles")}
              onClick={() => {
                setFilesOpen(false);
                document
                  .querySelector<HTMLButtonElement>("[data-mobile-files]")
                  ?.focus();
              }}
            >
              {text("changes.close")}
            </button>
          </div>
          <div className="changes-list-filters">
            <label className="changes-search">
              <span>{text("changes.search")}</span>
              <input
                type="search"
                data-search
                placeholder={text("changes.pathOrTitle")}
                value={filters.query}
                onChange={(event) =>
                  setFilters({ ...filters, query: event.target.value })
                }
              />
            </label>
            <label>
              <span>{text("changes.status")}</span>
              <select
                data-status
                value={filters.status}
                onChange={(event) =>
                  setFilters({ ...filters, status: event.target.value })
                }
              >
                <option value="">{text("changes.all")}</option>
                <option value="added">{text("changes.added")}</option>
                <option value="untracked">{text("changes.untracked")}</option>
                <option value="modified">{text("changes.modified")}</option>
                <option value="deleted">{text("changes.deleted")}</option>
                <option value="renamed">{text("changes.renamed")}</option>
              </select>
            </label>
            <label>
              <span>{text("changes.type")}</span>
              <select
                data-scope
                value={filters.scope}
                onChange={(event) =>
                  setFilters({ ...filters, scope: event.target.value })
                }
              >
                <option value="">{text("changes.allFiles")}</option>
                <option value="documents">{text("changes.documents")}</option>
                <option value="other">{text("changes.otherFiles")}</option>
              </select>
            </label>
          </div>
          <div className="changes-file-list" data-file-list>
            {visible.length ? (
              <div className="changes-file-tree">
                <Tree
                  node={buildChangeTree(visible)}
                  selected={selected?.path}
                  locale={locale}
                  onSelect={(change) => {
                    setSelected({ ...change });
                    setFilesOpen(false);
                  }}
                />
              </div>
            ) : (
              <div className="changes-list-empty">
                <strong>{text("changes.noMatches")}</strong>
                <p>{text("changes.resetFilters")}</p>
              </div>
            )}
          </div>
          {REVIEW && (
            <FilePicker
              endpoint={`${API}/review`}
              onSelect={(change) => {
                setManualLinked((current) =>
                  [
                    ...current.filter((item) => item.path !== change.path),
                    change,
                  ],
                );
                setSelected(change);
              }}
            />
          )}
        </aside>
        <section className="changes-detail" data-detail aria-live="polite">
          {selected ? (
            renderDetail(selected, tab, setTab)
          ) : (
            <div className="changes-empty" data-ui-state="empty">
              <h2>{text("changes.none")}</h2>
              <p>{text("changes.adjustFilters")}</p>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
