import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { text } from "../../core/locale";

export type ProjectTerminalProps = {
  active: boolean;
  frames: { id: number; data: string }[];
  onSend: (value: Record<string, unknown>) => boolean;
};

export function ProjectTerminal({ active, frames, onSend }: ProjectTerminalProps) {
  const host = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const fitter = useRef<FitAddon | null>(null);
  const processed = useRef(0);
  const isActive = useRef(active);
  useEffect(() => { isActive.current = active; }, [active]);
  useEffect(() => {
    if (!host.current) return;
    const next = new Terminal({ cursorBlink: true, convertEol: false, fontSize: 13, allowTransparency: true, theme: { background: "transparent" } });
    const fit = new FitAddon();
    next.loadAddon(fit); next.open(host.current); fit.fit(); terminal.current = next; fitter.current = fit;
    const input = next.onData((value) => { if (isActive.current) onSend({ action: "terminal-input", text: value }); });
    const resize = new ResizeObserver(() => { fit.fit(); if (isActive.current && next.cols >= 2 && next.rows >= 2) onSend({ action: "terminal-resize", columns: next.cols, rows: next.rows }); });
    resize.observe(host.current);
    return () => { resize.disconnect(); input.dispose(); next.dispose(); terminal.current = null; fitter.current = null; };
  }, [onSend]);
  useEffect(() => {
    if (!active || !terminal.current || !fitter.current) return;
    fitter.current.fit();
    if (terminal.current.cols >= 2 && terminal.current.rows >= 2) onSend({ action: "terminal-resize", columns: terminal.current.cols, rows: terminal.current.rows });
  }, [active, onSend]);
  useEffect(() => {
    if (!terminal.current) return;
    for (const frame of frames) {
      if (frame.id <= processed.current) continue;
      const binary = atob(frame.data), bytes = new Uint8Array(binary.length);
      for (let index = 0; index < binary.length; index++) bytes[index] = binary.charCodeAt(index);
      terminal.current.write(bytes); processed.current = frame.id;
    }
  }, [frames]);
  return <section className="agent-terminal-mode" aria-label={text("core.agent.064")}><div ref={host} className="agent-terminal-host" /></section>;
}
