import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { text } from "../../core/locale";

export type TerminalModeProps = {
  active: boolean;
  busy: boolean;
  frames: { id: number; data: string }[];
  onStart: () => void;
  onStop: () => void;
  onSend: (value: Record<string, unknown>) => boolean;
};

export function TerminalMode({ active, busy, frames, onStart, onStop, onSend }: TerminalModeProps) {
  const host = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const processed = useRef(0);
  useEffect(() => {
    if (!host.current) return;
    const next = new Terminal({ cursorBlink: true, convertEol: false, fontSize: 13, allowTransparency: true, theme: { background: "transparent" } });
    const fit = new FitAddon();
    next.loadAddon(fit); next.open(host.current); fit.fit(); terminal.current = next;
    const input = next.onData((value) => onSend({ action: "terminal-input", text: value }));
    const resize = new ResizeObserver(() => { fit.fit(); onSend({ action: "terminal-resize", columns: next.cols, rows: next.rows }); });
    resize.observe(host.current);
    return () => { resize.disconnect(); input.dispose(); next.dispose(); terminal.current = null; };
  }, [onSend]);
  useEffect(() => {
    if (!terminal.current) return;
    for (const frame of frames) {
      if (frame.id <= processed.current) continue;
      const binary = atob(frame.data), bytes = new Uint8Array(binary.length);
      for (let index = 0; index < binary.length; index++) bytes[index] = binary.charCodeAt(index);
      terminal.current.write(bytes); processed.current = frame.id;
    }
  }, [frames]);
  return <section className="agent-terminal-mode" aria-label={text("core.agent.064")}>
    <header><div><strong>{text("core.agent.064")}</strong><span>{text("core.agent.065")}</span></div><div>{active && <button disabled={busy} onClick={() => onSend({ action: "terminal-interrupt" })}>{text("core.agent.032")}</button>}{active ? <button className="is-danger" disabled={busy} onClick={onStop}>{text("core.agent.033")}</button> : <button className="is-primary" disabled={busy} onClick={onStart}>{text("core.agent.066")}</button>}</div></header>
    <div ref={host} className="agent-terminal-host" />
  </section>;
}
