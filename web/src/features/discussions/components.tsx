import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { createPortal } from "react-dom";
import { Dialog } from "../../ui";
import { text } from "../../core/locale";

export type DiscussionMessage = {
  id: string;
  author: string;
  createdAt: string;
  deliveryId?: string;
  intent?: string;
  outcome?: string;
  state?: string;
  text: string;
};
export type Discussion = {
  id: string;
  state: string;
  createdAt: string;
  target?: { path?: string };
  anchor?: { selectedText?: string };
  placement?: {
    path?: string;
    status?: string;
    range?: { start: { line: number } };
  };
  messages: DiscussionMessage[];
};
export type DiscussionState = {
  revision: number;
  stateDigest: string;
  session?: { discussions?: Discussion[] };
  deliveries?: { id: string; discussionId: string; state: string }[];
};
type Selection = { text: string; occurrence: number };
export type DiscussionComposerState = {
  operation: "create" | "reply" | "edit";
  discussionId?: string;
  message?: DiscussionMessage;
  selection?: Selection;
  target?: Record<string, unknown>;
  quote?: string;
};
const directRequestURL = (endpoint: string, path: string) =>
  `${endpoint}${path}`;

export function useDiscussionState(
  endpoint: string,
  signal: AbortSignal,
  requestURL: (endpoint: string, path: string) => string = directRequestURL,
) {
  const [state, setState] = useState<DiscussionState | null>(null);
  const stateRef = useRef<DiscussionState | null>(null);
  const etag = useRef("");
  const update = useCallback((next: DiscussionState) => {
    stateRef.current = next;
    setState(next);
  }, []);
  const load = useCallback(async () => {
    const response = await fetch(requestURL(endpoint, "/discussions"), {
      headers: etag.current ? { "If-None-Match": etag.current } : {},
      cache: "no-store",
      signal,
    });
    if (response.status === 304) return stateRef.current;
    const result = await response.json();
    if (!response.ok)
      throw new Error(
        result.diagnostics?.[0]?.message || `HTTP ${response.status}`,
      );
    etag.current = response.headers.get("ETag") || "";
    update(result);
    return result as DiscussionState;
  }, [endpoint, requestURL, signal, update]);
  const mutate = useCallback(
    async (
      path: string,
      action: string,
      method: string,
      body: Record<string, unknown>,
    ) => {
      const guard = () => ({
        expectedRevision: stateRef.current?.revision || 0,
        expectedStateDigest: stateRef.current?.stateDigest || "",
      });
      const send = async (payload: Record<string, unknown>) => {
        const response = await fetch(requestURL(endpoint, path), {
          method,
          headers: {
            "Content-Type": "application/json",
            "X-Toudocu-Action": action,
          },
          body: JSON.stringify(payload),
          signal,
        });
        return { response, result: await response.json() };
      };
      let attempt = await send(body);
      if (
        !attempt.response.ok &&
        attempt.result.diagnostics?.[0]?.code === "AGENT_REVISION_CONFLICT"
      ) {
        await load();
        attempt = await send({ ...body, ...guard() });
      }
      if (!attempt.response.ok)
        throw new Error(
          attempt.result.diagnostics?.[0]?.message ||
            `HTTP ${attempt.response.status}`,
        );
      update(attempt.result);
      return attempt.result as DiscussionState;
    },
    [endpoint, load, requestURL, signal, update],
  );
  return {
    state,
    load,
    mutate,
    guard: () => ({
      expectedRevision: stateRef.current?.revision || 0,
      expectedStateDigest: stateRef.current?.stateDigest || "",
    }),
  };
}

