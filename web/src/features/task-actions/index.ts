import { text } from "../../core/locale";
import { createIcon } from "../../design/icons";

export type Delivery = { type: "agent-console" | "handoff"; available: boolean; unavailableReason?: string; openSession?: boolean };
export type Action = { id: string; label: string; input: "none" | "text"; deliveries: Delivery[] };
export type AgentState = { relation: "none" | "current-task" | "other-task" | "unbound"; status: "off" | "idle" | "running" | "stopping" | "failed"; needsAttention: boolean };
export type Projection = { schemaVersion: 1; task: { id: string; status: string; workspaceState: string; digest: string }; agent: AgentState; actions: Action[] };
type Result = { error?: { code: string; message: string }; openSession?: boolean; handoff?: { text: string }; projection?: Projection };
type Outcome = "done" | "copied" | "failed";

const actionLabel = (action: Action, delivery?: Delivery) => delivery?.openSession ? text("work.agent.open-agent") : text(`work.agent.${action.id}`);
const deliveryLabel = (delivery: Delivery) => text(`work.agent.delivery.${delivery.type}`);
function dialog(title: string, signal: AbortSignal): { node: HTMLDialogElement; body: HTMLElement; status: HTMLElement; close: () => void } {
  const node = document.createElement("dialog");
  node.className = "task-action-dialog";
  node.dataset.taskActionDialog = "";
  node.innerHTML = `<form method="dialog"><header><h2></h2><button class="task-action-dialog-close" type="submit" value="cancel" formnovalidate aria-label="${text("work.agent.close")}"></button></header><div data-task-action-dialog-body></div><p role="alert" data-task-action-status></p></form>`;
  node.querySelector("h2")!.textContent = title;
  node.querySelector(".task-action-dialog-close")!.append(createIcon("close"));
  const restore = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  const close = () => node.open && node.close();
  node.addEventListener("close", () => { node.remove(); restore?.focus(); }, { once: true, signal });
  node.addEventListener("cancel", close, { signal });
  document.body.append(node);
  node.showModal();
  return { node, body: node.querySelector("[data-task-action-dialog-body]")!, status: node.querySelector("[data-task-action-status]")!, close };
}

async function copyHandoff(instruction: string, signal: AbortSignal): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(instruction);
    return true;
  } catch { /* expose a selectable fallback */ }
  const view = dialog(text("work.agent.handoff-title"), signal);
  const label = document.createElement("label");
  label.textContent = text("work.agent.handoff-label");
  const output = document.createElement("textarea");
  output.readOnly = true;
  output.value = instruction;
  const copy = document.createElement("button");
  copy.type = "button";
  copy.textContent = text("work.agent.copy");
  copy.addEventListener("click", async () => {
    output.select();
    try { await navigator.clipboard.writeText(output.value); view.close(); }
    catch { view.status.textContent = text("work.agent.copy-failed"); copy.textContent = text("work.agent.copy-again"); }
  }, { signal });
  label.append(output);
  view.body.append(label, copy);
  output.focus(); output.select();
  return false;
}

