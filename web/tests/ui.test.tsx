import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, test } from "vitest";
import { Dialog, IconButton, Menu, Tabs } from "../src/ui";

afterEach(cleanup);

describe("shared UI accessibility", () => {
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
});
