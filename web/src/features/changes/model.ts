export type RangeState = {
  base: string;
  branchBase: string;
  target: string;
  revision: string;
};
export type Change = {
  path: string;
  oldPath?: string;
  status: string;
  language?: string;
  binary?: boolean;
  asset?: unknown;
  lines: { added: number; deleted: number };
  documentation?: unknown;
  diagnostics?: Array<{ severity: string; code: string; message: string }>;
  sourceDiff?: string;
  sourceDiffHunks?: unknown[];
  renderedDiffAvailable?: boolean;
  semanticDiffAvailable?: boolean;
  semanticChanges?: unknown[];
  relationChanges?: unknown[];
  mermaidBlocks?: unknown[];
  classification?: string;
  entitiesBefore?: Array<{ type: string }>;
  entitiesAfter?: Array<{ type: string }>;
  [key: string]: unknown;
};
export type Filters = { query: string; status: string; scope: string };
export type ChangeTree = {
  directories: Map<string, ChangeTree>;
  files: Change[];
};

export const rangeFromURL = (url = location.href): RangeState => {
  const params = new URL(url).searchParams;
  const target = params.get("target") || "working-tree";
  return {
    base: params.get("base") || "HEAD",
    branchBase: params.get("branchBase") || "",
    target: ["working-tree", "index", "HEAD"].includes(target)
      ? target
      : "revision",
    revision: ["working-tree", "index", "HEAD"].includes(target) ? "" : target,
  };
};

export const rangeQuery = (range: RangeState) => {
  const params = new URLSearchParams();
  if (range.base.trim() && range.base.trim() !== "HEAD")
    params.set("base", range.base.trim());
  if (range.branchBase.trim())
    params.set("branchBase", range.branchBase.trim());
  const target =
    range.target === "revision" ? range.revision.trim() : range.target;
  if (target && target !== "working-tree") params.set("target", target);
  return params;
};

export const isDocumentation = (change: Change) =>
  Boolean(change.documentation) ||
  change.path === "CHANGELOG.md" ||
  change.oldPath === "CHANGELOG.md";
export const visibleChanges = (
  changes: Change[],
  filters: Filters,
  locale: string,
) => {
  const query = filters.query.trim().toLocaleLowerCase(locale);
  return changes
    .filter(
      (change) =>
        (!query ||
          `${change.path} ${change.oldPath || ""}`
            .toLocaleLowerCase(locale)
            .includes(query)) &&
        (!filters.status || change.status === filters.status) &&
        (filters.scope !== "documents" || isDocumentation(change)) &&
        (filters.scope !== "other" || !isDocumentation(change)),
    )
    .sort((left, right) => left.path.localeCompare(right.path, locale));
};

export const buildChangeTree = (changes: Change[]) => {
  const root: ChangeTree = { directories: new Map(), files: [] };
  for (const change of changes) {
    let node = root;
    const parts = change.path.split("/");
    for (const part of parts.slice(0, -1)) {
      let child = node.directories.get(part);
      if (!child) {
        child = { directories: new Map(), files: [] };
        node.directories.set(part, child);
      }
      node = child;
    }
    node.files.push(change);
  }
  return root;
};

export const mergeLinkedFiles = (changes: Change[], manual: Change[], discussionPaths: string[]) => {
  const result = [...changes]; const known = new Set(changes.map((item) => item.path));
  for (const item of manual) if (!known.has(item.path)) { known.add(item.path); result.push(item); }
  for (const path of discussionPaths) if (path && !known.has(path)) { known.add(path); result.push({ path, status: "linked", lines: { added: 0, deleted: 0 } }); }
  return result;
};

export const detailCacheKey = (range: URLSearchParams, change: Change) => `${range}|${change.path}|${change._revision || ""}`;

export const lineDiff = (before: string, after: string) => {
  const oldLines = before ? before.split("\n") : [];
  const newLines = after ? after.split("\n") : [];
  const rows = Array.from({ length: oldLines.length + 1 }, () =>
    Array(newLines.length + 1).fill(0),
  );
  for (let i = oldLines.length - 1; i >= 0; i--)
    for (let j = newLines.length - 1; j >= 0; j--)
      rows[i][j] =
        oldLines[i] === newLines[j]
          ? rows[i + 1][j + 1] + 1
          : Math.max(rows[i + 1][j], rows[i][j + 1]);
  const result: string[] = [];
  let i = 0;
  let j = 0;
  while (i < oldLines.length || j < newLines.length) {
    if (
      i < oldLines.length &&
      j < newLines.length &&
      oldLines[i] === newLines[j]
    ) {
      result.push(`  ${oldLines[i++]}`);
      j++;
    } else if (
      j < newLines.length &&
      (i === oldLines.length || rows[i][j + 1] >= rows[i + 1][j])
    )
      result.push(`+ ${newLines[j++]}`);
    else result.push(`- ${oldLines[i++]}`);
  }
  return result.join("\n");
};
