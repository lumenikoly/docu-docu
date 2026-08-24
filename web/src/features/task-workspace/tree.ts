import type { Item } from "./model";

export function visibleTreeItems(items: Item[], matching: Set<string>): Set<string> {
  const parents = new Map(items.map((item) => [item.id, item.parentID]));
  const visible = new Set(matching);
  matching.forEach((id) => { for (let parent = parents.get(id); parent; parent = parents.get(parent)) visible.add(parent); });
  return visible;
}

export function wireTree(): void {
  document.querySelectorAll<HTMLButtonElement>("[data-task-tree-toggle]").forEach((button) => button.addEventListener("click", () => {
    const list = button.parentElement?.querySelector(":scope > ul") as HTMLElement | null;
    if (!list) return;
    const expanded = button.getAttribute("aria-expanded") !== "true";
    button.setAttribute("aria-expanded", String(expanded));
    button.setAttribute("aria-label", button.dataset[expanded ? "collapseLabel" : "expandLabel"] || "");
    list.hidden = !expanded;
  }));
}
