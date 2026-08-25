import { describe, expect, it } from "vitest";
import {
  buildChangeTree,
  detailCacheKey,
  lineDiff,
  mergeLinkedFiles,
  rangeFromURL,
  rangeQuery,
  visibleChanges,
  type Change,
} from "../src/features/changes/model";

const changes = [
  {
    path: "docs/a.md",
    status: "modified",
    lines: { added: 1, deleted: 0 },
    documentation: {},
  },
  { path: "server.go", status: "untracked", lines: { added: 2, deleted: 0 } },
] as Change[];

describe("Changes React model", () => {
  it("keeps Git ranges explicit and omits defaults", () => {
    expect(
      rangeFromURL("https://example.test/changes/?base=main&target=abc123"),
    ).toEqual({
      base: "main",
      branchBase: "",
      target: "revision",
      revision: "abc123",
    });
    expect(
      rangeQuery({
        base: "HEAD",
        branchBase: "",
        target: "working-tree",
        revision: "",
      }).toString(),
    ).toBe("");
  });
  it("filters and groups the server report without deriving domain semantics", () => {
    expect(
      visibleChanges(
        changes,
        { query: "", status: "", scope: "documents" },
        "en",
      ).map((item) => item.path),
    ).toEqual(["docs/a.md"]);
    expect(
      buildChangeTree(changes).directories.get("docs")?.files[0].path,
    ).toBe("docs/a.md");
  });
  it("keeps unchanged Mermaid lines and marks changed lines", () => {
    expect(lineDiff("flowchart LR\nA-->B", "flowchart LR\nA-->C")).toBe(
      "  flowchart LR\n+ A-->C\n- A-->B",
    );
  });
  it("keeps picker and discussion-linked files across report replacement", () => {
    const linked = { path: "notes.md", status: "linked", lines: { added: 0, deleted: 0 } } as Change;
    expect(mergeLinkedFiles(changes, [linked], ["docs/context.md"]).map((item) => item.path)).toEqual(["docs/a.md", "server.go", "notes.md", "docs/context.md"]);
    expect(mergeLinkedFiles([changes[0]], [linked], []).map((item) => item.path)).toEqual(["docs/a.md", "notes.md"]);
  });
  it("invalidates file detail when the report revision changes", () => {
    const first = detailCacheKey(new URLSearchParams("base=main"), { ...changes[0], _revision: "one" });
    const next = detailCacheKey(new URLSearchParams("base=main"), { ...changes[0], _revision: "two" });
    expect(next).not.toBe(first);
  });
});
