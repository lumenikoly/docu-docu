import type { Item } from "./model";

export type Filters = { q: string; state: string[]; priority: string; type: string; module: string; parent: string; completed: boolean; archive: boolean };
export const emptyFilters = (): Filters => ({ q: "", state: [], priority: "", type: "", module: "", parent: "", completed: false, archive: false });
const validStates = new Set(["in-progress", "ready", "ready-candidate", "waiting", "needs-attention", "draft", "blocked", "done", "cancelled", "archive"]);

export function parseFilters(params = new URLSearchParams(location.search)): Filters {
  const state = (params.get("state") || "").split(",").filter((value) => validStates.has(value));
  return { q: params.get("q") || "", state, priority: params.get("priority") || "", type: params.get("type") || "", module: params.get("module") || "", parent: params.get("parent") || "", completed: params.get("completed") === "1", archive: params.get("archive") === "1" };
}

export function serializeFilters(view: string, filters: Filters): URLSearchParams {
  const params = new URLSearchParams();
  if (view !== "board") params.set("view", view);
  if (filters.q) params.set("q", filters.q);
  if (filters.state.length) params.set("state", filters.state.join(","));
  (["priority", "type", "module", "parent"] as const).forEach((name) => { if (filters[name]) params.set(name, filters[name]); });
  if (filters.completed) params.set("completed", "1");
  if (filters.archive) params.set("archive", "1");
  return params;
}

export function matches(item: Item, filters: Filters): boolean {
  const query = filters.q.trim().toLocaleLowerCase();
  if (query && !`${item.id} ${item.title} ${item.moduleID}`.toLocaleLowerCase().includes(query)) return false;
  if (filters.state.length && !filters.state.includes(item.workspaceState)) return false;
  if (!filters.completed && (item.workspaceState === "done" || item.workspaceState === "cancelled")) return false;
  if (!filters.archive && item.workspaceState === "archive") return false;
  return (!filters.priority || item.priority === filters.priority) && (!filters.type || item.type === filters.type) && (!filters.module || item.moduleID === filters.module) && (!filters.parent || item.parentID === filters.parent);
}
