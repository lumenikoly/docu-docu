import "../core/bootstrap";
import "../core/runtime";
import "../core/serve-runtime";
import { islandHost, registerIsland } from "../core/react/island-host";
import "../core/serve-navigation";

registerIsland("discussions", () => import("../features/discussions/island"));
registerIsland("roadmap", () => import("../features/roadmap/island"));
registerIsland("agent-console", () => import("../features/agent-console/island"));
islandHost.discover();
const restoreDiscussions = () => {
  try {
    if (document.querySelector("[data-td-island-instance='project-discussions']") && sessionStorage.getItem("toudocu-discussions-activated")) void islandHost.activate("project-discussions");
  } catch { /* storage can be disabled */ }
};
restoreDiscussions();
document.addEventListener("toudocu:pagechange", () => queueMicrotask(restoreDiscussions));

document.addEventListener("click", (event) => {
  const toggle = (event.target as Element).closest<HTMLElement>("[data-discussions-toggle]");
  const island = document.querySelector<HTMLElement>("[data-td-island='discussions']");
  if (!toggle || !island || toggle.getAttribute("aria-expanded") === "true") return;
  island.dataset.discussionsOpen = "";
  void islandHost.activate("project-discussions");
});

document.addEventListener("pointerup", (event) => {
  const island = document.querySelector<HTMLElement>("[data-td-island='discussions']");
  const selection = window.getSelection();
  if (!island || island.dataset.tdIslandState || !selection || selection.isCollapsed || !selection.toString().trim()) return;
  const target = event.target as Element;
  if (!target.closest(".page-content")) return;
  void islandHost.activate("project-discussions").then(() => target.dispatchEvent(new PointerEvent("pointerup", { bubbles: true })));
});
