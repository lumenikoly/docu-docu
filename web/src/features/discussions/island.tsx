import { createRoot } from "react-dom/client";
import type { IslandMount } from "../../core/react/island-host";
import { DiscussionPanel } from "./components";

export { DiscussionComposer, DiscussionPanel, useDiscussionState } from "./components";
export type { Discussion, DiscussionComposerState, DiscussionMessage, DiscussionState } from "./components";

export const mount: IslandMount = (element, { signal, onCleanup }) => {
  try { sessionStorage.setItem("toudocu-discussions-activated", "1"); } catch { /* storage can be disabled */ }
  const root = createRoot(element);
  onCleanup(() => root.unmount());
  const open = element.hasAttribute("data-discussions-open");
  delete element.dataset.discussionsOpen;
  const page = window.ToudocuPage;
  const context = document.querySelector<HTMLElement>("[data-copy-document-context]");
  root.render(<DiscussionPanel endpoint={page?.endpoints?.review || ""} pageId={page?.page?.id || ""} path={context?.dataset.documentContextPath || ""} title={context?.dataset.documentContextTitle || ""} signal={signal} initiallyOpen={open} composeEvent="toudocu:discussion-compose" />);
};
