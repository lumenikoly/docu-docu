import { useEffect, useRef, useState, type FormEvent } from "react";
import { createRoot } from "react-dom/client";
import { Dialog } from "../../ui";
import { text } from "../../core/locale";
import type { IslandMount } from "../../core/react/island-host";

type Stage = { anchor: string; title: string; status: { kind: string; label: string }; itemCount: number };
type RoadmapState = { digest: string; suggestedId: string; stages: Stage[] };
type ErrorEnvelope = { error?: { code?: string; message?: string; details?: RoadmapState }; item?: { id: string; stageAnchor: string } };

function RoadmapIsland({ endpoint, signal }: { endpoint: string; signal: AbortSignal }) {
  const [open, setOpen] = useState(false);
  const [state, setState] = useState<RoadmapState | null>(null);
  const [stage, setStage] = useState("");
  const [id, setID] = useState("");
  const [wording, setWording] = useState("");
  const [status, setStatus] = useState<{ kind: string; message: string }>({ kind: "", message: "" });
  const [busy, setBusy] = useState(false);
  const form = useRef<HTMLFormElement>(null);
  const trigger = document.querySelector<HTMLButtonElement>("[data-roadmap-add]");

  const applyState = (next: RoadmapState, preserve: boolean) => {
    setState(next);
    setStage((current) => preserve && next.stages.some((item) => item.anchor === current) ? current : next.stages.find((item) => item.status.kind !== "done")?.anchor || next.stages[0]?.anchor || "");
    setID((current) => preserve && current ? current : next.suggestedId);
  };
  const load = async (preserve: boolean) => {
    const response = await fetch(`${endpoint}/roadmap`, { cache: "no-store", credentials: "same-origin", signal });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(payload.error?.message || `HTTP ${response.status}`);
    applyState(payload, preserve);
  };

  useEffect(() => {
    if (!trigger) return;
    trigger.addEventListener("click", () => setOpen(true), { signal });
  }, [signal, trigger]);
  useEffect(() => {
    if (!open) return;
    setStatus({ kind: "loading", message: text("features.roadmap.009") });
    setBusy(true);
    void load(false).then(() => setStatus({ kind: "", message: "" })).catch((error) => setStatus({ kind: "error", message: text("features.roadmap.013", [error.message]) })).finally(() => setBusy(false));
  }, [open]);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!state || !form.current?.reportValidity()) return;
    setBusy(true); setStatus({ kind: "loading", message: text("features.roadmap.010") });
    try {
      const response = await fetch(`${endpoint}/roadmap/items`, { method: "POST", cache: "no-store", credentials: "same-origin", headers: { "Content-Type": "application/json", "X-Toudocu-Action": "roadmap-add" }, body: JSON.stringify({ stageAnchor: stage, id, text: wording, expectedDigest: state.digest }), signal });
      const payload = await response.json().catch(() => ({})) as ErrorEnvelope;
      if (!response.ok) {
        if (payload.error?.code === "stale_digest" && payload.error.details) {
          applyState(payload.error.details, true);
          setStatus({ kind: "conflict", message: text("features.roadmap.012") });
          return;
        }
        throw new Error(payload.error?.message || `HTTP ${response.status}`);
      }
      const createdID = payload.item?.id || id.toUpperCase();
      setStatus({ kind: "success", message: text("features.roadmap.011", [createdID]) });
      const anchor = payload.item?.stageAnchor || stage;
      window.setTimeout(() => { window.location.hash = anchor; window.location.reload(); }, 500);
    } catch (error) {
      if ((error as Error).name !== "AbortError") setStatus({ kind: "error", message: text("features.roadmap.014", [(error as Error).message]) });
    } finally { setBusy(false); }
  };

  return <Dialog.Root modal="trap-focus" open={open} onOpenChange={setOpen}><Dialog.Portal><Dialog.Backdrop className="ui-backdrop" /><Dialog.Viewport className="ui-dialog-viewport"><Dialog.Popup className="ui-dialog" data-roadmap-dialog>
    <form ref={form} className="roadmap-dialog-form" onSubmit={submit}><Dialog.Title>{text("features.roadmap.001")}</Dialog.Title><p className="roadmap-dialog-description">{status.kind === "loading" && !state ? status.message : ""}</p>
      <label><span>{text("features.roadmap.002")}</span><select required name="stageAnchor" value={stage} disabled={!state?.stages.length} onChange={(event) => setStage(event.target.value)}>{state?.stages.map((item) => <option key={item.anchor} value={item.anchor} title={text("features.roadmap.015", [item.title, item.itemCount])}>{item.title} · {item.status.label} · {item.itemCount}</option>)}</select></label>
      <label><span>{text("features.roadmap.003")}</span><input required name="text" type="text" placeholder={text("features.roadmap.004")} value={wording} onChange={(event) => setWording(event.target.value)} /></label>
      <label><span>{text("features.roadmap.005")}</span><input required name="id" type="text" autoCapitalize="characters" spellCheck={false} pattern="DLV-[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*" value={id} onChange={(event) => setID(event.target.value)} /></label>
      <small className="roadmap-id-hint">{state && text("features.roadmap.006", [state.suggestedId])}</small><p className="roadmap-dialog-status" data-state={status.kind || undefined} role="status" aria-live="polite">{status.kind === "loading" && !state ? "" : status.message}</p>
      <div className="roadmap-dialog-actions"><Dialog.Close className="roadmap-dialog-cancel" disabled={busy}>{text("features.roadmap.007")}</Dialog.Close><button type="submit" className="roadmap-dialog-submit" disabled={busy || !state?.stages.length}>{text("features.roadmap.008")}</button></div>
    </form>
  </Dialog.Popup></Dialog.Viewport></Dialog.Portal></Dialog.Root>;
}

export const mount: IslandMount = (element, { signal, onCleanup }) => {
  const endpoint = window.ToudocuPage?.runtime === "serve" && window.ToudocuPage.capabilities?.editor ? window.ToudocuPage.endpoints?.editor || "" : "";
  if (!endpoint) return;
  const root = createRoot(element);
  onCleanup(() => root.unmount());
  root.render(<RoadmapIsland endpoint={endpoint} signal={signal} />);
};
