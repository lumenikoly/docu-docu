import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, test, vi } from "vitest";
import { Dialog, IconButton, Menu, Tabs } from "../src/ui";
import { ActionError, applyTurnEvent, stopConflictDetails, stripControlSequences } from "../src/features/agent-console/island";
import { mountTaskActions, type Projection } from "../src/features/task-actions";

afterEach(() => { cleanup(); document.body.replaceChildren(); vi.unstubAllGlobals(); delete window.ToudocuPage; });

describe("shared UI accessibility", () => {
  test("renders agent and command text without terminal control sequences", () => {
    expect(stripControlSequences("safe\u001b[31m red\u001b[0m\u0007\u001b]8;;https://example.com\u0007link\u001b]8;;\u0007")).toBe("safe redlink");
  });

  test("uses authoritative pending counts from a stop conflict", () => {
    expect(stopConflictDetails(new ActionError("pending", 409, { queued: 2, notSent: 1 }))).toEqual({ queued: 2, notSent: 1 });
  });

  test("clears the active response when its turn completes", () => {
    expect(applyTurnEvent({ active: true, status: "running", activeTurn: "turn-1" }, { type: "turn_completed", turnID: "turn-1" })).toEqual({ active: true, status: "idle", activeTurn: undefined });
  });

  test("requires an accessible IconButton label", () => {
    expect(() => render(<IconButton />)).toThrow("aria-label");
  });

  test("moves through tabs with the keyboard", async () => {
    const user = userEvent.setup();
    render(<Tabs.Root defaultValue="one"><Tabs.List><Tabs.Tab value="one">One</Tabs.Tab><Tabs.Tab value="two">Two</Tabs.Tab></Tabs.List><Tabs.Panel value="one">First</Tabs.Panel><Tabs.Panel value="two">Second</Tabs.Panel></Tabs.Root>);
    await user.click(screen.getByRole("tab", { name: "One" }));
    await user.keyboard("{ArrowRight}");
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "Two" }));
    await user.keyboard("{Enter}");
    expect(screen.getByRole("tab", { name: "Two" }).getAttribute("aria-selected")).toBe("true");
  });

  test("traps dialog focus, closes on Escape, and restores focus", async () => {
    const user = userEvent.setup();
    render(<Dialog.Root><Dialog.Trigger>Open</Dialog.Trigger><Dialog.Portal><Dialog.Popup><Dialog.Title>Confirm</Dialog.Title><Dialog.Close>Close</Dialog.Close></Dialog.Popup></Dialog.Portal></Dialog.Root>);
    const trigger = screen.getByRole("button", { name: "Open" });
    await user.click(trigger);
    expect(screen.getByRole("dialog").contains(document.activeElement)).toBe(true);
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement).toBe(trigger);
  });

  test("opens a labeled menu and supports arrow navigation", async () => {
    const user = userEvent.setup();
    render(<Menu.Root><Menu.Trigger>Actions</Menu.Trigger><Menu.Portal><Menu.Positioner><Menu.Popup><Menu.Item>Open</Menu.Item><Menu.Item>Copy</Menu.Item></Menu.Popup></Menu.Positioner></Menu.Portal></Menu.Root>);
    await user.click(screen.getByRole("button", { name: "Actions" }));
    await user.keyboard("{ArrowDown}");
    expect(document.activeElement?.textContent).toBe("Open");
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("menu")).toBeNull();
  });

  test("keeps the task visible after launch and opens the console from the active agent indicator", async () => {
    const projection: Projection = {
      schemaVersion: 1,
      task: { id: "TASK-X", status: "in-progress", workspaceState: "in-progress", digest: "digest" },
      agent: { relation: "current-task", status: "running", needsAttention: false },
      actions: [{ id: "next", label: "Next", input: "none", deliveries: [{ type: "agent-console", available: true }, { type: "handoff", available: true }] }],
    };
    window.ToudocuPage = { ui: { locale: "en" }, endpoints: { taskActions: "/tasks" } } as typeof window.ToudocuPage;
    vi.stubGlobal("CSS", { escape: (value: string) => value });
    const writeText = vi.fn(async () => undefined); Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    vi.stubGlobal("fetch", vi.fn(async (_input, init?: RequestInit) => {
      const payload = init?.method === "POST" && String(init.body).includes('"handoff"') ? { schemaVersion: 1, actionID: "next", delivery: "handoff", handoff: { instruction: "prompt" }, projection } : projection;
      return new Response(JSON.stringify(payload), { status: 200, headers: { "Content-Type": "application/json" } });
    }));
    document.body.innerHTML = '<div data-task-actions data-task-id="TASK-X"></div>';
    let opens = 0; const opened = () => { opens += 1; }; document.addEventListener("toudocu:agent-open", opened);
    const controller = new AbortController(); mountTaskActions(controller.signal);
    await waitFor(() => expect(screen.getByRole("button", { name: /Agent is working/ })).toBeTruthy());
    await userEvent.click(screen.getByRole("button", { name: /Ask what to do next.*Agent Console/ }));
    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2)); expect(opens).toBe(0);
    await userEvent.click(screen.getByRole("button", { name: /Ask what to do next.*external agent/ }));
    await waitFor(() => expect(screen.getByText("Prompt copied")).toBeTruthy()); expect(writeText).toHaveBeenCalledWith("prompt");
    await userEvent.click(screen.getByRole("button", { name: /Agent is working/ })); expect(opens).toBe(1);
    controller.abort(); document.removeEventListener("toudocu:agent-open", opened);
  });
});