export function mountTaskActions(signal: AbortSignal, onProjection?: (projection: Projection) => void): void {
  const endpoint = window.ToudocuPage?.endpoints?.taskActions;
  if (!endpoint) return;
  signal.addEventListener("abort", () => { document.querySelectorAll<HTMLDialogElement>("[data-task-action-dialog]").forEach((node) => node.close()); }, { once: true });

  const generations = new Map<string, number>();
  const inFlight = new Map<string, Promise<Projection>>();
  const begin = (taskID: string) => { const next = (generations.get(taskID) || 0) + 1; generations.set(taskID, next); return next; };
  const commit = (projection: Projection, generation: number) => {
    if (generations.get(projection.task.id) !== generation) return;
    document.querySelectorAll<HTMLElement>(`[data-task-actions][data-task-id="${CSS.escape(projection.task.id)}"]`).forEach((root) => render(root, projection));
    onProjection?.(projection);
  };
  const load = async (taskID: string, generation: number): Promise<void> => {
    let request = inFlight.get(taskID);
    if (!request) {
      request = fetch(`${endpoint}/${encodeURIComponent(taskID)}/actions`, { cache: "no-store", signal }).then(async (response) => {
        const result = await response.json() as Projection & Result;
        if (!response.ok) throw new Error(result.error?.message || text("work.agent.failed"));
        return result;
      });
      inFlight.set(taskID, request);
      void request.finally(() => { if (inFlight.get(taskID) === request) inFlight.delete(taskID); }).catch(() => {});
    }
    const result = await request;
    commit(result, generation);
  };
  const execute = async (projection: Projection, action: Action, delivery: Delivery, input: string, status?: HTMLElement): Promise<Outcome> => {
    if (delivery.openSession) { document.dispatchEvent(new CustomEvent("toudocu:agent-open")); return "done"; }
    const generation = begin(projection.task.id);
    const response = await fetch(`${endpoint}/${encodeURIComponent(projection.task.id)}/actions/${encodeURIComponent(action.id)}`, { method: "POST", headers: { "Content-Type": "application/json", "X-Toudocu-Action": "task-action-execute" }, body: JSON.stringify({ delivery: delivery.type, expectedDigest: projection.task.digest, input: { text: input } }), signal });
    const result = await response.json() as Result;
    if (!response.ok) {
      if (status) { status.textContent = `${result.error?.code || "task_action_failed"}: ${result.error?.message || text("work.agent.failed")}`; status.dataset.state = "error"; }
      if (result.error?.code === "stale_digest") {
        const freshGeneration = begin(projection.task.id);
        await load(projection.task.id, freshGeneration);
      }
      return "failed";
    }
    if (result.projection) {
      commit(result.projection, generation);
    }
    if (result.handoff?.text && await copyHandoff(result.handoff.text, signal)) return "copied";
    return "done";
  };
  const findControl = (taskID: string, actionID: string, delivery: Delivery) => document.querySelector<HTMLButtonElement>(`[data-task-actions][data-task-id="${CSS.escape(taskID)}"] [data-task-agent-action="${CSS.escape(actionID)}"] [data-delivery="${delivery.type}"]`);
  const setPending = (taskID: string, actionID: string, delivery: Delivery, pending: boolean) => {
    const button = findControl(taskID, actionID, delivery);
    if (!button) return;
    button.disabled = pending || button.dataset.available === "false"; button.toggleAttribute("aria-busy", pending); button.dataset.state = pending ? "pending" : "";
    const status = button.closest<HTMLElement>("[data-task-agent-action]")?.querySelector<HTMLElement>("[data-task-action-status]");
    if (status && pending) { status.dataset.state = ""; status.textContent = text(delivery.type === "agent-console" ? "work.agent.sending" : "work.agent.preparing"); }
    else if (status?.dataset.state !== "error") status!.textContent = "";
  };
  const showOutcome = (taskID: string, actionID: string, delivery: Delivery, outcome: Outcome) => {
    const button = findControl(taskID, actionID, delivery);
    if (!button || outcome !== "copied") return;
    button.dataset.state = "copied";
    const status = button.closest<HTMLElement>("[data-task-agent-action]")?.querySelector<HTMLElement>("[data-task-action-status]");
    if (status) status.textContent = text("work.agent.copied");
    window.setTimeout(() => { if (!signal.aborted && button.isConnected) { button.dataset.state = ""; if (status) status.textContent = ""; } }, 1600);
  };
  const enterText = (projection: Projection, action: Action, selected?: Delivery) => {
    const custom = !selected;
    const view = dialog(custom ? text("work.agent.customize-title", [actionLabel(action)]) : text("work.agent.ask-title", [projection.task.id]), signal);
    const label = document.createElement("label"); label.textContent = text(custom ? "work.agent.custom-instruction" : "work.agent.question");
    const input = document.createElement("textarea"); input.name = custom ? "instruction" : "question"; input.required = true; label.append(input);
    const actions = document.createElement("footer"); actions.className = "task-action-deliveries";
    (selected ? [selected] : action.deliveries).forEach((delivery) => {
      const button = document.createElement("button"); button.type = "button"; button.textContent = deliveryLabel(delivery); button.disabled = !delivery.available;
      if (delivery.type === "agent-console") button.className = "is-primary";
      if (delivery.unavailableReason) button.title = text(`work.agent.reason.${delivery.unavailableReason}`);
      button.addEventListener("click", async () => {
        if (!input.reportValidity()) return;
        button.disabled = true; setPending(projection.task.id, action.id, delivery, true);
        let outcome: Outcome | undefined;
        try {
          outcome = await execute(projection, action, delivery, input.value, view.status);
        } catch (error) {
          view.status.textContent = error instanceof Error ? error.message : text("work.agent.failed"); view.status.dataset.state = "error";
        } finally {
          setPending(projection.task.id, action.id, delivery, false); button.disabled = !delivery.available;
        }
        if (outcome) { showOutcome(projection.task.id, action.id, delivery, outcome); if (outcome !== "failed") view.close(); }
      }, { signal });
      actions.append(button);
    });
    view.body.append(label, actions); input.focus();
  };
  const run = async (projection: Projection, action: Action, delivery: Delivery, status?: HTMLElement): Promise<Outcome | void> => {
    if (action.input === "text") enterText(projection, action, delivery);
    else return execute(projection, action, delivery, "", status);
  };
  const render = (root: HTMLElement, projection: Projection) => {
    root.dataset.taskDigest = projection.task.digest;
    root.replaceChildren();
    const item = root.closest<HTMLElement>("[data-task-workspace-item]");
    if (item) {
      item.dataset.state = projection.task.workspaceState;
      const state = item.querySelector<HTMLElement>(".task-workspace-state, td:nth-child(3), :scope > span");
      if (state) state.textContent = text(`work.state.${projection.task.workspaceState}`);
    } else {
      const state = document.querySelector<HTMLElement>(".page-header .status-chip");
      if (state) state.textContent = text(`work.state.${projection.task.workspaceState}`);
    }
    if (projection.agent.relation === "current-task") {
      const visibleState = projection.agent.status === "failed" ? "failed" : projection.agent.needsAttention ? "attention" : projection.agent.status;
      const agent = document.createElement("button"); agent.type = "button"; agent.className = "task-agent-session"; agent.dataset.state = visibleState;
      agent.append(createIcon("terminal"));
      const label = document.createElement("span"); label.textContent = text(`work.agent.session.${visibleState}`); agent.append(label);
      agent.setAttribute("aria-label", `${label.textContent}. ${text("work.agent.open-agent")}`); agent.title = agent.getAttribute("aria-label")!;
      agent.addEventListener("click", () => document.dispatchEvent(new CustomEvent("toudocu:agent-open")), { signal }); root.append(agent);
    }
    projection.actions.forEach((action) => {
      const row = document.createElement("div"); row.className = "task-agent-action"; row.dataset.taskAgentAction = action.id;
      const label = document.createElement("span"); label.textContent = actionLabel(action); row.append(label);
      const controls = document.createElement("span"); controls.className = "task-agent-action-controls";
      (["agent-console", "handoff"] as const).forEach((type) => {
        const delivery = action.deliveries.find((candidate) => candidate.type === type);
        if (!delivery) return;
        const button = document.createElement("button"); button.type = "button"; button.disabled = !delivery.available; button.dataset.delivery = type; button.dataset.available = String(delivery.available);
        if (type === "agent-console") button.append(createIcon("terminal"));
        else {
          const idle = createIcon("clipboard"); idle.classList.add("task-action-icon-idle");
          const success = createIcon("checkCircle"); success.classList.add("task-action-icon-success"); button.append(idle, success);
        }
        const description = `${actionLabel(action, delivery)} — ${deliveryLabel(delivery)}`;
        button.setAttribute("aria-label", description); button.title = delivery.unavailableReason ? `${description}: ${text(`work.agent.reason.${delivery.unavailableReason}`)}` : description;
        button.addEventListener("click", () => {
          const status = row.querySelector<HTMLElement>("[data-task-action-status]")!;
          if (action.input === "text" || delivery.openSession) { void run(projection, action, delivery, status); return; }
          setPending(projection.task.id, action.id, delivery, true);
          void run(projection, action, delivery, status).then((outcome) => {
            setPending(projection.task.id, action.id, delivery, false); if (outcome) showOutcome(projection.task.id, action.id, delivery, outcome);
          }).catch((error) => { setPending(projection.task.id, action.id, delivery, false); status.textContent = error instanceof Error ? error.message : text("work.agent.failed"); });
        }, { signal });
        controls.append(button);
      });
      if (action.input === "none") {
        const customize = document.createElement("button"); customize.type = "button"; customize.dataset.customize = ""; customize.append(createIcon("plus"));
        customize.setAttribute("aria-label", text("work.agent.customize", [actionLabel(action)])); customize.title = customize.getAttribute("aria-label")!;
        customize.addEventListener("click", () => enterText(projection, action), { signal }); controls.append(customize);
      }
      const status = document.createElement("small"); status.dataset.taskActionStatus = ""; status.setAttribute("role", "status"); status.setAttribute("aria-live", "polite");
      row.append(controls, status); root.append(row);
    });
  };
  const refresh = (root: HTMLElement) => { const taskID = root.dataset.taskId; if (!taskID) return; void load(taskID, begin(taskID)).catch(() => {}); };
  const refreshAll = () => new Set([...document.querySelectorAll<HTMLElement>("[data-task-actions]")].map((root) => root.dataset.taskId).filter(Boolean)).forEach((taskID) => refresh(document.querySelector<HTMLElement>(`[data-task-actions][data-task-id="${CSS.escape(taskID!)}"]`)!));
  refreshAll();
  document.addEventListener("toudocu:agentstatechange", refreshAll, { signal });
}
