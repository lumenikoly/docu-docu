import { useCallback, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import { DiscussionPanel } from "../discussions/components";
import { ChangesDetail } from "./detail";
import { ChangesWorkspace } from "./workspace";

function App() {
  const [selectedPath, setSelectedPath] = useState("");
  const [linkedPaths, setLinkedPaths] = useState<string[]>([]);
  const controller = useMemo(() => new AbortController(), []);
  const page = window.ToudocuPage;
  const review =
    page?.runtime === "serve" && page.capabilities?.review
      ? page.endpoints?.review || ""
      : "";
  const writable =
    (new URLSearchParams(location.search).get("target") || "working-tree") ===
    "working-tree";
  useEffect(() => {
    const toggle = document.querySelector<HTMLElement>(
      "[data-discussions-toggle]",
    );
    if (toggle) toggle.hidden = !writable;
    return () => controller.abort();
  }, [controller, writable]);
  const requestURL = useCallback((endpoint: string, path: string) => {
    const source = new URLSearchParams(location.search);
    const params = new URLSearchParams();
    for (const key of ["base", "branchBase", "target"])
      if (source.has(key)) params.set(key, source.get(key)!);
    return `${endpoint}${path}${params.size ? `?${params}` : ""}`;
  }, []);
  const compose = useCallback(
    (target: any, returnElement?: HTMLElement | null) => {
      const normalized = {
        kind: target.kind || "file",
        path: target.path,
        ...(target.start
          ? { range: { start: target.start, end: target.end } }
          : {}),
      };
      document.dispatchEvent(
        new CustomEvent("toudocu:changes-compose", {
          detail: { target: normalized, quote: target.quote, returnElement },
        }),
      );
    },
    [],
  );
  const selection = useCallback(
    (change: any) => setSelectedPath(change?.path || ""),
    [],
  );
  const discussions = useCallback(
    (state: any) =>
      setLinkedPaths([
        ...new Set(
          (state?.session?.discussions || [])
            .flatMap((item: any) => [item.target?.path, item.placement?.path])
            .filter(Boolean) as string[],
        ),
      ]),
    [],
  );
  return (
    <>
      <ChangesWorkspace
        linkedPaths={linkedPaths}
        onSelectionChange={selection}
        renderDetail={(change, tab, onTab) =>
          change ? (
            <ChangesDetail
              change={change}
              tab={tab}
              onTab={onTab}
              onCompose={compose}
            />
          ) : null
        }
      />
      {review && writable && (
        <DiscussionPanel
          endpoint={review}
          pageId=""
          path={selectedPath}
          title={selectedPath}
          signal={controller.signal}
          initiallyOpen={false}
          requestURL={requestURL}
          targetKind="file"
          variant="changes"
          composeEvent="toudocu:changes-compose"
          onStateChange={discussions}
        />
      )}
    </>
  );
}

const root = document.querySelector<HTMLElement>("[data-changes-root]");
if (root) createRoot(root).render(<App />);
