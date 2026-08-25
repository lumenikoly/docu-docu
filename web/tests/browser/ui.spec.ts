import { expect, test } from "@playwright/test";

test("UI gallery exposes labels and respects reduced motion", async ({ page }) => {
  await page.goto("http://127.0.0.1:4173/dev/ui.html");
  await expect(page.getByRole("heading", { name: "Toudocu UI" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Закрыть", exact: true })).toBeVisible();
  await expect(page.getByRole("tablist", { name: "Разделы примера" })).toBeVisible();
  await expect(page.getByRole("combobox", { name: "Статус" })).toBeVisible();
  await expect(page.getByRole("status", { name: "Загрузка" })).toHaveCSS("animation-name", "ui-spin");
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.reload();
  await expect(page.getByRole("status", { name: "Загрузка" })).toHaveCSS("animation-name", "none");
  await page.getByRole("button", { name: "Открыть диалог" }).click();
  await expect(page.getByRole("dialog", { name: "Подтверждение" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});
