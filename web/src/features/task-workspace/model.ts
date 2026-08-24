export type Dependency = { id: string; status: string; href: string };
export type Item = { id: string; title: string; workspaceState: string; type: string; priority: string; moduleID: string; parentID: string; dependsOn: Dependency[] };
export type Data = { schemaVersion: number; items: Item[] };

export function readData(): Data | null {
  const node = document.getElementById("task-workspace-data");
  if (!node?.textContent) return null;
  try {
    const value = JSON.parse(node.textContent) as Data;
    return value.schemaVersion === 1 && Array.isArray(value.items) ? value : null;
  } catch { return null; }
}
