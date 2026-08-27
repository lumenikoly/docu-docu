import { useCallback, useEffect, useMemo, useRef, useState, type ComponentType, type FormEvent } from "react";
import { createRoot } from "react-dom/client";
import type { IslandMount } from "../../core/react/island-host";
import { text } from "../../core/locale";
import { Dialog, Icon, IconButton, Tabs } from "../../ui";

type ConnectionState = "connecting" | "fresh" | "stale";
type AccessPreset = "default" | "full-access";
type PendingMessage = { id?: string; text: string; state?: "queued" | "not-sent"; notSent?: boolean; reason?: string; position?: number };
type Approval = { requestID: string; kind?: string; reason?: string };
type Setup = {
  availableProviders: string[];
  selectedProvider: string;
  preference: { launchPreset: AccessPreset };
  skill: { state: string; diagnostic: string; command?: string };
};
type SessionState = {
  active: boolean;
  status?: "idle" | "running" | "stopping" | "failed";
  activeTurn?: string;
  pending?: PendingMessage[];
  approvals?: Approval[];
  failure?: string;
  verification?: { status: string; task: { id: string }; commands: { command: string; status: string; stdout: string; stderr: string }[] };
  terminal?: { available: boolean; active: boolean; failure?: string };
  settings?: {
    taskID?: string;
    provider?: string;
    preset?: AccessPreset;
    effectiveAccess?: { known?: boolean; unrestricted?: boolean };
    capabilities?: { steering?: boolean; interrupt?: boolean; approvals?: boolean; readOnlyTurns?: boolean };
  };
};
type AgentEvent = {
  type: string;
  turnID?: string;
  itemID?: string;
  text?: string;
  command?: string;
  cwd?: string;
  status?: string;
  exitCode?: number;
  durationMillis?: number;
  approvalState?: string;
  truncated?: boolean;
  approval?: Approval;
};
type WireMessage = { sequence: number; kind: "event" | "state" | "replay_gap" | "terminal"; event?: AgentEvent; state?: SessionState; replayGap?: { after: number; before: number }; terminal?: { type: string; data?: string } };
type ConversationMessage = { id: string; text: string };
type Command = {
  id: string;
  command: string;
  cwd?: string;
  status: string;
  output: string;
  exitCode?: number;
  durationMillis?: number;
  approvalState?: string;
  truncated?: boolean;
};

const OPEN_KEY = "toudocu-agent-console-open";
const TAB_KEY = "toudocu-agent-console-tab";
const emptyState: SessionState = { active: false };

export function stripControlSequences(value: string): string {
  return value
    .replace(/\u001b\][^\u0007]*(?:\u0007|\u001b\\)/g, "")
    .replace(/\u001b\[[0-?]*[ -/]*[@-~]/g, "")
    .replace(/\u001b[@-_]/g, "")
    .replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/g, "");
}

function storedOpen(): boolean {
  try { return sessionStorage.getItem(OPEN_KEY) === "1"; } catch { return false; }
}

function storedTab(): "agent" | "output" {
  try { return sessionStorage.getItem(TAB_KEY) === "output" ? "output" : "agent"; } catch { return "agent"; }
}

function useNarrow(): boolean {
  const [narrow, setNarrow] = useState(() => matchMedia("(max-width: 760px)").matches);
  useEffect(() => {
    const query = matchMedia("(max-width: 760px)");
    const update = () => setNarrow(query.matches);
    query.addEventListener("change", update);
    return () => query.removeEventListener("change", update);
  }, []);
  return narrow;
}

export class ActionError extends Error {
  constructor(message: string, readonly status: number, readonly details?: Record<string, unknown>) { super(message); }
}

export function stopConflictDetails(failure: unknown): { queued: number; notSent: number } | null {
  if (!(failure instanceof ActionError) || failure.status !== 409) return null;
  const { queued, notSent } = failure.details || {};
  return typeof queued === "number" && typeof notSent === "number" ? { queued, notSent } : null;
}

function eventItemID(event: AgentEvent): string {
  return event.itemID || event.turnID || "current";
}

