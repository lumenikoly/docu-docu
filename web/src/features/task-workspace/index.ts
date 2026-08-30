import { matches, emptyFilters, parseFilters, serializeFilters, type Filters } from "./filters";
import { readData } from "./model";
import { updateBoard } from "./board";
import { updateList } from "./list";
import { visibleTreeItems, wireTree } from "./tree";
import { text } from "../../core/locale";
import { mountTaskActions } from "../task-actions";

let lifecycle: AbortController | null = null;

function initializeTaskWorkspace(): void {
  lifecycle?.abort();
  lifecycle = new AbortController();
  const { signal } = lifecycle;
  const root = document.querySelector<HTMLElement>("[data-workspace-controls]");
  const data = readData();
  mountTaskActions(signal, (projection) => { const item = data?.items.find((candidate) => candidate.id === projection.task.id); if (item) { item.digest = projection.task.digest; item.workspaceState = projection.task.workspaceState; item.agentActions = projection.actions; } });
  if (!root || !data) return;
  let view = ["board", "list", "tree"].includes(new URLSearchParams(location.search).get("view") || "") ? new URLSearchParams(location.search).get("view") || "board" : "board";
  let filters = parseFilters();
  const inputs = new Map<string, HTMLInputElement | HTMLSelectElement>();
  root.querySelectorAll<HTMLInputElement | HTMLSelectElement>("[data-workspace-query], [data-workspace-filter]").forEach((input) => inputs.set(input.dataset.workspaceQuery === "" ? "q" : input.dataset.workspaceFilter || "q", input));
  const setControls = () => { inputs.get("q")!.value = filters.q; (["state", "priority", "type", "module", "parent"] as const).forEach((name) => { const input = inputs.get(name) as HTMLSelectElement | undefined; if (input) input.value = name === "state" ? filters.state[0] || "" : filters[name]; }); (inputs.get("completed") as HTMLInputElement | undefined)!.checked = filters.completed; (inputs.get("archive") as HTMLInputElement | undefined)!.checked = filters.archive; };
  const syncURL = () => history.replaceState(null, "", `${location.pathname}?${serializeFilters(view, filters)}`);
  const render = (writeURL = true) => {
    const matching = new Set(data.items.filter((item) => matches(item, filters)).map((item) => item.id));
    const treeVisible = visibleTreeItems(data.items, matching);
    document.querySelectorAll<HTMLElement>("[data-task-workspace-item]").forEach((node) => { const id = node.dataset.taskId || ""; node.hidden = !((node.closest("[data-workspace-tree]") ? treeVisible : matching).has(id)); node.classList.toggle("is-context", !matching.has(id)); });
    updateBoard(matching); updateList(matching);
    document.querySelectorAll<HTMLElement>("[data-workspace-view-panel]").forEach((panel) => { panel.hidden = panel.dataset.workspaceViewPanel !== view; });
    document.querySelectorAll<HTMLButtonElement>("[data-workspace-view]").forEach((button) => button.setAttribute("aria-current", String(button.dataset.workspaceView === view)));
    const empty = document.querySelector<HTMLElement>("[data-workspace-empty]"); if (empty) empty.hidden = matching.size > 0;
    if (writeURL) syncURL();
  };
  const reset = () => { filters = emptyFilters(); setControls(); render(); };
  const setup = document.querySelector<HTMLElement>("[data-task-agent-setup]");
  const loadAgentSetup = async () => {
    if (!setup || !data.items.some((item) => item.agentActions.length > 0)) return;
    const response = await fetch("/_toudocu/api/agent-console/", { cache: "no-store", signal, headers: { "Accept-Language": document.documentElement.lang } });
    const result = await response.json();
    if (!response.ok) return;
    const skill = result.setup.skill as { state: string; diagnostic: string; command?: string };
    setup.replaceChildren(document.createTextNode([result.setup.selectedProvider, skill.state, skill.diagnostic, skill.command].filter(Boolean).join(" · ")));
    if (["not-installed", "outdated"].includes(skill.state)) {
      const prepare = document.createElement("button");
      prepare.type = "button";
      prepare.textContent = text(skill.state === "outdated" ? "work.agent.update-skill" : "work.agent.install-skill");
      prepare.addEventListener("click", async () => {
        if (!window.confirm(skill.diagnostic)) return;
        prepare.disabled = true;
        const changed = await fetch("/_toudocu/api/agent-console/skill", { method: "POST", headers: { "Content-Type": "application/json", "X-Toudocu-Action": "agent-skill-change" }, body: JSON.stringify({ operation: skill.state === "outdated" ? "update" : "install", confirmed: true }) });
        if (changed.ok) await loadAgentSetup(); else window.alert((await changed.json()).error?.message || text("work.agent.skill-failed"));
      }, { signal });
      setup.append(" ", prepare);
    }
    setup.hidden = false;
  };
  void loadAgentSetup().catch(() => {});
  root.addEventListener("input", () => { const q = inputs.get("q") as HTMLInputElement; filters.q = q.value; render(); }, { signal });
  root.addEventListener("change", () => { filters.state = [(inputs.get("state") as HTMLSelectElement).value].filter(Boolean); filters.priority = (inputs.get("priority") as HTMLSelectElement).value; filters.type = (inputs.get("type") as HTMLSelectElement).value; filters.module = (inputs.get("module") as HTMLSelectElement).value; filters.parent = (inputs.get("parent") as HTMLSelectElement).value; filters.completed = (inputs.get("completed") as HTMLInputElement).checked; filters.archive = (inputs.get("archive") as HTMLInputElement).checked; render(); }, { signal });
  document.querySelectorAll<HTMLButtonElement>("[data-workspace-quick-filter]").forEach((button) => button.addEventListener("click", () => { filters.state = (button.dataset.workspaceQuickFilter || "").split(","); setControls(); render(); }, { signal }));
  document.querySelectorAll<HTMLButtonElement>("[data-workspace-view]").forEach((button) => button.addEventListener("click", () => { view = button.dataset.workspaceView || "board"; render(); }, { signal }));
  document.querySelectorAll<HTMLButtonElement>("[data-workspace-reset]").forEach((button) => button.addEventListener("click", reset, { signal }));
  addEventListener("popstate", () => { filters = parseFilters(); view = new URLSearchParams(location.search).get("view") || "board"; setControls(); render(false); }, { signal });
  setControls(); wireTree(); render(false);
}

initializeTaskWorkspace();
document.addEventListener("toudocu:pagebeforechange", () => lifecycle?.abort());
document.addEventListener("toudocu:pagechange", initializeTaskWorkspace);