export function DiscussionComposer({
  mode,
  title,
  busy,
  error,
  onClose,
  onSubmit,
}: {
  mode: DiscussionComposerState | null;
  title: string;
  busy: boolean;
  error: string;
  onClose: () => void;
  onSubmit: (textValue: string, intent: string) => void;
}) {
  const [message, setMessage] = useState("");
  const [intent, setIntent] = useState("question");
  const dialog = useRef<HTMLDialogElement>(null);
  const returnFocus = useRef<HTMLElement | null>(null);
  useEffect(() => {
    setMessage(mode?.message?.text || "");
    setIntent(mode?.message?.intent || "question");
  }, [mode]);
  useEffect(() => {
    if (!mode || !dialog.current) return;
    returnFocus.current = document.activeElement as HTMLElement;
    dialog.current.showModal();
    return () => {
      if (returnFocus.current?.isConnected) returnFocus.current.focus();
    };
  }, [mode]);
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (message.trim()) onSubmit(message.trim(), intent);
  };
  if (!mode) return null;
  const target = mode.target as
    | {
        path?: string;
        range?: {
          start: { line: number; column: number };
          end: { line: number; column: number };
        };
      }
    | undefined;
  const targetSummary = target?.path
    ? `${target.path}${target.range ? ` · ${target.range.start.line}:${target.range.start.column}–${target.range.end.line}:${target.range.end.column}` : ""}`
    : title;
  return createPortal(
    <dialog
      ref={dialog}
      className="portal-review-dialog review-composer"
      data-review-composer
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
    >
      <form onSubmit={submit}>
        <header>
          <h2>
            {text(
              mode.operation === "reply"
                ? "core.portal.082"
                : "core.portal.042",
            )}
          </h2>
          <span data-review-target-summary>{targetSummary}</span>
        </header>
        {mode?.selection && (
          <label>
            {text("core.portal.043")}
            <pre data-portal-review-selection>{mode.selection.text}</pre>
          </label>
        )}
        <label>
          {text("core.portal.092")}
          <select
            data-portal-review-intent
            data-review-intent
            value={intent}
            onChange={(event) => setIntent(event.target.value)}
          >
            <option value="question">{text("core.portal.093")}</option>
            <option value="change_request">{text("core.portal.094")}</option>
          </select>
        </label>
        <label>
          {text("core.portal.044")}
          <textarea
            autoFocus
            required
            maxLength={65536}
            rows={5}
            placeholder={text("core.portal.045")}
            data-portal-review-question
            data-review-message
            value={message}
            onChange={(event) => setMessage(event.target.value)}
          />
        </label>
        <p className="portal-review-error" role="alert">
          {error}
        </p>
        <footer>
          <button type="button" onClick={onClose}>
            {text("core.portal.046")}
          </button>
          <button
            type="submit"
            className="portal-review-submit"
            disabled={busy}
          >
            {busy
              ? text("core.portal.048")
              : text(
                  mode.operation === "reply"
                    ? "core.portal.083"
                    : mode.operation === "edit"
                      ? "core.portal.099"
                      : "core.portal.047",
                )}
          </button>
        </footer>
      </form>
    </dialog>,
    document.body,
  );
}

