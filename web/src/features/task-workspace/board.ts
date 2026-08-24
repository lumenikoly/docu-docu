export function updateBoard(visible: Set<string>): void {
  document.querySelectorAll<HTMLElement>("[data-workspace-column]").forEach((column) => {
    const cards = [...column.querySelectorAll<HTMLElement>("[data-task-id]")];
    column.hidden = cards.length > 0 && cards.every((card) => !visible.has(card.dataset.taskId || ""));
  });
}
