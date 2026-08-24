export function updateList(visible: Set<string>): void {
  document.querySelectorAll<HTMLElement>(".task-workspace-terminal-section").forEach((section) => {
    section.hidden = [...section.querySelectorAll<HTMLElement>("[data-task-id]")].every((node) => !visible.has(node.dataset.taskId || ""));
  });
}