function AgentConsole({ endpoint, signal }: { endpoint: string; signal: AbortSignal }) {
  const [open, setOpen] = useState(storedOpen);
  const [visible, setVisible] = useState(open);
  const [terminalOpen, setTerminalOpen] = useState(false);
  const [tab, setTab] = useState<"agent" | "output">(storedTab);
  const [connection, setConnection] = useState<ConnectionState>("connecting");
  const [session, setSession] = useState<SessionState>(emptyState);
  const [setup, setSetup] = useState<Setup | null>(null);
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [commands, setCommands] = useState<Command[]>([]);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [selectedCommand, setSelectedCommand] = useState("");
  const [followLatest, setFollowLatest] = useState(true);
  const [draft, setDraft] = useState("");
  const [draftPolicy, setDraftPolicy] = useState<"normal" | "filesystem-read-only">("normal");
  const [provider, setProvider] = useState("");
  const [preset, setPreset] = useState<AccessPreset>("default");
  const [error, setError] = useState("");
  const [structuredUnavailable, setStructuredUnavailable] = useState(false);
  const [busy, setBusy] = useState(false);
  const [confirmation, setConfirmation] = useState<"full-access" | "stop" | null>(null);
  const [stopConflict, setStopConflict] = useState({ queued: 0, notSent: 0 });
  const [gap, setGap] = useState("");
  const [announcement, setAnnouncement] = useState("");
  const [terminalFrames, setTerminalFrames] = useState<{ id: number; data: string }[]>([]);
  const [terminalGeneration, setTerminalGeneration] = useState(0);
  const terminalFrameID = useRef(0);
  const [ProjectTerminal, setProjectTerminal] = useState<ComponentType<import("./terminal").ProjectTerminalProps> | null>(null);
  const socket = useRef<WebSocket | null>(null);
  const panel = useRef<HTMLElement | null>(null);
  const sequence = useRef(0);
  const toggle = document.querySelector<HTMLButtonElement>("[data-agent-console-toggle]");
  const terminalToggle = document.querySelector<HTMLButtonElement>("[data-agent-terminal-toggle]");
  const narrow = useNarrow();

  const applyState = useCallback((next: SessionState) => {
    setSession(next);
    if (next.approvals) setApprovals(next.approvals);
    setConnection("fresh");
  }, []);

  const applySetup = useCallback((next: Setup) => {
    setSetup(next);
    setProvider(next.selectedProvider);
    setPreset(next.preference.launchPreset);
  }, []);

  const applyEvent = useCallback((event: AgentEvent) => {
    const id = eventItemID(event);
    if (event.type === "message_delta" && event.text) {
      const safe = stripControlSequences(event.text);
      setMessages((current) => {
        const index = current.findIndex((item) => item.id === id);
        if (index < 0) return [...current, { id, text: safe }];
        return current.map((item, itemIndex) => itemIndex === index ? { ...item, text: item.text + safe } : item);
      });
    }
    if (event.type === "command_started") {
      setCommands((current) => [...current.filter((item) => item.id !== id), {
        id, command: event.command || "", cwd: event.cwd, status: event.status || "running", output: "",
        approvalState: event.approvalState, truncated: event.truncated,
      }]);
    }
    if (event.type === "command_output") {
      setCommands((current) => current.map((item) => item.id === id ? { ...item, output: item.output + stripControlSequences(event.text || ""), truncated: event.truncated ?? item.truncated } : item));
    }
    if (event.type === "command_finished") {
      setCommands((current) => current.map((item) => item.id === id ? {
        ...item, status: event.status || "completed", exitCode: event.exitCode,
        durationMillis: event.durationMillis, approvalState: event.approvalState ?? item.approvalState,
        truncated: event.truncated ?? item.truncated,
      } : item));
    }
    if (event.type === "approval" && event.approval) {
      setApprovals((current) => [...current.filter((item) => item.requestID !== event.approval!.requestID), event.approval!]);
      setAnnouncement(text("core.agent.029"));
    }
    if (event.type === "error" && event.text) setError(stripControlSequences(event.text));
  }, []);

  useEffect(() => {
    let retry = 0;
    const connect = async () => {
      setConnection((current) => current === "fresh" ? "stale" : "connecting");
      try {
        const response = await fetch(endpoint + "/", { cache: "no-store", signal, headers: { "Accept-Language": document.documentElement.lang } });
        const result = await response.json();
        if (!response.ok) throw new Error(result.diagnostics?.[0]?.message || `HTTP ${response.status}`);
        applySetup(result.setup as Setup);
        applyState(result.state);
      } catch (failure) {
        if (!signal.aborted) setError(failure instanceof Error ? failure.message : text("core.agent.044"));
      }
      if (signal.aborted) return;
      const url = new URL(endpoint + `/ws?since=${sequence.current}`, location.href);
      url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
      const ws = new WebSocket(url);
      socket.current = ws;
      ws.onmessage = (message) => {
        const payload = JSON.parse(String(message.data)) as WireMessage;
        sequence.current = Math.max(sequence.current, payload.sequence || 0);
        if (payload.kind === "state" && payload.state) applyState(payload.state);
        if (payload.kind === "event" && payload.event) applyEvent(payload.event);
        if (payload.kind === "replay_gap" && payload.replayGap) setGap(text("core.agent.030"));
        if (payload.kind === "terminal" && payload.terminal?.type === "output") {
          const frame = { id: ++terminalFrameID.current, data: payload.terminal.data || "" };
          setTerminalFrames((current) => [...current, frame].slice(-512));
        }
      };
      ws.onclose = () => {
        if (socket.current === ws) socket.current = null;
        if (!signal.aborted) {
          setConnection("stale");
          retry = window.setTimeout(connect, 900);
        }
      };
      ws.onerror = () => ws.close();
    };
    void connect();
    return () => { window.clearTimeout(retry); socket.current?.close(); };
  }, [applyEvent, applySetup, applyState, endpoint, signal]);

  useEffect(() => {
    if (!terminalOpen || !session.terminal?.available || ProjectTerminal) return;
    void import("./terminal").then((module) => setProjectTerminal(() => module.ProjectTerminal));
  }, [ProjectTerminal, session.terminal?.available, terminalOpen]);

  useEffect(() => {
    const summary = toggle?.querySelector<HTMLElement>("[data-agent-console-summary]");
    const status = session.active ? text(`core.agent.status.${session.status || "idle"}`) : text("core.agent.status.off");
    toggle?.setAttribute("aria-expanded", String(open && !terminalOpen));
    terminalToggle?.setAttribute("aria-expanded", String(open && terminalOpen));
    toggle?.setAttribute("aria-label", `${text("core.agent.001")} · ${status} · ${approvals.length}`);
    if (summary) {
      summary.textContent = `· ${status} · ${approvals.length}`;
      summary.dataset.count = String(approvals.length);
    }
    try { sessionStorage.setItem(OPEN_KEY, open ? "1" : "0"); } catch { /* storage can be disabled */ }
  }, [approvals.length, open, session.active, session.status, terminalOpen, terminalToggle, toggle]);

  useEffect(() => {
    try { sessionStorage.setItem(TAB_KEY, tab); } catch { /* storage can be disabled */ }
  }, [tab]);

  useEffect(() => {
    document.body.classList.toggle("agent-console-open", open && !narrow);
    return () => document.body.classList.remove("agent-console-open");
  }, [narrow, open]);

  useEffect(() => {
    if (open) { setVisible(true); return; }
    if (!visible) return;
    const timeout = window.setTimeout(() => setVisible(false), 180);
    return () => window.clearTimeout(timeout);
  }, [open, visible]);

  const close = useCallback(() => {
    setOpen(false);
    requestAnimationFrame(() => (terminalOpen ? terminalToggle : toggle)?.focus());
  }, [terminalOpen, terminalToggle, toggle]);

  useEffect(() => {
    if (!toggle) return;
    const click = () => {
      if (terminalOpen) { setTerminalOpen(false); setOpen(true); }
      else if (open) close();
      else setOpen(true);
    };
    toggle.addEventListener("click", click);
    return () => toggle.removeEventListener("click", click);
  }, [close, open, terminalOpen, toggle]);

  useEffect(() => {
    if (!terminalToggle) return;
    const click = () => {
      if (open && terminalOpen) close();
      else { setTerminalOpen(true); setOpen(true); }
    };
    terminalToggle.addEventListener("click", click);
    return () => terminalToggle.removeEventListener("click", click);
  }, [close, open, terminalOpen, terminalToggle]);

  useEffect(() => {
    const show = () => { setTerminalOpen(false); setOpen(true); };
    document.addEventListener("toudocu:agent-open", show);
    return () => document.removeEventListener("toudocu:agent-open", show);
  }, []);

  useEffect(() => {
    const show = () => { setTerminalOpen(true); setOpen(true); };
    document.addEventListener("toudocu:terminal-open", show);
    return () => document.removeEventListener("toudocu:terminal-open", show);
  }, []);

  useEffect(() => {
    if (!open) return;
    const escape = (event: KeyboardEvent) => { if (event.key === "Escape" && !confirmation) close(); };
    document.addEventListener("keydown", escape);
    return () => document.removeEventListener("keydown", escape);
  }, [close, confirmation, open]);

  useEffect(() => {
    const background = () => [...document.querySelectorAll<HTMLElement>(".skip-link, .site-header, [data-td-island='discussions'], .site-layout")];
    const apply = () => { for (const element of background()) element.inert = open && narrow; };
    apply();
    document.addEventListener("toudocu:pagechange", apply);
    if (open && narrow && visible) requestAnimationFrame(() => panel.current?.querySelector<HTMLButtonElement>("button:not(:disabled)")?.focus());
    return () => { document.removeEventListener("toudocu:pagechange", apply); for (const element of background()) element.inert = false; };
  }, [narrow, open, visible]);

  useEffect(() => {
    if (!open || !narrow) return;
    const trap = (event: KeyboardEvent) => {
      if (event.key !== "Tab" || confirmation || !panel.current) return;
      const focusable = [...panel.current.querySelectorAll<HTMLElement>("button:not(:disabled), select:not(:disabled), textarea:not(:disabled), [href], [tabindex]:not([tabindex='-1'])")].filter((element) => !element.hidden);
      if (focusable.length === 0) return;
      const first = focusable[0], last = focusable.at(-1)!;
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", trap);
    return () => document.removeEventListener("keydown", trap);
  }, [confirmation, narrow, open]);

  useEffect(() => {
    if (!followLatest || commands.length === 0) return;
    setSelectedCommand(([...commands].reverse().find((item) => item.status === "running") || commands.at(-1))!.id);
  }, [commands, followLatest]);

  const sendSocket = useCallback((value: Record<string, unknown>) => {
    if (connection !== "fresh" || socket.current?.readyState !== WebSocket.OPEN) return false;
    socket.current.send(JSON.stringify(value));
    return true;
  }, [connection]);

  useEffect(() => {
    const compose = (event: Event) => {
      const detail = (event as CustomEvent<{ text?: string; send?: boolean; policy?: "normal" | "filesystem-read-only" }>).detail;
      if (!detail?.text) return;
      setOpen(true);
      if (detail.send && session.active && sendSocket({ action: "message", text: detail.text, policy: detail.policy || "normal" })) return;
      else { setDraft(detail.text); setDraftPolicy(detail.policy || "normal"); }
    };
    document.addEventListener("toudocu:agent-compose", compose);
    return () => document.removeEventListener("toudocu:agent-compose", compose);
  }, [sendSocket, session.active]);

  const post = async (path: string, action: string, body: Record<string, unknown>) => {
    const response = await fetch(endpoint + path, {
      method: "POST", signal,
      headers: { "Content-Type": "application/json", "X-Toudocu-Action": action },
      body: JSON.stringify(body),
    });
    const result = await response.json();
    if (!response.ok) throw new ActionError(result.diagnostics?.[0]?.message || `HTTP ${response.status}`, response.status, result.details);
    return result;
  };

  const act = async (action: () => Promise<unknown>) => {
    setBusy(true); setError("");
    try { await action(); } catch (failure) { setError(failure instanceof Error ? failure.message : text("core.agent.044")); }
    finally { setBusy(false); }
  };

  const start = () => act(async () => {
    try {
      await post("/start", "agent-session-start", {
        ...(window.ToudocuPage?.page.kind === "task" && window.ToudocuPage.page.id ? { taskID: window.ToudocuPage.page.id } : {}),
        provider, preset,
      });
      setStructuredUnavailable(false);
    } catch (failure) {
      setStructuredUnavailable(true);
      throw failure;
    }
  });

  const stop = (discardPending = false) => act(async () => {
    try { await post("/stop", "agent-session-stop", { discardPending }); }
    catch (failure) {
      const conflict = !discardPending && stopConflictDetails(failure);
      if (conflict) {
        setStopConflict(conflict);
        setConfirmation("stop");
        return;
      }
      throw failure;
    }
  });

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const value = draft.trim();
    if (!value) return;
    sendSocket({ action: "message", text: value, policy: draftPolicy });
    setDraft("");
    setDraftPolicy("normal");
  };

  const savePreset = (next: AccessPreset, confirmed: boolean) => act(async () => {
    await post("/preference", "agent-preference-save", { preset: next, confirmed });
    setPreset(next);
    setSetup((current) => current ? { ...current, preference: { launchPreset: next } } : current);
  });

  const cancelPending = (id: string) => act(() => post("/pending/cancel", "agent-pending-cancel", { id }));
  const cleanup = () => act(() => post("/cleanup", "agent-session-cleanup", {}));
  const openTerminal = () => { setError(""); setTerminalOpen(true); setOpen(true); };
  const startTerminal = () => act(() => {
    setTerminalFrames([]);
    setTerminalGeneration((current) => current + 1);
    return post("/terminal/start", "project-terminal-start", {});
  });
  const stopTerminal = () => act(() => post("/terminal/stop", "project-terminal-stop", {}));
  const verify = () => {
    const taskID = session.settings?.taskID;
    if (!taskID || !window.confirm(text("core.agent.054"))) return;
    void act(() => post("/verify", "agent-task-verify", { taskID, confirmed: true }));
  };

  const selected = commands.find((item) => item.id === selectedCommand) || commands.at(-1);
  const providers = setup?.availableProviders || [];
  const mutable = connection === "fresh" && !busy;
  const pending = [...(session.pending || [])].sort((a, b) => (a.position || 0) - (b.position || 0));

  const agentView = <section className="agent-console-view" aria-label={text("core.agent.005")}>
    <div className="agent-console-setup">
      {providers.length === 1 && <p><span>{text("core.agent.010")}</span><strong>{providers[0]}</strong></p>}
      {providers.length > 1 && <label>{text("core.agent.010")}<select value={provider} disabled={session.active || !mutable} onChange={(event) => setProvider(event.target.value)}>{providers.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>}
      <label>{text("core.agent.011")}<select value={preset} disabled={!mutable} onChange={(event) => {
        const value = event.target.value as AccessPreset;
        if (value === "full-access") setConfirmation("full-access"); else void savePreset(value, false);
      }}><option value="default">{text("core.agent.012")}</option><option value="full-access">{text("core.agent.013")}</option></select></label>
      {session.active && <p className="agent-console-access"><span>{text("core.agent.014")}</span><strong>{session.settings?.preset === "full-access" ? text("core.agent.013") : text("core.agent.012")}</strong><span>{text("core.agent.015")}</span><strong>{session.settings?.effectiveAccess?.known ? (session.settings.effectiveAccess.unrestricted ? text("core.agent.013") : text("core.agent.012")) : text("core.agent.016")}</strong></p>}
      {session.active && preset !== session.settings?.preset && <small>{text("core.agent.017")}</small>}
      {setup?.skill.diagnostic && <p className="agent-console-diagnostic">{setup.skill.diagnostic}</p>}
      {structuredUnavailable && <div className="agent-console-fallback"><p>{text("core.agent.068")}</p><button type="button" className="is-primary" onClick={openTerminal}>{text("core.agent.069")}</button></div>}
    </div>
    {gap && <p className="agent-console-gap">{gap}</p>}
    <div className="agent-console-conversation" aria-label={text("core.agent.018")}>
      {messages.length === 0 ? <p className="agent-console-empty">{text("core.agent.019")}</p> : messages.map((message) => <p key={message.id}>{message.text}</p>)}
    </div>
    {approvals.length > 0 && <section className="agent-console-approvals"><h3>{text("core.agent.020")}</h3>{approvals.map((approval) => <div key={approval.requestID}><p><strong>{approval.kind || text("core.agent.021")}</strong>{approval.reason && <span>{approval.reason}</span>}</p><div><button disabled={!mutable} onClick={() => { sendSocket({ action: "approval", requestID: approval.requestID, decision: "decline" }); setApprovals((current) => current.filter((item) => item.requestID !== approval.requestID)); }}>{text("core.agent.022")}</button><button disabled={!mutable} onClick={() => { sendSocket({ action: "approval", requestID: approval.requestID, decision: "accept" }); setApprovals((current) => current.filter((item) => item.requestID !== approval.requestID)); }}>{text("core.agent.023")}</button></div></div>)}</section>}
    {pending.length > 0 && <ol className="agent-console-pending" aria-label={text("core.agent.024")}>{pending.map((item, index) => <li key={item.id || `${index}-${item.text}`}><span>{item.state === "not-sent" || item.notSent ? text("core.agent.026") : text("core.agent.025")}</span><p>{item.text}</p>{item.reason && <small>{item.reason}</small>}{item.id && <button disabled={!mutable} onClick={() => void cancelPending(item.id!)}>{text("core.agent.027")}</button>}</li>)}</ol>}
    <form className="agent-console-composer" onSubmit={submit}>
      <label htmlFor="agent-console-message">{text("core.agent.028")}</label>
      <textarea id="agent-console-message" rows={3} maxLength={65536} value={draft} disabled={!session.active || session.status === "failed" || !mutable} onChange={(event) => { setDraft(event.target.value); setDraftPolicy("normal"); }} />
      <div>{!session.active && <button type="button" disabled={!mutable} onClick={start}>{text("core.agent.031")}</button>}{session.active && session.settings?.taskID && <button type="button" disabled={!mutable || session.status === "running"} onClick={verify}>{text("core.agent.055")}</button>}{session.status === "running" && session.settings?.capabilities?.interrupt && <button type="button" disabled={!mutable} onClick={() => sendSocket({ action: "interrupt" })}>{text("core.agent.032")}</button>}{session.active && <button type="button" className="is-danger" disabled={!mutable} onClick={() => void (session.status === "failed" ? cleanup() : stop())}>{session.status === "failed" ? text("core.agent.034") : text("core.agent.033")}</button>}<button type="submit" className="is-primary" disabled={!draft.trim() || !session.active || session.status === "failed" || session.status === "stopping" || !mutable}>{session.status === "running" ? text("core.agent.036") : text("core.agent.035")}</button></div>
    </form>
  </section>;

  const commandOutput = <section className="agent-command-view" aria-label={text("core.agent.006")}>
    <div className="agent-command-list">{commands.length === 0 ? <p className="agent-console-empty">{text("core.agent.037")}</p> : commands.map((command) => <button key={command.id} className={command.id === selected?.id ? "is-selected" : ""} onClick={() => { setSelectedCommand(command.id); setFollowLatest(false); }}><code>{command.command || text("core.agent.038")}</code><span>{command.status}</span>{command.cwd && <small>{command.cwd}</small>}</button>)}</div>
    {!followLatest && <button className="agent-follow-latest" onClick={() => setFollowLatest(true)}>{text("core.agent.039")}</button>}
    {selected && <div className="agent-command-detail"><dl><div><dt>{text("core.agent.040")}</dt><dd>{selected.status}</dd></div>{selected.exitCode !== undefined && <div><dt>{text("core.agent.041")}</dt><dd>{selected.exitCode}</dd></div>}{selected.durationMillis !== undefined && <div><dt>{text("core.agent.042")}</dt><dd>{selected.durationMillis} ms</dd></div>}{selected.approvalState && <div><dt>{text("core.agent.043")}</dt><dd>{selected.approvalState}</dd></div>}</dl><div className="agent-command-actions"><button onClick={() => void navigator.clipboard.writeText(selected.command)}>{text("core.agent.056")}</button><button onClick={() => { setDraft(text("core.agent.057", [selected.command])); setDraftPolicy("filesystem-read-only"); }}>{text("core.agent.058")}</button><button onClick={() => void navigator.clipboard.writeText(window.getSelection()?.toString() || selected.output)}>{text("core.agent.059")}</button><button onClick={() => { setDraft(text("core.agent.060", [window.getSelection()?.toString() || selected.output])); setDraftPolicy("filesystem-read-only"); }}>{text("core.agent.061")}</button></div>{selected.truncated && <p className="agent-console-gap">{text("core.agent.046")}</p>}<pre>{selected.output}</pre></div>}
    {session.verification && <section className="agent-verification-output" aria-label={text("core.agent.062")}><h3>{text("core.agent.062")}</h3><strong>{session.verification.status}</strong>{session.verification.commands.map((command, index) => <div key={index}><code>{command.command}</code><span>{command.status}</span><pre>{stripControlSequences(command.stdout + command.stderr)}</pre></div>)}{session.verification.status === "failed" && <button disabled={!mutable || !session.active} onClick={() => void act(() => post("/verification/send", "agent-verification-send", {}))}>{text("core.agent.063")}</button>}</section>}
  </section>;

  if (!visible) return <span className="visually-hidden" aria-live="polite">{announcement}</span>;
  return <>
    <div className="agent-console-scrim" aria-hidden="true" hidden={!narrow} />
    <aside ref={panel} id="agent-console-panel" className={`agent-console-panel${open ? " is-open" : ""}`} role={narrow ? "dialog" : undefined} aria-modal={narrow ? true : undefined} aria-label={terminalOpen ? text("core.agent.064") : text("core.agent.001")}>
      <header><div><strong>{terminalOpen ? text("core.agent.064") : text("core.agent.001")}</strong><span data-connection={connection}>{connection === "fresh" ? text("core.agent.007") : connection === "stale" ? text("core.agent.008") : text("core.agent.009")}</span></div>{terminalOpen ? <IconButton className={`agent-terminal-control${session.terminal?.active ? " is-danger" : " is-primary"}`} aria-label={text(session.terminal?.active ? "core.agent.072" : "core.agent.066")} title={text(session.terminal?.active ? "core.agent.072" : "core.agent.066")} disabled={!mutable} onClick={session.terminal?.active ? stopTerminal : startTerminal}><Icon name={session.terminal?.active ? "stop" : "play"} /></IconButton> : <button data-agent-console-close onClick={close}>{text("core.agent.004")}</button>}</header>
      <span className="visually-hidden" aria-live="polite">{announcement}</span>
      {error && <p className="agent-console-error" role="alert">{error}</p>}
      {terminalOpen ? ProjectTerminal && session.terminal?.available && <ProjectTerminal key={terminalGeneration} active={session.terminal.active} frames={terminalFrames} onSend={sendSocket} /> : narrow ? <Tabs.Root className="agent-console-tabs" value={tab} onValueChange={(value) => setTab(value as "agent" | "output")}><Tabs.List><Tabs.Tab value="agent">{text("core.agent.005")} {approvals.length > 0 && <span>{approvals.length}</span>}</Tabs.Tab><Tabs.Tab value="output">{text("core.agent.006")}</Tabs.Tab></Tabs.List><Tabs.Panel value="agent">{agentView}</Tabs.Panel><Tabs.Panel value="output">{commandOutput}</Tabs.Panel></Tabs.Root> : <div className="agent-console-columns">{agentView}{commandOutput}</div>}
      {session.terminal?.failure && <p className="agent-console-error" role="alert">{session.terminal.failure}</p>}
    </aside>
    <Dialog.Root open={confirmation !== null} onOpenChange={(next) => { if (!next) setConfirmation(null); }}><Dialog.Portal><Dialog.Backdrop className="ui-backdrop" /><Dialog.Popup className="agent-console-confirm"><Dialog.Title>{confirmation === "full-access" ? text("core.agent.047") : text("core.agent.049")}</Dialog.Title><Dialog.Description>{confirmation === "full-access" ? text("core.agent.048") : text("core.agent.050", [stopConflict.queued, stopConflict.notSent])}</Dialog.Description><div><Dialog.Close>{text("core.agent.051")}</Dialog.Close><button className="is-danger" onClick={() => { if (confirmation === "full-access") void savePreset("full-access", true); else void stop(true); setConfirmation(null); }}>{confirmation === "full-access" ? text("core.agent.052") : text("core.agent.053")}</button></div></Dialog.Popup></Dialog.Portal></Dialog.Root>
  </>;
}

export const mount: IslandMount = (element, { signal, onCleanup }) => {
  const page = window.ToudocuPage;
  if (!page?.capabilities.agentConsole || !page.endpoints?.agentConsole) return;
  const root = createRoot(element);
  onCleanup(() => root.unmount());
  root.render(<AgentConsole endpoint={page.endpoints.agentConsole} signal={signal} />);
};