export function DiscussionPanel({
  endpoint,
  pageId,
  path,
  title,
  signal,
  initiallyOpen,
  requestURL = directRequestURL,
  targetKind = "document",
  variant = "portal",
  composeEvent,
  onStateChange,
}: {
  endpoint: string;
  pageId: string;
  path: string;
  title: string;
  signal: AbortSignal;
  initiallyOpen: boolean;
  requestURL?: (endpoint: string, path: string) => string;
  targetKind?: "document" | "file";
  variant?: "portal" | "changes";
  composeEvent?: string;
  onStateChange?: (state: DiscussionState | null) => void;
}) {
  const { state, load, mutate, guard } = useDiscussionState(endpoint, signal, requestURL);
  const [open, setOpen] = useState(initiallyOpen);
  const [composer, setComposer] = useState<DiscussionComposerState | null>(
    null,
  );
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<string | null>(null);
  const [selection, setSelection] = useState<
    (Selection & { left: number; top: number }) | null
  >(null);
  const returnFocus = useRef<HTMLElement | null>(null);
  const toggle = document.querySelector<HTMLElement>(
    "[data-discussions-toggle]",
  );
  const canCreate = Boolean(path);
  const discussions = useMemo(
    () =>
      [...(state?.session?.discussions || [])].sort(
        (a, b) =>
          Number(a.state !== "open") - Number(b.state !== "open") ||
          a.createdAt.localeCompare(b.createdAt),
      ),
    [state],
  );
  const openCount = discussions.filter((item) => item.state === "open").length;
  const editable = (message: DiscussionMessage) =>
    message.author === "human" &&
    (message.state === "draft" ||
      (state?.deliveries || []).some(
        (delivery) =>
          delivery.id === message.deliveryId && delivery.state === "pending",
      ));
  const inFlight = (id: string) =>
    (state?.deliveries || []).some(
      (delivery) =>
        delivery.discussionId === id && delivery.state !== "responded",
    );
  const close = useCallback(() => {
    setOpen(false);
    toggle?.setAttribute("aria-expanded", "false");
    returnFocus.current?.focus();
  }, [toggle]);
  const show = useCallback(() => {
    returnFocus.current = document.activeElement as HTMLElement;
    setOpen(true);
    toggle?.setAttribute("aria-expanded", "true");
    void load().catch((failure) => setError(failure.message));
  }, [load, toggle]);

  useEffect(() => {
    load().catch((failure) => setError(failure.message));
  }, [load]);
  useEffect(() => onStateChange?.(state), [onStateChange, state]);
  useEffect(() => {
    toggle?.setAttribute("aria-expanded", String(open));
    if (open)
      requestAnimationFrame(() =>
        document
          .querySelector<HTMLElement>("[data-portal-review-close]")
          ?.focus(),
      );
  }, [open, toggle]);
  useEffect(() => {
    const count = toggle?.querySelector("[data-open-discussion-count]");
    if (count) count.textContent = String(openCount);
  }, [openCount, toggle]);
  useEffect(() => {
    if (!toggle) return;
    const click = () => (open ? close() : show());
    toggle.addEventListener("click", click, { signal });
  }, [close, open, show, signal, toggle]);
  useEffect(() => {
    if (!open) return;
    const timer = window.setInterval(() => {
      if (!document.hidden && !composer && !pendingDelete)
        load().catch(() => {});
    }, 2000);
    return () => window.clearInterval(timer);
  }, [composer, load, open, pendingDelete]);
  useEffect(() => {
    const area = document
      .querySelector<HTMLElement>("[data-copy-document-context]")
      ?.closest<HTMLElement>(".page-content");
    if (!area) return;
    const select = () => {
      const current = window.getSelection();
      if (
        !current ||
        current.isCollapsed ||
        !current.rangeCount ||
        !current.toString().trim()
      )
        return setSelection(null);
      const range = current.getRangeAt(0);
      if (!area.contains(range.commonAncestorContainer))
        return setSelection(null);
      const textValue = current.toString();
      const before = document.createRange();
      before.selectNodeContents(area);
      before.setEnd(range.startContainer, range.startOffset);
      let occurrence = 1;
      for (
        let offset = 0;
        (offset = before.toString().indexOf(textValue, offset)) >= 0;
        offset += textValue.length
      )
        occurrence++;
      const rect = range.getBoundingClientRect();
      setSelection({
        text: textValue,
        occurrence,
        left: Math.min(innerWidth - 360, Math.max(8, rect.left)),
        top: Math.min(innerHeight - 60, Math.max(8, rect.top - 52)),
      });
    };
    area.addEventListener("pointerup", select, { signal });
    area.addEventListener("keyup", select, { signal });
  }, [signal]);
  useEffect(() => {
    const key = (event: KeyboardEvent) => {
      if (event.key === "Escape" && open && !composer && !pendingDelete)
        close();
    };
    document.addEventListener("keydown", key, { capture: true, signal });
  }, [close, composer, open, pendingDelete, signal]);
  useEffect(() => {
    if (!composeEvent) return;
    const compose = (event: Event) => {
      const detail = (event as CustomEvent).detail || {};
      returnFocus.current = detail.returnElement || document.activeElement;
      setComposer({
        operation: "create",
        target: detail.target,
        quote: detail.quote,
      });
    };
    document.addEventListener(composeEvent, compose, { signal });
  }, [composeEvent, signal]);

  const submit = async (value: string, intent: string) => {
    if (!composer) return;
    setBusy(true);
    setError("");
    try {
      if (composer.operation === "reply")
        await mutate(
          `/discussions/${composer.discussionId}/messages`,
          "agent-message-create",
          "POST",
          { ...guard(), intent, text: value },
        );
      else if (composer.operation === "edit")
        await mutate(
          `/discussions/${composer.discussionId}/messages/${composer.message!.id}`,
          "agent-message-update",
          "PATCH",
          { ...guard(), intent, text: value },
        );
      else
        await mutate("/discussions", "agent-discussion-create", "POST", {
          ...guard(),
          target: composer.target || {
            kind: targetKind,
            path,
            ...(targetKind === "document" ? { documentId: pageId } : {}),
          },
          selection: composer.selection
            ? {
                selectedText: composer.selection.text,
                occurrence: composer.selection.occurrence,
              }
            : undefined,
          intent,
          text: composer.quote ? `${value}\n\n${composer.quote}` : value,
        });
      setComposer(null);
      show();
    } catch (failure) {
      setError(text("core.portal.049", [(failure as Error).message]));
    } finally {
      setBusy(false);
    }
  };
  const action = (operation: Promise<unknown>) => {
    void operation.catch(async (failure) => {
      setError((failure as Error).message);
      await load().catch(() => {});
    });
  };
  const placementLabel = (value: string) =>
    text(
      (
        {
          current: "core.portal.063",
          moved: "core.portal.064",
          stale: "core.portal.065",
          deleted: "core.portal.066",
        } as Record<string, string>
      )[value] || value,
    );
  const outcomeLabel = (value: string) =>
    text(
      (
        {
          answered: "core.portal.067",
          changed: "core.portal.090",
          no_change: "core.portal.068",
          needs_clarification: "core.portal.069",
          failed: "core.portal.091",
        } as Record<string, string>
      )[value] || value,
    );

  return createPortal(
    <>
      {selection && (
        <div
          className="review-selection-menu"
          role="toolbar"
          aria-label={text("core.portal.034")}
          style={{ left: selection.left, top: selection.top }}
          onMouseDown={(event) => event.preventDefault()}
        >
          <button
            onClick={() => {
              void navigator.clipboard.writeText(selection.text);
              setSelection(null);
            }}
          >
            {text("core.portal.035")}
          </button>
          <button
            onClick={() => {
              void navigator.clipboard.writeText(
                text("core.portal.038", [title, path, selection.text]),
              );
              setSelection(null);
            }}
          >
            {text("core.portal.036")}
          </button>
          <button
            onClick={() => {
              setComposer({ operation: "create", selection });
              setSelection(null);
            }}
          >
            {text("core.portal.037")}
          </button>
        </div>
      )}
      <button
        type="button"
        className="portal-review-scrim"
        data-discussions-scrim={variant === "changes" || undefined}
        aria-label={text("core.portal.057")}
        hidden={!open}
        onClick={close}
      />
      <aside
        id="project-discussions-panel"
        data-discussions-panel={variant === "changes" || undefined}
        className={`portal-review-panel${variant === "changes" ? " changes-review-panel" : ""}${open ? " is-open" : ""}`}
        aria-label={text("core.portal.054")}
        hidden={!open}
        role={
          matchMedia("(max-width: 560px)").matches ? "dialog" : "complementary"
        }
        aria-modal={matchMedia("(max-width: 560px)").matches || undefined}
      >
        <header>
          <div>
            <h2>{text("core.portal.054")}</h2>
            <span>{path}</span>
            <p data-review-summary>
              {text("core.portal.055", [
                openCount,
                discussions.length - openCount,
              ])}
            </p>
          </div>
          <button
            type="button"
            data-portal-review-close
            data-discussions-close={variant === "changes" || undefined}
            aria-label={text("core.portal.057")}
            onClick={close}
          >
            {text("changes.close")}
          </button>
        </header>
        <div className="portal-review-actions">
          <button
            type="button"
            data-portal-review-new
            hidden={!canCreate}
            onClick={() =>
              setComposer({
                operation: "create",
                target: { kind: targetKind, path },
              })
            }
          >
            {text("core.portal.056")}
          </button>
          <button
            type="button"
            data-portal-review-copy-prompt
            data-send-feedback={variant === "changes" || undefined}
            onClick={() => variant === "changes"
              ? document.dispatchEvent(new CustomEvent("toudocu:agent-compose", { detail: { text: "$toudocu feedback", send: true } }))
              : void navigator.clipboard.writeText(text("core.portal.087"))}
          >
            {text("core.portal.075")}
          </button>
        </div>
        <div
          className="portal-review-list"
          data-discussion-list={variant === "changes" || undefined}
        >
          {error && (
            <div className="portal-review-empty is-error">
              <p>{text("core.portal.080", [error])}</p>
              <button
                onClick={() => {
                  setError("");
                  void load();
                }}
              >
                {text("core.portal.081")}
              </button>
            </div>
          )}
          {!error && !discussions.length && (
            <div className="portal-review-empty">
              <p>{text("core.portal.058")}</p>
              {canCreate && (
                <button
                  onClick={() =>
                    setComposer({
                      operation: "create",
                      target: { kind: targetKind, path },
                    })
                  }
                >
                  {text("core.portal.056")}
                </button>
              )}
            </div>
          )}
          {discussions.map((discussion) => (
            <article
              key={discussion.id}
              className={`portal-review-thread${variant === "changes" ? " review-thread" : ""} is-${discussion.state}`}
            >
              <header>
                <div>
                  <strong>{text("core.portal.070")}</strong>
                  <span>
                    {text(
                      discussion.state === "open"
                        ? "core.portal.059"
                        : "core.portal.060",
                    )}{" "}
                    ·{" "}
                    {placementLabel(discussion.placement?.status || "current")}
                  </span>
                </div>
                <div className="portal-review-thread-actions">
                  <button
                    data-portal-thread-state
                    onClick={() =>
                      action(
                        mutate(
                          `/discussions/${discussion.id}`,
                          "agent-discussion-update",
                          "PATCH",
                          {
                            ...guard(),
                            state:
                              discussion.state === "open" ? "resolved" : "open",
                          },
                        ),
                      )
                    }
                  >
                    {text(
                      discussion.state === "open"
                        ? "core.portal.071"
                        : "core.portal.073",
                    )}
                  </button>
                  <button
                    className="is-danger"
                    data-delete-discussion={variant === "changes" || undefined}
                    onClick={() => setPendingDelete(discussion.id)}
                  >
                    {text("core.portal.072")}
                  </button>
                </div>
              </header>
              <p className="portal-review-anchor">
                {discussion.placement?.path || discussion.target?.path || path}
                {discussion.placement?.range
                  ? `:${discussion.placement.range.start.line}`
                  : ""}
              </p>
              {discussion.anchor?.selectedText && (
                <blockquote className="portal-review-quote">
                  {discussion.anchor.selectedText}
                </blockquote>
              )}
              <ol>
                {discussion.messages.map((message) => (
                  <li
                    key={message.id}
                    className={`is-${message.author}`}
                    data-portal-message={message.id}
                  >
                    <div>
                      <strong>
                        {message.author === "agent"
                          ? `${text("core.portal.061")} · ${outcomeLabel(message.outcome || "answered")}`
                          : `${text("core.portal.070")} · ${text(message.intent === "change_request" ? "core.portal.094" : "core.portal.093")}`}
                      </strong>
                      <time>
                        {new Date(message.createdAt).toLocaleString(
                          window.ToudocuPage?.ui.locale || "en",
                        )}
                      </time>
                    </div>
                    <p>{message.text}</p>
                    {editable(message) && (
                      <div className="portal-review-thread-actions">
                        <button
                          data-portal-message-edit
                          data-edit-message={
                            variant === "changes" ? message.id : undefined
                          }
                          onClick={() =>
                            setComposer({
                              operation: "edit",
                              discussionId: discussion.id,
                              message,
                            })
                          }
                        >
                          {text("core.portal.095")}
                        </button>
                        <button
                          className="is-danger"
                          data-portal-message-delete
                          onClick={() =>
                            action(
                              mutate(
                                `/discussions/${discussion.id}/messages/${message.id}`,
                                "agent-message-delete",
                                "DELETE",
                                guard(),
                              ),
                            )
                          }
                        >
                          {text("core.portal.072")}
                        </button>
                      </div>
                    )}
                  </li>
                ))}
              </ol>
              <button
                className="portal-review-reply"
                disabled={
                  discussion.state !== "open" ||
                  inFlight(discussion.id) ||
                  discussion.messages.some(
                    (message) => message.state === "draft",
                  )
                }
                onClick={() =>
                  setComposer({
                    operation: "reply",
                    discussionId: discussion.id,
                  })
                }
              >
                {text("core.portal.074")}
              </button>
            </article>
          ))}
        </div>
      </aside>
      <DiscussionComposer
        mode={composer}
        title={title}
        busy={busy}
        error={error}
        onClose={() => setComposer(null)}
        onSubmit={submit}
      />
      <Dialog.Root
        modal={false}
        open={Boolean(pendingDelete)}
        onOpenChange={(value) => {
          if (!value) setPendingDelete(null);
        }}
      >
        <Dialog.Portal>
          <Dialog.Backdrop className="ui-backdrop" />
          <Dialog.Viewport className="ui-dialog-viewport">
            <Dialog.Popup
              className="portal-review-dialog portal-review-confirm"
              data-review-delete-confirm={variant === "changes" || undefined}
            >
              <Dialog.Title>{text("core.portal.077")}</Dialog.Title>
              <Dialog.Description>{text("core.portal.078")}</Dialog.Description>
              <footer>
                <Dialog.Close className="ui-button">
                  {text("core.portal.046")}
                </Dialog.Close>
                <button
                  className="portal-review-delete"
                  onClick={() => {
                    if (pendingDelete)
                      action(
                        mutate(
                          `/discussions/${pendingDelete}`,
                          "agent-discussion-delete",
                          "DELETE",
                          guard(),
                        ).then(() => setPendingDelete(null)),
                      );
                  }}
                >
                  {text("core.portal.072")}
                </button>
              </footer>
            </Dialog.Popup>
          </Dialog.Viewport>
        </Dialog.Portal>
      </Dialog.Root>
    </>,
    document.body,
  );
}
