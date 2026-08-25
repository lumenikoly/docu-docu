import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import ts from "typescript";

const source = await readFile(new URL("../src/features/task-workspace/filters.ts", import.meta.url), "utf8");
const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 } }).outputText;
const filters = await import(`data:text/javascript;base64,${Buffer.from(code).toString("base64")}`);
const item = { id: "TASK-A-001", title: "Task Workspace", workspaceState: "ready", type: "feature", priority: "high", moduleID: "MOD-SITE", parentID: "", dependsOn: [] };

test("workspace filters parse safe query state and combine predicates", () => {
  const state = filters.parseFilters(new URLSearchParams("q=workspace&state=ready,invalid&priority=high&type=feature&module=MOD-SITE"));
  assert.deepEqual(state.state, ["ready"]);
  assert.equal(filters.matches(item, state), true);
  assert.equal(filters.matches(item, { ...state, type: "bug" }), false);
  assert.equal(filters.serializeFilters("list", state).get("view"), "list");
});
