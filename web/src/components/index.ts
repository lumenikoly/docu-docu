import { createIcon, type IconName } from "../design/icons";

export function setExpanded(control: HTMLElement, expanded: boolean): void {
  control.setAttribute("aria-expanded", String(expanded));
}

export function closeDialogOnEscape(dialog: HTMLDialogElement): () => void {
  const listener = (event: KeyboardEvent) => {
    if (event.key === "Escape" && dialog.open) dialog.close();
  };
  dialog.addEventListener("keydown", listener);
  return () => dialog.removeEventListener("keydown", listener);
}

export function selectTab(tabs: readonly HTMLElement[], next: number): void {
  tabs.forEach((tab, index) => {
    const selected = index === next;
    tab.setAttribute("aria-selected", String(selected));
    tab.tabIndex = selected ? 0 : -1;
  });
  tabs[next]?.focus();
}

export type ButtonOptions = {
  label: string;
  onActivate?: () => void;
  disabled?: boolean;
};

export function createButton(options: ButtonOptions): HTMLButtonElement {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "ui-button";
  button.textContent = options.label;
  button.disabled = options.disabled === true;
  if (options.onActivate) button.addEventListener("click", options.onActivate);
  return button;
}

export function createIconButton(label: string, icon: IconName, onActivate?: () => void): HTMLButtonElement {
  const button = createButton({ label: "", onActivate });
  button.className = "ui-icon-button";
  button.setAttribute("aria-label", label);
  button.append(createIcon(icon));
  return button;
}

export function createBadge(text: string, tone = "neutral"): HTMLSpanElement {
  const badge = document.createElement("span");
  badge.className = "ui-badge";
  badge.dataset.tone = tone;
  badge.textContent = text;
  return badge;
}

export function createTabs(tabs: readonly { id: string; label: string; panel: HTMLElement }[]): HTMLElement {
  const list = document.createElement("div");
  list.className = "ui-tabs";
  list.setAttribute("role", "tablist");
  const controls = tabs.map((tab, index) => {
    const control = createButton({ label: tab.label });
    control.id = `${tab.id}-tab`;
    control.setAttribute("role", "tab");
    control.setAttribute("aria-controls", tab.id);
    control.setAttribute("aria-selected", String(index === 0));
    control.tabIndex = index === 0 ? 0 : -1;
    tab.panel.id = tab.id;
    tab.panel.setAttribute("role", "tabpanel");
    tab.panel.setAttribute("aria-labelledby", control.id);
    tab.panel.hidden = index !== 0;
    control.addEventListener("click", () => activate(index));
    return control;
  });
  function activate(next: number): void {
    selectTab(controls, next);
    tabs.forEach((tab, index) => { tab.panel.hidden = index !== next; });
  }
  list.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight" && event.key !== "Home" && event.key !== "End") return;
    event.preventDefault();
    const current = Math.max(0, controls.indexOf(document.activeElement as HTMLButtonElement));
    const next = event.key === "Home" ? 0 : event.key === "End" ? controls.length - 1 : (current + (event.key === "ArrowRight" ? 1 : -1) + controls.length) % controls.length;
    activate(next);
  });
  list.append(...controls);
  return list;
}

export function wireDisclosure(control: HTMLElement, panel: HTMLElement): () => void {
  const toggle = () => {
    const expanded = control.getAttribute("aria-expanded") !== "true";
    setExpanded(control, expanded);
    panel.hidden = !expanded;
  };
  control.addEventListener("click", toggle);
  return () => control.removeEventListener("click", toggle);
}

export function createDialog(label: string, content: Node): HTMLDialogElement {
  const dialog = document.createElement("dialog");
  dialog.className = "ui-dialog";
  dialog.setAttribute("aria-label", label);
  dialog.append(content);
  closeDialogOnEscape(dialog);
  return dialog;
}

export function installTooltip(control: HTMLElement, text: string): HTMLElement {
  const tooltip = document.createElement("span");
  tooltip.className = "ui-tooltip";
  tooltip.role = "tooltip";
  tooltip.textContent = text;
  tooltip.hidden = true;
  const show = () => { tooltip.hidden = false; };
  const hide = () => { tooltip.hidden = true; };
  control.addEventListener("focus", show);
  control.addEventListener("blur", hide);
  control.addEventListener("pointerenter", show);
  control.addEventListener("pointerleave", hide);
  control.after(tooltip);
  return tooltip;
}

export function createCommandMenu(items: readonly { label: string; action: () => void }[]): HTMLElement {
  const menu = document.createElement("div");
  menu.className = "ui-command-menu";
  menu.setAttribute("role", "menu");
  const controls = items.map((item) => {
    const control = createButton({ label: item.label, onActivate: item.action });
    control.setAttribute("role", "menuitem");
    control.tabIndex = -1;
    return control;
  });
  menu.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    event.preventDefault();
    const current = Math.max(-1, controls.indexOf(document.activeElement as HTMLButtonElement));
    const next = (current + (event.key === "ArrowDown" ? 1 : -1) + controls.length) % controls.length;
    controls[next]?.focus();
  });
  menu.append(...controls);
  return menu;
}

export function createTree(items: readonly { label: string; level: number }[]): HTMLElement {
  const tree = document.createElement("ul");
  tree.className = "ui-tree";
  tree.setAttribute("role", "tree");
  items.forEach((item) => {
    const node = document.createElement("li");
    node.setAttribute("role", "treeitem");
    node.setAttribute("aria-level", String(item.level));
    node.textContent = item.label;
    tree.append(node);
  });
  return tree;
}

export function createDataTable(headers: readonly string[], rows: readonly (readonly string[])[]): HTMLTableElement {
  const table = document.createElement("table");
  table.className = "ui-data-table";
  const head = table.createTHead().insertRow();
  headers.forEach((header) => { const cell = document.createElement("th"); cell.scope = "col"; cell.textContent = header; head.append(cell); });
  const body = table.createTBody();
  rows.forEach((row) => { const line = body.insertRow(); row.forEach((value) => { line.insertCell().textContent = value; }); });
  return table;
}

function stateBlock(className: string, state: string, message: string): HTMLElement {
  const block = document.createElement("div");
  block.className = className;
  block.dataset.uiState = state;
  block.textContent = message;
  return block;
}

export const createEmptyState = (message: string): HTMLElement => stateBlock("ui-empty-state", "empty", message);
export const createDiagnostic = (message: string, severity: "info" | "warning" | "error"): HTMLElement => stateBlock("ui-diagnostic", severity, message);

export function createDiffBlock(source: string, label = "Diff"): HTMLElement {
  const block = document.createElement("pre");
  block.className = "ui-diff-block";
  block.setAttribute("aria-label", label);
  block.textContent = source;
  return block;
}
