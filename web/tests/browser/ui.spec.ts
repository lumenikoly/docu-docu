import { expect, test } from "@playwright/test";

test("UI page opens and closes its dialog", async ({ page }) => {
  await page.goto("/dev/ui.html");
  await expect(page.getByRole("heading", { name: "Toudocu UI" })).toBeVisible();
  await page.getByRole("button", { name: "Открыть диалог" }).click();
  await expect(page.getByRole("dialog", { name: "Подтверждение" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
});
