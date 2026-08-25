import "../core/bootstrap";
import "../core/runtime";
import "../core/serve-runtime";
import { islandHost, registerIsland } from "../core/react/island-host";
import "../core/serve-navigation";

registerIsland("discussions", () => import("../features/discussions/island"));
registerIsland("roadmap", () => import("../features/roadmap/island"));
islandHost.discover();
const restoreDiscussions = () => {
  try {
    if (sessionStorage.getItem("toudocu-discussions-activated")) void islandHost.activate("project-discussions");
  } catch { /* storage can be disabled */ }
};
restoreDiscussions();
document.addEventListener("toudocu:pagechange", () => queueMicrotask(restoreDiscussions));

document.addEventListener("click", (event) => {
  const toggle = (event.target as Element).closest<HTMLElement>("[data-discussions-toggle]");
  if (!toggle || toggle.getAttribute("aria-expanded") === "true") return;
  document.querySelector<HTMLElement>("[data-td-island='discussions']")!.dataset.discussionsOpen = "";
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
