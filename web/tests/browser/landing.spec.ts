import { expect, test } from "@playwright/test";
import { readFileSync, statSync } from "node:fs";
import { createServer, type Server } from "node:http";
import { dirname, extname, join, normalize, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../../../landing");

async function serveLanding(): Promise<{ server: Server; origin: string }> {
  const server = createServer((request, response) => {
    const relative = decodeURIComponent(new URL(request.url ?? "/", "http://localhost").pathname.slice(1)) || "index.html";
    let target = normalize(join(root, relative));
    if (target !== root && !target.startsWith(`${root}${sep}`)) return response.writeHead(403).end();
    try {
      if (statSync(target).isDirectory()) target = join(target, "index.html");
      const contentType = ({ ".css": "text/css", ".html": "text/html", ".js": "text/javascript", ".svg": "image/svg+xml" } as Record<string, string>)[extname(target)];
      if (contentType) response.setHeader("Content-Type", contentType);
      response.end(readFileSync(target));
    } catch {
      response.writeHead(404).end();
    }
  });
  await new Promise<void>((resolveListen) => server.listen(0, "127.0.0.1", resolveListen));
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("landing server did not start");
  return { server, origin: `http://127.0.0.1:${address.port}/` };
}

test("landing selects a locale and switches it", async ({ browser }) => {
  const hosted = await serveLanding();
  const context = await browser.newContext({ locale: "ru-RU" });
  const page = await context.newPage();
  try {
    await page.goto(hosted.origin);
    await expect(page).toHaveURL(`${hosted.origin}ru/`);
    await expect(page.locator("h1")).toBeVisible();
    await page.locator("[data-locale-choice='en']").click();
    await expect(page).toHaveURL(`${hosted.origin}en/`);
  } finally {
    await context.close();
    await new Promise<void>((resolveClose) => hosted.server.close(() => resolveClose()));
  }
});

test("landing copies the install command", async ({ page, context }) => {
  await context.addInitScript(() => Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: async (text: string) => { (window as any).__copiedText = text; } },
  }));
  const hosted = await serveLanding();
  try {
    await page.goto(`${hosted.origin}en/`);
    await page.locator(".copy-button").first().click();
    await expect.poll(() => page.evaluate(() => (window as any).__copiedText)).toContain("install.sh | sh");
  } finally {
    await new Promise<void>((resolveClose) => hosted.server.close(() => resolveClose()));
  }
});
