import { expect, test, type Page } from "@playwright/test";
import { spawn, spawnSync, type ChildProcess } from "node:child_process";
import { cpSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { createServer, type Server } from "node:http";
import { tmpdir } from "node:os";
import { dirname, extname, join, normalize, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
let cliPath = "";

function run(command: string, args: string[], cwd = repo): void {
  const result = spawnSync(command, args, { cwd, encoding: "utf8" });
  if (result.status !== 0) {
    throw new Error(`${command} ${args.join(" ")} failed\n${result.stdout}\n${result.stderr}`);
  }
}

function testCLI(): string {
  if (!cliPath) {
    cliPath = join(mkdtempSync(join(tmpdir(), "toudocu-browser-cli-")), process.platform === "win32" ? "toudocu.exe" : "toudocu");
    run("go", ["build", "-o", cliPath, "./cmd/toudocu"]);
  }
  return cliPath;
}

async function stopChild(child: ChildProcess): Promise<void> {
  if (child.exitCode !== null || child.signalCode !== null) return;
  await new Promise<void>((resolveExit, rejectExit) => {
    child.once("exit", () => resolveExit());
    child.once("error", rejectExit);
    child.kill("SIGTERM");
  });
}

function mime(path: string): string {
  return ({ ".html": "text/html", ".css": "text/css", ".js": "text/javascript", ".json": "application/json", ".svg": "image/svg+xml" } as Record<string, string>)[extname(path)] ?? "application/octet-stream";
}

async function staticServer(root: string, mount = "/docs/"): Promise<{ server: Server; origin: string }> {
  const server = createServer((request, response) => {
    const url = new URL(request.url ?? "/", "http://127.0.0.1");
    if (!url.pathname.startsWith(mount)) {
      response.writeHead(404).end();
      return;
    }
    const relative = decodeURIComponent(url.pathname.slice(mount.length)) || "index.html";
    const target = normalize(join(root, relative));
    if (target !== root && !target.startsWith(`${root}${sep}`)) {
      response.writeHead(403).end();
      return;
    }
    try {
      response.setHeader("Content-Type", mime(target));
      response.end(readFileSync(target));
    } catch {
      response.writeHead(404).end();
    }
  });
  await new Promise<void>((resolveListen) => server.listen(0, "127.0.0.1", resolveListen));
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("static server did not bind TCP");
  return { server, origin: `http://127.0.0.1:${address.port}${mount}` };
}

async function waitForHTTP(url: string): Promise<void> {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    try {
      if ((await fetch(url)).ok) return;
    } catch {
      // The listener is still starting.
    }
    await new Promise((resolveWait) => setTimeout(resolveWait, 100));
  }
  throw new Error(`timed out waiting for ${url}`);
}

async function exerciseStaticPortal(page: Page, origin: string): Promise<void> {
  const responses: string[] = [];
  page.on("response", (response) => responses.push(response.url()));
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto(`${origin}index.html`);
  await expect(page.locator("main")).toContainText("Toudocu");
  await expect(page.locator("script#toudocu-page")).toHaveCount(1);
  await expect(page.locator("[data-server-rebuild], [data-roadmap-add], a[href^='/_toudocu/editor'], a[href='/changes/']")).toHaveCount(0);
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).scrollBehavior)).toBe("auto");
  await page.locator("[data-global-search]").fill("Toudocu");
  await expect(page.locator("[data-search-results]")).not.toBeEmpty();
  await page.locator("[data-color-scheme-select]").selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-color-scheme", "dark");
  await page.goto(`${origin}work/TASK-CLI-003.html`);
  const taskFolder = page.locator('[data-nav-folder="task-task-cli-002"]');
  const taskToggle = taskFolder.locator('[data-nav-folder-toggle]');
  const taskChildren = taskFolder.locator(":scope > ul");
  await expect(taskFolder).toContainText("TASK-CLI-003");
  await expect(taskToggle).toHaveAttribute("aria-expanded", "true");
  await expect(taskToggle).toHaveAttribute("aria-label", /Свернуть подзадачи TASK-CLI-002/);
  await taskToggle.click();
  await expect(taskToggle).toHaveAttribute("aria-expanded", "false");
  await expect(taskToggle).toHaveAttribute("aria-label", /Развернуть подзадачи TASK-CLI-002/);
  await expect(taskChildren).toBeHidden();
  await taskToggle.click();
  await expect(taskChildren).toBeVisible();
  await page.goto(`${origin}use-cases/UC-DOCS-01.html`);
  await expect(page.locator("main article.doc-content").first()).toContainText("Разработчик");
  await expect.poll(() => page.locator("script#toudocu-page").textContent()).toContain("UC-DOCS-01");
  const tabs = page.locator("[data-usecase-tab]");
  await expect(tabs).toHaveCount(4);
  await tabs.nth(1).click();
  await expect(tabs.nth(1)).toHaveAttribute("aria-selected", "true");
  await tabs.nth(1).press("ArrowRight");
  await expect(tabs.nth(2)).toHaveAttribute("aria-selected", "true");
  await expect(tabs.nth(2)).toBeFocused();
  await page.goto(`${origin}flows/FLOW-DOCS-BUILD.html`);
  await expect(page.locator("[data-mermaid-diagram] svg").first()).toBeVisible();
  await expect.poll(() => responses.some((url) => url.endsWith("/assets/portal.css"))).toBe(true);
  await expect.poll(() => responses.some((url) => url.endsWith("/assets/portal.js"))).toBe(true);
  await page.goto(`${origin}notes.html`);
  await expect(page.locator("[data-mermaid-error]")).toBeVisible();
  await page.goto(`${origin}risks.html`);
  await expect(page.locator(".risk-status")).toContainText("Незакрытых рисков: 2 из 3");
  await expect(page.locator(".risk-status-explanations")).toContainText("Снижается — меры выполняются, риск ещё не закрыт.");
}

test("static portal works over HTTP at root and nested paths", async ({ browser }) => {
  test.slow();
  const fixture = mkdtempSync(join(tmpdir(), "toudocu-static-"));
  cpSync(join(repo, "docs"), join(fixture, "docs"), { recursive: true });
  writeFileSync(join(fixture, "docs", "notes.md"), "# Заметки\n\nТестовая заметка.\n");
  const childTask = join(fixture, "docs", "work", "TASK-CLI-003.md");
  writeFileSync(childTask, readFileSync(childTask, "utf8").replace("taskType: maintenance", "taskType: maintenance\nparentTask: TASK-CLI-002"));
  cpSync(join(repo, ".toudocu"), join(fixture, ".toudocu"), { recursive: true });
  for (const directory of ["web/src/features/task-workspace", "web/src/features/discussions", "web/src/features/changes", "web/src/core/react", "web/src/entries", "web/src/styles", "web/src/ui", "internal/app", "internal/site/templates", "internal/site/i18n", ".github"]) mkdirSync(join(fixture, directory), { recursive: true });
  writeFileSync(join(fixture, "Makefile"), "");
  const output = join(fixture, "site");
  run(testCLI(), ["build", join(fixture, "docs"), "--repository-root", fixture, "-o", output, "--clean"]);
  const notesPage = join(output, "notes.html");
  writeFileSync(notesPage, readFileSync(notesPage, "utf8").replace("</article>", '<figure data-mermaid-container><pre class="mermaid" data-mermaid-diagram>graph TD\nbroken[</pre><p data-mermaid-error hidden>Не удалось отобразить диаграмму.</p></figure></article>'));
  for (const mount of ["/", "/docs/"]) {
    const hosted = await staticServer(output, mount);
    const page = await browser.newPage();
    try {
      await exerciseStaticPortal(page, hosted.origin);
    } finally {
      await page.close();
      await new Promise<void>((resolveClose) => hosted.server.close(() => resolveClose()));
    }
  }
});

test("Portal and workspaces share visual language, rebuild, and Changes", async ({ page }) => {
  test.setTimeout(180_000);
  const fixture = mkdtempSync(join(tmpdir(), "toudocu-serve-"));
  cpSync(join(repo, "docs"), join(fixture, "docs"), { recursive: true });
  writeFileSync(join(fixture, "docs", "notes.md"), "# Заметки\n\nТестовая заметка.\n");
  cpSync(join(repo, ".toudocu"), join(fixture, ".toudocu"), { recursive: true });
  run("git", ["init", "-q"], fixture);
  run("git", ["config", "user.email", "browser@example.invalid"], fixture);
  run("git", ["config", "user.name", "Browser Test"], fixture);
  run("git", ["add", "."], fixture);
  run("git", ["commit", "-qm", "fixture"], fixture);
  const portServer = createServer();
  await new Promise<void>((resolveListen) => portServer.listen(0, "127.0.0.1", resolveListen));
  const address = portServer.address();
  if (!address || typeof address === "string") throw new Error("failed to reserve port");
  const port = address.port;
  await new Promise<void>((resolveClose) => portServer.close(() => resolveClose()));
  const output = mkdtempSync(join(tmpdir(), "toudocu-serve-site-"));
  const child: ChildProcess = spawn(testCLI(), ["serve", join(fixture, "docs"), "--repository-root", fixture, "-o", output, "--host", "127.0.0.1", "--port", String(port)], { cwd: repo, stdio: "pipe" });
  const origin = `http://127.0.0.1:${port}`;
  const terminalAssets: string[] = [];
  page.on("request", (request) => { if (/terminal.*\.(?:js|css)$/.test(request.url())) terminalAssets.push(request.url()); });
  let latestVersion = "0.0.2";
  try {
    await waitForHTTP(origin);
    await page.route("**/_toudocu/api/version", (route) => route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        schemaVersion: 1,
        currentVersion: "0.0.1",
        status: "update-available",
        latestVersion,
        releaseURL: `https://github.com/lumenikoly/toudocu/releases/tag/${latestVersion}`,
      }),
    }));
    await page.route("**/assets/*.js", async (route) => {
      if (!route.request().url().endsWith("/appearance.js")) await new Promise((resolveDelay) => setTimeout(resolveDelay, 250));
      await route.continue();
    });
    await page.addInitScript(() => {
      localStorage.setItem("toudocu-site-theme", "paper");
      localStorage.setItem("toudocu-color-scheme", "dark");
      localStorage.setItem("toudocu-accent", "violet");
      requestAnimationFrame(() => {
        (window as any).__toudocuFirstFrame = {
          siteTheme: document.documentElement.dataset.siteTheme,
          colorScheme: document.documentElement.dataset.colorScheme,
          theme: document.documentElement.dataset.theme,
          accent: document.documentElement.dataset.accent,
        };
      });
    });
    await page.goto(origin);
    await expect(page.locator("[data-td-island-instance='agent-console']")).toHaveAttribute("data-td-island-state", "mounted");
    await expect.poll(() => page.evaluate(() => (window as any).__toudocuFirstFrame)).toEqual({ siteTheme: "paper", colorScheme: "dark", theme: "dark", accent: "violet" });
    await expect(page.locator("[data-server-rebuild]")).toBeVisible();
    const agentToggle = page.locator("[data-agent-console-toggle]");
    await expect(agentToggle).toBeVisible();
    await expect(agentToggle.locator("[data-agent-console-summary]")).toContainText("Выключен");
    await page.setViewportSize({ width: 390, height: 844 });
    await agentToggle.click();
    await expect(page.locator("#agent-console-panel")).toHaveAttribute("role", "dialog");
    await expect(page.locator(".agent-console-setup")).toContainText("codex");
    await expect(page.locator(".agent-console-setup select")).toHaveValue("default");
    await expect(page.locator("[data-agent-console-close]")).toBeFocused();
    await expect(page.locator(".site-layout")).toHaveJSProperty("inert", true);
    await expect(page.locator(".site-header")).toHaveJSProperty("inert", true);
    await page.getByRole("button", { name: "Запустить агента" }).focus();
    await page.keyboard.press("Tab");
    await expect(page.locator("[data-agent-console-close]")).toBeFocused();
    await expect(page.getByRole("tab", { name: "Agent View" })).toHaveAttribute("aria-selected", "true");
    await page.getByRole("tab", { name: "Agent View" }).press("ArrowRight");
    await expect(page.getByRole("tab", { name: "Command Output" })).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(agentToggle).toBeFocused();
    await expect(page.locator(".site-layout")).toHaveJSProperty("inert", false);
    await expect(page.locator(".site-header")).toHaveJSProperty("inert", false);
    await page.setViewportSize({ width: 1280, height: 720 });
    await agentToggle.click();
    await expect(page.locator(".agent-console-columns .agent-console-view")).toBeVisible();
    await expect(page.locator(".agent-console-columns .agent-command-view")).toBeVisible();
    expect(terminalAssets).toEqual([]);
    await page.keyboard.press("Escape");
    await page.locator("[data-agent-terminal-toggle]").click();
    await expect(page.getByRole("button", { name: "Закрыть Agent Console" })).toHaveCount(0);
    await expect(page.getByText("Терминал проекта", { exact: true })).toHaveCount(1);
    await expect(page.getByRole("button", { name: "Запустить Codex", exact: true })).toHaveCount(0);
    await page.getByRole("button", { name: "Запустить терминал" }).click();
    await expect(page.getByRole("button", { name: "Остановить терминал", exact: true })).toBeVisible();
    await expect.poll(() => terminalAssets.length).toBeGreaterThan(0);
    const terminalInput = page.locator(".xterm-helper-textarea");
    await terminalInput.pressSequentially("set -- $(stty size); if [ \"$1\" -gt 0 ] && [ \"$2\" -gt 0 ]; then printf '\\033[?1049h\\033[2J\\033[Hfullscreen-ready'; fi");
    await terminalInput.press("Enter");
    await expect(page.locator(".xterm-screen")).toContainText("fullscreen-ready");
    await page.locator("[data-agent-terminal-toggle]").click();
    await expect(page.locator("#agent-console-panel")).not.toBeVisible();
    await expect.poll(async () => (await fetch(`${origin}/_toudocu/api/agent-console/`).then((response) => response.json())).state.terminal.active).toBe(true);
    await page.locator("[data-agent-terminal-toggle]").click();
    await page.getByRole("button", { name: "Остановить терминал", exact: true }).click();
    await page.locator("[data-agent-terminal-toggle]").click();
    const updateNotice = page.locator("[data-update-notice]");
    await expect(updateNotice).toContainText("Доступна Toudocu 0.0.2");
    await expect(updateNotice).toContainText("У вас 0.0.1");
    await expect(updateNotice.locator("a")).toHaveAttribute("href", "https://github.com/lumenikoly/toudocu/releases/tag/0.0.2");
    await expect(updateNotice.locator("a")).toHaveAttribute("rel", "noopener noreferrer");
    const portalFonts: Record<string, Record<string, string>> = {};
    for (const siteTheme of ["classic", "paper", "terminal"]) {
      await page.locator("[data-site-theme-select]").selectOption(siteTheme);
      portalFonts[siteTheme] = await page.evaluate(() => {
        const codeProbe = document.createElement("span");
        codeProbe.style.fontFamily = "var(--td-font-mono)";
        document.body.append(codeProbe);
        const mono = getComputedStyle(codeProbe).fontFamily;
        codeProbe.remove();
        return {
          body: getComputedStyle(document.body).fontFamily,
          interface: getComputedStyle(document.querySelector(".site-header")!).fontFamily,
          heading: getComputedStyle(document.querySelector("h1")!).fontFamily,
          mono,
          surface: getComputedStyle(document.documentElement).getPropertyValue("--td-surface").trim(),
        };
      });
    }
    expect(portalFonts.paper.surface).not.toBe(portalFonts.classic.surface);
    const tokenVariants = await page.evaluate(() => {
      const appearance = (window as any).ToudocuAppearance;
      const value = (name: string) => getComputedStyle(document.documentElement).getPropertyValue(name).trim();
      appearance.set("density", "comfortable");
      const comfortable = value("--td-control-height");
      appearance.set("density", "compact");
      const compact = value("--td-control-height");
      appearance.set("colorScheme", "light");
      const light = value("--td-surface");
      appearance.set("colorScheme", "dark");
      const dark = value("--td-surface");
      return { comfortable, compact, light, dark };
    });
    expect(tokenVariants.compact).not.toBe(tokenVariants.comfortable);
    expect(tokenVariants.dark).not.toBe(tokenVariants.light);
    await page.goto(`${origin}/work/index.html`);
    await expect(page.locator(".task-workspace-toolbar")).toBeVisible();
    await expect(page.locator(".task-workspace-board")).toBeVisible();
    await expect(page.locator(".task-workspace-views [aria-current='true']")).toBeVisible();
    await expect(page.locator("[data-task-agent-setup]")).toContainText("codex");
    const prepareSkill = page.locator("[data-task-agent-setup] button");
    if (await prepareSkill.isVisible()) {
      page.once("dialog", (dialog) => dialog.accept());
      await prepareSkill.click();
      await expect(page.locator("[data-task-agent-setup]")).toContainText("installed");
    }
    await expect(page.locator("[data-task-id='TASK-AGENT-010'] [data-task-agent-action='explain-problems']").first()).toBeVisible();
    await expect(page.locator("[data-task-id='TASK-AGENT-006'] [data-task-agent-action]")).toHaveCount(0);
    await page.goto(origin);
    await page.evaluate(() => {
      (window as any).__toudocuLifecycle = [];
      document.addEventListener("toudocu:pagebeforechange", () => (window as any).__toudocuLifecycle.push("before"));
      document.addEventListener("toudocu:pagechange", () => (window as any).__toudocuLifecycle.push("change"));
    });
    await page.locator('main a.recommended-entry[href="architecture/overview.html"]').click();
    await page.waitForURL("**/architecture/overview.html");
    await expect(updateNotice).toBeVisible();
    await expect.poll(() => page.evaluate(() => window.ToudocuPage?.page.path)).toBe("architecture/overview.html");
    await expect.poll(() => page.evaluate(() => window.ToudocuPage?.portal.dataBase)).toBe("../data/");
    await page.locator("a.brand").click();
    await page.waitForURL("**/index.html");
    await page.locator('main a.recommended-entry[href="architecture/overview.html"]').click();
    await page.waitForURL("**/architecture/overview.html");
    await expect.poll(() => page.evaluate(() => (window as any).__toudocuLifecycle)).toEqual(["before", "change", "before", "change", "before", "change"]);
    await page.locator("[data-global-search]").fill("Toudocu");
    await expect(page.locator("[data-search-results]")).not.toBeEmpty();
    await page.goto(origin);
    await updateNotice.getByRole("button", { name: "Скрыть уведомление о версии 0.0.2" }).click();
    await expect(updateNotice).toHaveCount(0);
    await page.reload();
    await expect(page.locator("[data-update-notice]")).toHaveCount(0);
    latestVersion = "0.0.3";
    await page.reload();
    await expect(page.locator("[data-update-notice]")).toContainText("Доступна Toudocu 0.0.3");
    const rebuilt = page.waitForEvent("framenavigated", (frame) => frame === page.mainFrame());
    await page.locator("[data-server-rebuild]").click();
    await expect(page.locator("[data-server-rebuild]")).not.toHaveClass(/is-rebuilding/);
    await rebuilt;
    await page.waitForLoadState("load");

    const roadmapPath = join(fixture, "docs", "roadmap.md");
    await page.goto(`${origin}/roadmap.html`);
    const initialRoadmapTotal = Number((await page.locator(".progress-label").textContent())?.match(/из (\d+)/)?.[1]);
    expect(initialRoadmapTotal).toBeGreaterThan(0);
    await expect(page.locator("[data-td-island-instance='roadmap-add']")).toHaveAttribute("data-td-island-state", "mounted");
    const roadmapTrigger = page.locator("[data-roadmap-add]");
    await roadmapTrigger.focus();
    await roadmapTrigger.press("Enter");
    await expect(page.locator("[data-roadmap-dialog]")).toBeVisible();
    await page.locator("[data-roadmap-dialog]").press("Escape");
    await expect(page.locator("[data-roadmap-dialog]")).not.toBeVisible();
    await expect(roadmapTrigger).toBeFocused();
    await page.locator("a.brand").click();
    await page.waitForURL("**/index.html");
    await expect(page.locator("[data-roadmap-dialog]")).toHaveCount(0);
    await page.locator('a[href="roadmap.html"]').first().click();
    await page.waitForURL("**/roadmap.html");
    await expect(page.locator('[data-td-island-instance="roadmap-add"]')).toHaveCount(1);
    await roadmapTrigger.click();
    await expect(page.locator("[data-roadmap-dialog]")).toHaveCount(1);
    const roadmapDialog = page.locator("[data-roadmap-dialog]");
    await expect(roadmapDialog.locator('input[name="id"]')).toHaveValue("DLV-ROADMAP-002");
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.locator("[data-update-notice]")).toBeInViewport();
    await expect(page.locator("[data-update-notice] a")).toBeVisible();
    await expect(page.locator("[data-update-notice] button")).toBeVisible();
    await expect(roadmapDialog).toBeInViewport();
    await expect(roadmapDialog.locator('input[name="text"]')).toBeVisible();
    await page.setViewportSize({ width: 1280, height: 720 });
    await roadmapDialog.locator('input[name="id"]').fill("DLV-BROWSER-001");
    await roadmapDialog.locator('input[name="text"]').fill("Browser value survives conflict.");
    writeFileSync(roadmapPath, `${readFileSync(roadmapPath, "utf8")}\n<!-- toudocu:section roadmap-stage -->\n<!-- toudocu\nstatus: planned\n-->\n\n## Browser stage\n\n- Status: Planned\n\n- [ ] \`DLV-ROADMAP-007\` External result.\n`);
    await roadmapDialog.locator('button[type="submit"]').click();
    await expect(roadmapDialog.locator("[data-state='conflict']")).toContainText("введённые данные сохранены");
    await expect(roadmapDialog.locator('input[name="id"]')).toHaveValue("DLV-BROWSER-001");
    await expect(roadmapDialog.locator('input[name="text"]')).toHaveValue("Browser value survives conflict.");
    await expect(roadmapDialog.locator('select[name="stageAnchor"] option[value="browser-stage"]')).toHaveCount(1);
    await expect(roadmapDialog.locator(".roadmap-id-hint")).toContainText("DLV-ROADMAP-008");
    await roadmapDialog.locator('select[name="stageAnchor"]').selectOption("browser-stage");
    await roadmapDialog.locator('input[name="text"]').fill("Browser-added deliverable.");
    await roadmapDialog.locator('button[type="submit"]').click();
    await page.waitForURL("**/roadmap.html#browser-stage");
    await expect(page.locator("#browser-stage").locator("xpath=..")).toContainText("DLV-BROWSER-001");
    await expect(page.locator(".progress-label")).toContainText(`из ${initialRoadmapTotal + 2}`);

    await page.goto(`${origin}/_toudocu/editor/`);
    await expect.poll(() => page.evaluate(() => (window as any).__toudocuFirstFrame)).toEqual({ siteTheme: "paper", colorScheme: "dark", theme: "dark", accent: "violet" });
    await expect(page.locator("[data-editor-root]")).toBeVisible();
    await expect(page.locator("[data-td-island-instance='agent-console']")).toHaveAttribute("data-td-island-state", "mounted");
    const filePath = join(fixture, "docs", "notes.md");
    writeFileSync(filePath, `${readFileSync(filePath, "utf8")}\nExternal edit.\n`);
    writeFileSync(join(fixture, "server.go"), "package main\n\nfunc main() {}\n");

    await page.route("**/_toudocu/api/changes?**", async (route) => {
      const response = await route.fetch();
      if (response.status() === 304) {
        await route.fulfill({ response });
        return;
      }
      const report = await response.json();
      for (const change of report.changes ?? []) {
        if (change.path.endsWith("notes.md")) {
          change.semanticDiffAvailable = false;
          change.semanticChanges = [];
          change.diagnostics = [...(change.diagnostics ?? []), { severity: "error", code: "semantic-unavailable", message: "semantic unavailable" }];
        }
      }
      await route.fulfill({ response, json: report });
    });
    await page.route("**/_toudocu/api/changes/review/repository/changes?**", async (route) => {
      const response = await route.fetch();
      if (response.status() === 304) {
        await route.fulfill({ response });
        return;
      }
      const report = await response.json();
      for (const file of report.files ?? []) {
        if (file.path.endsWith("notes.md") && file.documentation) {
          file.documentation.semanticDiffAvailable = false;
          file.documentation.semanticChanges = [];
          file.documentation.diagnostics = [...(file.documentation.diagnostics ?? []), { severity: "error", code: "semantic-unavailable", message: "semantic unavailable" }];
        }
      }
      await route.fulfill({ response, json: report });
    });
    await page.route("**/_toudocu/api/changes/review/repository/file?**", async (route) => {
      const response = await route.fetch();
      const detail = await response.json();
      if (new URL(route.request().url()).searchParams.get("path")?.endsWith("notes.md")) {
        delete detail.renderedBefore;
        delete detail.renderedCurrent;
      }
      await route.fulfill({ response, json: detail });
    });
    await page.route("**/_toudocu/api/changes/render?**", (route) => route.fulfill({ status: 503, body: "render unavailable" }));
    await page.goto(`${origin}/changes/?tab=summary&type=module&group=status`);
    await expect.poll(() => page.evaluate(() => (window as any).__toudocuFirstFrame)).toEqual({ siteTheme: "paper", colorScheme: "dark", theme: "dark", accent: "violet" });
    await expect(page.locator("[data-file-list]")).toBeVisible();
    await expect(page.locator("body")).toContainText("notes.md");
    await expect(page.locator('[data-file-list] details').filter({ has: page.locator('summary', { hasText: 'docs' }) })).toHaveCount(1);
    await page.locator('[data-file-list] [data-path="docs/notes.md"]').click();
    await expect(page.locator('[data-tab="source"]')).toHaveAttribute("aria-selected", "true");
    await expect(page.locator('[data-tab="source"]')).toHaveText("Изменения");
    await expect(page.locator('[data-tab="file"]')).toHaveText("Файл целиком");
    await expect(page.locator('[data-tab="summary"]')).toHaveCount(0);
    await expect(page).not.toHaveURL(/[?&](?:type|group)=|[?&]tab=summary/);
    await expect(page.locator(".changes-diagnostics")).toHaveAttribute("open", "");
    await expect(page.locator(".changes-diagnostics .diagnostic-severity")).toContainText("Ошибка");
    await expect(page.locator(".workspace-header [data-discussions-toggle]")).toHaveAttribute("aria-controls", "project-discussions-panel");
    await expect(page.locator(".changes-overview [data-discussions-toggle]")).toHaveCount(0);
    await page.locator("[data-scope]").selectOption("documents");
    await expect(page.locator('[data-file-list] [data-path="docs/notes.md"]')).toBeVisible();
    await expect(page.locator('[data-file-list] [data-path="server.go"]')).toHaveCount(0);
    await expect(page).toHaveURL(/scope=documents/);
    await page.locator("[data-scope]").selectOption("other");
    await expect(page.locator('[data-file-list] [data-path="server.go"]')).toBeVisible();
    await expect(page.locator('[data-file-list] [data-path="docs/notes.md"]')).toHaveCount(0);
    await page.locator("[data-scope]").selectOption("");
    const range = page.locator("[data-range-details]");
    const rangeSummary = range.locator("summary");
    await rangeSummary.click();
    await page.keyboard.press("Escape");
    await expect(range).not.toHaveAttribute("open", "");
    await expect(rangeSummary).toBeFocused();
    await rangeSummary.click();
    await page.getByRole("heading", { name: "Изменения", exact: true }).click();
    await expect(range).not.toHaveAttribute("open", "");
    await expect(rangeSummary).toBeFocused();
    await rangeSummary.click();
    await page.locator("[data-apply-range]").click();
    await expect(range).not.toHaveAttribute("open", "");
    await expect(rangeSummary).toBeFocused();
    await page.locator("[data-search]").fill("missing-change");
    await expect(page.locator('[data-ui-state="empty"]')).toContainText("Изменений нет");
    await expect(page).toHaveURL(/q=missing-change/);
    await page.locator("[data-search]").fill("notes.md");
    await page.locator('[data-file-list] [data-path="docs/notes.md"]').click();
    await expect(page.locator('[data-tab="source"]')).toHaveAttribute("aria-selected", "true");
    await expect(page.locator('[data-tab="semantic"]')).toHaveCount(0);
    await page.locator('[data-tab="file"]').click();
    const fileContent = page.locator('[data-file-view] .cm-content');
    await expect(fileContent).toContainText("External edit.");
    await expect(page).toHaveURL(/tab=file/);
    await page.locator('[data-tab="rendered"]').click();
    await expect(page.locator("[data-tab-panel]")).toContainText("Сравнение отрисованных версий недоступно");
    await page.locator('[data-tab="source"]').click();
    await expect(page.locator("[data-source-view]")).toContainText("External edit");
    await page.locator("[data-site-theme-select]").selectOption("terminal");
    await expect(page.locator('[data-tab="source"]')).toHaveAttribute("aria-selected", "true");
    await expect(page.locator("[data-source-view]")).toContainText("External edit");
    await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.locator("[data-mobile-files]").click();
    await expect(page.locator("[data-files-panel]")).toHaveClass(/is-open/);
    await page.keyboard.press("Escape");
    await expect(page.locator("[data-files-panel]")).not.toHaveClass(/is-open/);
    await expect(page.locator("[data-mobile-files]")).toBeFocused();
    await page.locator("[data-discussions-toggle]").click();
    await expect(page.locator("[data-discussions-panel]")).toHaveClass(/is-open/);
    await expect(page.locator("[data-discussions-panel]")).toHaveAttribute("role", "dialog");
    await expect(page.locator("[data-discussions-panel]")).toHaveAttribute("aria-modal", "true");
    await expect(page.locator("[data-discussions-scrim]")).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(page.locator("[data-discussions-toggle]")).toBeFocused();
    await expect(page.locator("[data-discussions-scrim]")).toBeHidden();
    await page.setViewportSize({ width: 1280, height: 720 });
    await page.locator("[data-discussions-toggle]").click();
    await expect(page.locator("[data-discussions-panel]")).toHaveClass(/is-open/);
    await page.locator("[data-discussions-toggle]").click();
    await expect(page.locator("[data-discussions-panel]")).not.toHaveClass(/is-open/);
    await expect(page.locator("[data-discussions-toggle]")).toHaveAttribute("aria-expanded", "false");

    const fallbackContext = await page.context().browser()!.newContext();
    const fallbackPage = await fallbackContext.newPage();
    try {
      await fallbackPage.emulateMedia({ colorScheme: "dark" });
      await fallbackPage.addInitScript(() => {
        localStorage.setItem("toudocu-site-theme", "invalid-theme");
        localStorage.setItem("toudocu-color-scheme", "system");
      });
      await fallbackPage.goto(origin);
      await expect(fallbackPage.locator("html")).toHaveAttribute("data-site-theme", "classic");
      await expect(fallbackPage.locator("html")).toHaveAttribute("data-color-scheme", "system");
      await expect(fallbackPage.locator("html")).toHaveAttribute("data-theme", "dark");
    } finally {
      await fallbackContext.close();
    }

    const privateContext = await page.context().browser()!.newContext();
    const privatePage = await privateContext.newPage();
    try {
      await privatePage.emulateMedia({ colorScheme: "dark" });
      await privatePage.addInitScript(() => {
        Object.defineProperty(window, "localStorage", { configurable: true, get() { throw new Error("storage unavailable"); } });
      });
      await privatePage.goto(`${origin}/_toudocu/editor/`);
      await expect(privatePage.locator("html")).toHaveAttribute("data-site-theme", "classic");
      await expect(privatePage.locator("html")).toHaveAttribute("data-color-scheme", "system");
      await expect(privatePage.locator("html")).toHaveAttribute("data-theme", "dark");
      await expect(privatePage.locator("[data-file-tree]")).toBeVisible();
    } finally {
      await privateContext.close();
    }

    await page.route("**/_toudocu/editor/?disabled=1", async (route) => {
      const response = await route.fetch();
      const body = (await response.text()).replace('"editor":true', '"editor":false');
      await route.fulfill({ response, body });
    });
    await page.goto(`${origin}/_toudocu/editor/?disabled=1`);
    await expect(page.locator('[data-ui-state="capability-unavailable"]')).toContainText("Редактор недоступен");
  } finally {
    await stopChild(child);
  }
});

test("Changes loads only the selected file detail and ignores stale responses", async ({ page }) => {
  const fixture = mkdtempSync(join(tmpdir(), "toudocu-detail-browser-"));
  const output = mkdtempSync(join(tmpdir(), "toudocu-detail-site-"));
  mkdirSync(join(fixture, "docs", "architecture"), { recursive: true });
  mkdirSync(join(fixture, ".toudocu"));
  writeFileSync(join(fixture, ".toudocu", "config.yml"), "documentationVersion: 2\nproject:\n  locale: ru\n");
  writeFileSync(join(fixture, "docs", "index.md"), "# Detail test\n");
  writeFileSync(join(fixture, "docs", "architecture", "overview.md"), "# Architecture\n\n- Тип документа: Architecture Overview\n");
  for (const name of ["a", "b", "c"]) writeFileSync(join(fixture, `${name}.go`), `package detail\n\nconst ${name.toUpperCase()} = "old"\n`);
  run("git", ["init", "-q"], fixture);
  run("git", ["config", "user.email", "detail@example.invalid"], fixture);
  run("git", ["config", "user.name", "Detail Browser"], fixture);
  run("git", ["add", "."], fixture);
  run("git", ["commit", "-qm", "baseline"], fixture);
  for (const name of ["a", "b", "c"]) writeFileSync(join(fixture, `${name}.go`), `package detail\n\nconst ${name.toUpperCase()} = "current-${name}"\n`);
  writeFileSync(join(fixture, "docs", "architecture", "overview.md"), "# Architecture\n\n- Тип документа: Architecture Overview\n\nRendered current.\n");

  const portServer = createServer();
  await new Promise<void>((resolveListen) => portServer.listen(0, "127.0.0.1", resolveListen));
  const address = portServer.address();
  if (!address || typeof address === "string") throw new Error("failed to reserve detail port");
  const port = address.port;
  await new Promise<void>((resolveClose) => portServer.close(() => resolveClose()));
  const child = spawn(testCLI(), ["serve", join(fixture, "docs"), "--repository-root", fixture, "-o", output, "--host", "127.0.0.1", "--port", String(port), "--no-update-check"], { cwd: repo, stdio: "pipe" });
  const origin = `http://127.0.0.1:${port}`;
  const pending: Array<{ release: () => void }> = [];
  let requests = 0;
  let completed = 0;
  let failPath = '';
  let renderRequests = 0;
  let repositoryRequests = 0;
  let repositoryRequestsInFlight = 0;
  let maxRepositoryRequestsInFlight = 0;
  try {
    await waitForHTTP(origin);
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.addInitScript(() => {
      const nativeSetTimeout = window.setTimeout.bind(window);
      window.setTimeout = ((handler: TimerHandler, timeout?: number, ...args: any[]) => nativeSetTimeout(handler, timeout === 2000 ? 50 : timeout, ...args)) as typeof window.setTimeout;
    });
    await page.route("**/_toudocu/api/changes/review/repository/changes?**", async (route) => {
      repositoryRequests++;
      repositoryRequestsInFlight++;
      maxRepositoryRequestsInFlight = Math.max(maxRepositoryRequestsInFlight, repositoryRequestsInFlight);
      if (repositoryRequests > 1) await new Promise((resolve) => setTimeout(resolve, 200));
      await route.continue();
      repositoryRequestsInFlight--;
    });
    await page.route("**/_toudocu/api/changes/render?**", async (route) => {
      renderRequests++;
      await route.continue();
    });
    await page.route("**/_toudocu/api/changes/review/repository/file?**", async (route) => {
      requests++;
      await new Promise<void>((release) => pending.push({ release }));
      const path = new URL(route.request().url()).searchParams.get("path");
      if (path === failPath) {
        failPath = '';
        await route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ diagnostics: [{ message: "detail unavailable" }] }) });
      } else {
        await route.continue();
      }
      completed++;
    });
    await page.goto(`${origin}/changes/`);
    await expect.poll(() => pending.length).toBe(1);
    await expect(page.locator('[data-file-list] [data-path="a.go"]')).toHaveClass(/is-active/);
    await expect(page).toHaveURL(/path=a.go/);
    await expect(page.locator('.changes-detail-header p')).toContainText("a.go");
    await expect(page.locator('[data-detail]')).toHaveAttribute("aria-busy", "true");
    const loading = page.locator('[data-ui-state="loading"]');
    await expect(loading).toHaveText("Загрузка файла…");
    await expect.poll(() => loading.evaluate((element) => getComputedStyle(element, "::before").display)).toBe("none");
    pending[0].release();
    await expect(page.locator('[data-detail]')).not.toHaveAttribute("aria-busy", "true");
    await expect(page.locator("[data-source-view]")).toContainText('current-a');

    const cachedRequests = requests;
    await page.locator('[data-tab="file"]').click();
    await expect(page.locator('[data-file-view]')).toContainText("current-a");
    await page.locator('[data-tab="source"]').click();
    await expect.poll(() => requests).toBe(cachedRequests);

    await page.locator('[data-file-list] [data-path="b.go"]').click();
    await expect.poll(() => pending.length).toBe(2);
    await page.locator('[data-file-list] [data-path="c.go"]').click();
    await expect.poll(() => pending.length).toBe(3);
    pending[2].release();
    await expect(page.locator('[data-detail]')).not.toHaveAttribute("aria-busy", "true");
    await expect(page.locator('.changes-detail-header p')).toContainText("c.go");
    await expect(page.locator("[data-source-view]")).toContainText("current-c");
    pending[1].release();
    await expect.poll(() => completed).toBe(3);
    await expect(page.locator('.changes-detail-header p')).toContainText("c.go");
    await expect(page.locator("[data-source-view]")).toContainText("current-c");

    await page.locator('[data-file-list] [data-path="b.go"]').click();
    await expect.poll(() => pending.length).toBe(4);
    failPath = 'b.go';
    pending[3].release();
    await expect(page.locator('[data-tab-panel] .changes-error')).toContainText("detail unavailable");
    await expect(page.locator('[data-detail]')).not.toHaveAttribute("aria-busy", "true");
    await page.locator('[data-file-list] [data-path="b.go"]').click();
    await expect.poll(() => pending.length).toBe(5);
    pending[4].release();
    await expect(page.locator("[data-source-view]")).toContainText("current-b");

    await page.locator('[data-file-list] [data-path="docs/architecture/overview.md"]').click();
    await expect.poll(() => pending.length).toBe(6);
    pending[5].release();
    await page.locator('[data-tab="rendered"]').click();
    await expect(page.locator(".rendered-columns")).toContainText("Rendered current.");
    await page.locator('[data-tab="source"]').click();
    await page.locator('[data-tab="rendered"]').click();
    await page.locator("[data-site-theme-select]").selectOption("terminal");
    await expect.poll(() => renderRequests).toBe(0);
    await expect.poll(() => repositoryRequests, { timeout: 2_000 }).toBeGreaterThanOrEqual(3);
    expect(maxRepositoryRequestsInFlight).toBe(1);
  } finally {
    for (const request of pending) request.release();
    await stopChild(child);
  }
});

test("Portal and Changes share documentation discussions with the agent CLI", async ({ page, browser }) => {
  test.setTimeout(150_000);
  const fixture = mkdtempSync(join(tmpdir(), "toudocu-agent-feedback-browser-"));
  const stateRoot = mkdtempSync(join(tmpdir(), "toudocu-agent-feedback-state-"));
  mkdirSync(join(fixture, "docs", "architecture"), { recursive: true });
  mkdirSync(join(fixture, ".toudocu"));
  writeFileSync(join(fixture, ".toudocu", "config.yml"), "documentationVersion: 2\nproject:\n  locale: ru\n");
  writeFileSync(join(fixture, "docs", "index.md"), "# Review\n");
  writeFileSync(join(fixture, "docs", "architecture", "overview.md"), "# Architecture\n\n- Тип документа: Architecture Overview\n\n## Boundary\n\n```mermaid\nflowchart LR\n  A[\"Rendered<br>label\"]\n```\n\nUpdated.\n\nRepeated.\n\nRepeated.\n");
  writeFileSync(join(fixture, "server.go"), "package main\n\nfunc oldName() {}\n");
  writeFileSync(join(fixture, "image.bin"), Buffer.from([0, 1]));
  run("git", ["init", "-q"], fixture);
  run("git", ["config", "user.email", "review@example.invalid"], fixture);
  run("git", ["config", "user.name", "Review Browser"], fixture);
  run("git", ["add", "."], fixture);
  run("git", ["commit", "-qm", "baseline"], fixture);
  writeFileSync(join(fixture, "docs", "architecture", "overview.md"), "# Architecture\n\n- Тип документа: Architecture Overview\n\n## Boundary\n\n```mermaid\nflowchart LR\n  A[\"Rendered<br>label\"]\n```\n\nUpdated now.\n\nRepeated.\n\nRepeated.\n\nChanged.\n");
  writeFileSync(join(fixture, "server.go"), "package main\n\nfunc newName() {}\n");
  writeFileSync(join(fixture, "image.bin"), Buffer.from([0, 2]));

  const portServer = createServer();
  await new Promise<void>((resolveListen) => portServer.listen(0, "127.0.0.1", resolveListen));
  const address = portServer.address();
  if (!address || typeof address === "string") throw new Error("failed to reserve review port");
  const port = address.port;
  await new Promise<void>((resolveClose) => portServer.close(() => resolveClose()));
  const environment = { ...process.env, TOUDOCU_STATE_HOME: stateRoot };
  const output = mkdtempSync(join(tmpdir(), "toudocu-agent-feedback-site-"));
  const child = spawn(testCLI(), ["serve", join(fixture, "docs"), "--repository-root", fixture, "-o", output, "--host", "127.0.0.1", "--port", String(port), "--no-update-check"], { cwd: repo, env: environment, stdio: "pipe" });
  const origin = "http://127.0.0.1:" + port;
  const agentCLI = (args: string[]) => {
    const result = spawnSync(testCLI(), ["agent", ...args, "--repository-root", fixture, "--json"], { cwd: repo, env: environment, encoding: "utf8" });
    if (result.status !== 0) throw new Error("agent CLI failed\n" + result.stdout + "\n" + result.stderr);
    return JSON.parse(result.stdout);
  };

  try {
    await waitForHTTP(origin);
    await page.context().grantPermissions(["clipboard-read", "clipboard-write"], { origin });
    await page.goto(origin);
    await expect(page.locator("[data-td-island-instance='agent-console']")).toHaveAttribute("data-td-island-state", "mounted");
    const homeToggle = page.locator(".site-header [data-discussions-toggle]");
    await expect(homeToggle).toBeVisible();
    await expect(homeToggle).toHaveAttribute("aria-expanded", "false");
    await expect(homeToggle).toHaveAttribute("aria-controls", "project-discussions-panel");
    const initialIslandChunks = await page.evaluate(() => performance.getEntriesByType("resource").filter((entry) => /\/chunks\/island-[^/]+\.js$/.test(entry.name)).length);
    expect(initialIslandChunks).toBeGreaterThan(0);
    await homeToggle.click();
    await expect.poll(() => page.evaluate(() => performance.getEntriesByType("resource").filter((entry) => /\/chunks\/island-[^/]+\.js$/.test(entry.name)).length)).toBeGreaterThan(initialIslandChunks);
    await expect(page.locator(".portal-review-panel")).toBeVisible();
    await expect(page.locator("[data-portal-review-new]")).toBeHidden();
    await homeToggle.click();
    await expect(page.locator(".portal-review-panel")).toBeHidden();
    await expect(homeToggle).toHaveAttribute("aria-expanded", "false");
    await homeToggle.click();
    await page.keyboard.press("Escape");
    await expect(homeToggle).toBeFocused();
    await page.goto(origin + "/architecture/overview.html");
    await expect(page.locator(".site-header [data-discussions-toggle]")).toBeVisible();
    await expect(page.locator(".document-context-actions [data-discussions-toggle]")).toHaveCount(0);

    const selectPortalElement = (element: any) => element.evaluate((target: HTMLElement) => {
      const range = document.createRange();
      range.selectNodeContents(target);
      const selection = window.getSelection();
      selection?.removeAllRanges();
      selection?.addRange(range);
      target.dispatchEvent(new PointerEvent("pointerup", { bubbles: true }));
    });
    const selectionMenu = page.locator(".review-selection-menu");
    await selectPortalElement(page.locator(".page-header h1"));
    await expect(selectionMenu).toBeVisible();
    await selectionMenu.getByRole("button", { name: "Копировать текст" }).click();
    await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe("Architecture");

    await selectPortalElement(page.locator(".page-header h1"));
    await selectionMenu.getByRole("button", { name: "Добавить вопрос" }).click();
    const composer = page.locator(".portal-review-dialog:not(.portal-review-confirm)");
    await expect(composer).toBeVisible();
    await composer.getByRole("button", { name: "Отмена" }).click();
    await expect(composer).not.toBeVisible();

    await selectPortalElement(page.locator(".page-header h1"));
    await selectionMenu.getByRole("button", { name: "Добавить вопрос" }).click();
    await composer.locator("[data-portal-review-question]").fill("Почему документ так называется?");
    await composer.locator(".portal-review-submit").click();
    const panel = page.locator(".portal-review-panel");
    let titleThread = panel.locator(".portal-review-thread").filter({ hasText: "Почему документ так называется?" });
    await expect(titleThread.locator("[data-portal-message-edit]")).toBeVisible();
    await titleThread.locator("[data-portal-message-edit]").click();
    await composer.locator("[data-portal-review-question]").fill("Почему у документа такое название?");
    await composer.locator(".portal-review-submit").click();
    titleThread = panel.locator(".portal-review-thread").filter({ hasText: "Почему у документа такое название?" });
    await titleThread.locator("[data-portal-message-delete]").click();
    await expect(titleThread).toHaveCount(0);
    await panel.getByRole("button", { name: "Закрыть панель" }).click();

    await selectPortalElement(page.getByText("Repeated.", { exact: true }).last());
    await selectionMenu.getByRole("button", { name: "Добавить вопрос" }).click();
    await expect(composer.locator("[data-portal-review-selection]")).toHaveText("Repeated.");
    await composer.locator("[data-portal-review-question]").fill("Почему фрагмент повторяется?");
    await composer.locator(".portal-review-submit").click();

    await expect(panel).toBeVisible();
    const thread = panel.locator(".portal-review-thread").filter({ hasText: "Почему фрагмент повторяется?" });
    await expect(thread.locator("[data-portal-message-edit]")).toBeVisible();
    await expect(thread.locator(".portal-review-quote")).toHaveText("Repeated.");

    await panel.locator("[data-portal-review-copy-prompt]").click();
    await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe("$toudocu feedback");

    const request = agentCLI(["next"]);
    expect(request.pending).toBe(true);
    expect(request.discussion.messages[0].intent).toBe("question");
    expect(request.target.selectedText).toBe("Repeated.");
    expect(request.target.range.start.line).toBe(16);
    await expect(thread.locator("[data-portal-message-edit]")).toHaveCount(0, { timeout: 5_000 });
    const responsePath = join(mkdtempSync(join(tmpdir(), "toudocu-agent-response-")), "response.json");
    writeFileSync(responsePath, JSON.stringify({
      schemaVersion: 1,
      deliveryId: request.deliveryId,
      discussionId: request.discussion.id,
      outcome: "answered",
      message: "Повтор нужен для сравнения двух сценариев.",
      evidence: [],
      changedPaths: [],
    }));
    const accepted = agentCLI(["respond", "--input", responsePath]);
    expect(accepted.accepted).toBe(true);
    await expect(panel).toContainText("Повтор нужен для сравнения двух сценариев.", { timeout: 5_000 });

    const mermaidNode = page.locator("[data-mermaid-diagram] svg .node").filter({ hasText: "Renderedlabel" });
    await selectPortalElement(mermaidNode);
    await selectionMenu.getByRole("button", { name: "Добавить вопрос" }).click();
    const renderedSelection = await composer.locator("[data-portal-review-selection]").textContent();
    expect(renderedSelection).toBe("Rendered\nlabel");
    await composer.locator("[data-portal-review-question]").fill("Что означает подпись диаграммы?");
    await composer.locator(".portal-review-submit").click();
    const mermaidThread = panel.locator(".portal-review-thread").filter({ hasText: "Что означает подпись диаграммы?" });
    await expect(mermaidThread.locator(".portal-review-quote")).toHaveText("Rendered\nlabel");
    const mermaidRequest = agentCLI(["next"]);
    expect(mermaidRequest.target.path).toBe("docs/architecture/overview.md");
    expect(mermaidRequest.target.selectedText).toBe("Rendered\nlabel");
    expect(mermaidRequest.target.range).toBeUndefined();
    writeFileSync(responsePath, JSON.stringify({
      schemaVersion: 1,
      deliveryId: mermaidRequest.deliveryId,
      discussionId: mermaidRequest.discussion.id,
      outcome: "answered",
      message: "Это подпись шага диаграммы.",
      evidence: [],
      changedPaths: [],
    }));
    expect(agentCLI(["respond", "--input", responsePath]).accepted).toBe(true);
    await expect(mermaidThread).toContainText("Это подпись шага диаграммы.", { timeout: 5_000 });
    await mermaidThread.getByRole("button", { name: "Закрыть", exact: true }).click();
    await thread.getByRole("button", { name: "Закрыть", exact: true }).click();
    await expect(thread).toHaveClass(/is-resolved/);
    await thread.getByRole("button", { name: "Открыть снова" }).click();

    await panel.getByRole("button", { name: "Закрыть панель" }).click();
    await page.goto(origin);
    await expect(page.locator("[data-open-discussion-count]")).toHaveText("1");
    await page.locator(".site-header [data-discussions-toggle]").click();
    await expect(page.locator(".portal-review-thread").filter({ hasText: "Почему фрагмент повторяется?" })).toContainText("Почему фрагмент повторяется?");
    await expect(page.locator(".portal-review-thread").filter({ hasText: "Почему фрагмент повторяется?" }).locator(".portal-review-quote")).toHaveText("Repeated.");
    await expect(page.locator("[data-portal-review-new]")).toBeHidden();
    await page.keyboard.press("Escape");
    await expect(page.locator(".site-header [data-discussions-toggle]")).toBeFocused();
    await page.locator('[data-workspace="changes"]').click();
    const changesComposer = page.locator("[data-review-composer]");
    await page.locator('[data-file-list] [data-path="docs/architecture/overview.md"]').click();
    await expect(page.locator(".workspace-header [data-discussions-toggle]")).toHaveAttribute("aria-controls", "project-discussions-panel");
    await page.locator('[data-tab="file"]').click();
    await page.locator("[data-discussions-toggle]").click();
    await expect(page.locator(".review-thread").filter({ hasText: "Почему фрагмент повторяется?" }).locator(".portal-review-quote")).toHaveText("Repeated.");
    await page.locator("[data-discussions-close]").click();
    const fullFileLine = page.locator('[data-file-view] .cm-line').filter({ hasText: "Updated now." });
    await fullFileLine.selectText();
    await expect.poll(() => page.evaluate(() => window.getSelection()?.toString())).toBe("Updated now.");
    const fullFileMenu = page.locator("[data-tab-panel] .review-selection-menu");
    await expect(fullFileMenu).toBeVisible();
    await expect(fullFileMenu.locator("[data-selection-copy]")).toBeAttached();
    await expect(fullFileMenu.locator("[data-selection-context]")).toBeAttached();
    await expect(page.locator("[data-td-island-instance='agent-console']")).toHaveCount(1);
    await fullFileMenu.locator("[data-selection-question]").click();
    await expect(page.locator("#agent-console-message")).toHaveValue(/Updated now/);
    await page.locator("[data-agent-console-close]").click();
    await page.locator('[data-tab="source"]').click();
    const removedContent = page.locator(".diff-line-removed .diff-line-content").filter({ hasText: "Updated." });
    const addedContent = page.locator(".diff-line-added .diff-line-content").filter({ hasText: "Updated now." });
    await addedContent.evaluate((element) => {
      const range = document.createRange();
      range.selectNodeContents(element);
      const selection = window.getSelection();
      selection?.removeAllRanges();
      selection?.addRange(range);
      element.dispatchEvent(new PointerEvent("pointerup", { bubbles: true }));
    });
    const diffMenu = page.locator("[data-tab-panel] .review-selection-menu");
    await expect(diffMenu).toBeVisible();
    await expect.poll(() => page.evaluate(() => window.getSelection()?.toString())).toBe("Updated now.");
    await expect(diffMenu.getByRole("button", { name: "Копировать текст" })).toBeEnabled();
    await expect(diffMenu.getByRole("button", { name: "Копировать контекст" })).toBeEnabled();
    await expect(diffMenu.getByRole("button", { name: "Добавить вопрос" })).toBeEnabled();
    await diffMenu.getByRole("button", { name: "Копировать текст" }).click();
    await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe("Updated now.");
    await addedContent.dispatchEvent("pointerup");
    await diffMenu.getByRole("button", { name: "Добавить вопрос" }).click();
    await expect(changesComposer.locator("[data-review-target-summary]")).toContainText(/docs\/architecture\/overview\.md · \d+:1–\d+:\d+/);
    await changesComposer.getByRole("button", { name: "Отмена" }).click();

    await page.evaluate(() => {
      const start = [...document.querySelectorAll<HTMLElement>(".diff-line-removed .diff-line-content")].find((element) => element.textContent?.includes("Updated."));
      const end = [...document.querySelectorAll<HTMLElement>(".diff-line-added .diff-line-content")].find((element) => element.textContent?.includes("Updated now."));
      if (!start?.firstChild || !end?.firstChild) throw new Error("diff rows not found");
      const range = document.createRange();
      range.setStart(start.firstChild, 0);
      range.setEnd(end.firstChild, end.textContent?.length ?? 0);
      const selection = window.getSelection();
      selection?.removeAllRanges();
      selection?.addRange(range);
      end.dispatchEvent(new PointerEvent("pointerup", { bubbles: true }));
    });
    await expect(diffMenu.getByRole("button", { name: "Добавить вопрос" })).toBeDisabled();
    await diffMenu.getByRole("button", { name: "Копировать контекст" }).click();
    await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toContain("Updated.\nUpdated now.");

    const oldPlus = removedContent.locator("..").locator(".diff-comment");
    await oldPlus.hover();
    await expect(oldPlus).toHaveCSS("opacity", "1");
    await oldPlus.focus();
    await expect(oldPlus).toBeFocused();
    await oldPlus.click();
    await expect(changesComposer.locator("[data-review-target-summary]")).toHaveText("docs/architecture/overview.md");
    await changesComposer.locator("[data-review-message]").fill("Сохрани старый фрагмент.");
    await changesComposer.locator('button[type="submit"]').click();
    const quotedThread = page.locator(".review-thread").filter({ hasText: "Сохрани старый фрагмент." });
    await expect(quotedThread).toContainText(/> docs\/architecture\/overview\.md:\d+/);
    await expect(quotedThread).toContainText("> Updated.");
    await quotedThread.locator("[data-delete-discussion]").click();
    await page.locator("[data-review-delete-confirm]").getByRole("button", { name: "Удалить" }).click();
    await expect(quotedThread).toHaveCount(0);
    await page.locator("[data-discussions-close]").click();

    await page.locator("[data-detail] [data-file-comment]").click();
    await changesComposer.locator("[data-review-intent]").selectOption("change_request");
    await changesComposer.locator("[data-review-message]").fill("Проверь и уточни этот документ.");
    await changesComposer.locator('button[type="submit"]').click();
    const changesThread = page.locator(".review-thread").filter({ hasText: "Проверь и уточни этот документ." });
    await expect(changesThread.locator(".portal-review-quote")).toHaveCount(0);
    await expect(changesThread.locator("[data-edit-message]")).toBeVisible();
    const second = agentCLI(["next"]);
    expect(second.pending).toBe(true);
    expect(second.target.kind).toBe("file");
    expect(second.discussion.messages.at(-1).intent).toBe("change_request");
    await expect(changesThread.locator("[data-edit-message]")).toHaveCount(0, { timeout: 5_000 });

    await page.locator("[data-send-feedback]").click();
    await expect(page.locator("#agent-console-message")).toHaveValue("$toudocu feedback");
    await page.locator("[data-agent-console-close]").click();
    await changesThread.locator("[data-delete-discussion]").click();
    await page.locator("[data-review-delete-confirm]").getByRole("button", { name: "Удалить" }).click();
    await expect(changesThread).toHaveCount(0);
    expect(agentCLI(["next"]).pending).toBe(false);
    await page.locator("[data-discussions-close]").click();

    await page.locator('[data-file-list] [data-path="server.go"]').click();
    await expect(page.locator("[data-detail] h2")).toHaveText("server.go");
    await expect(page.locator("[data-detail]")).not.toHaveAttribute("aria-busy", "true");
    await page.locator("[data-detail] [data-file-comment]").click();
    await changesComposer.locator("[data-review-message]").fill("Почему переименована функция?");
    await changesComposer.locator('button[type="submit"]').click();
    await expect(page.locator(".review-thread").filter({ hasText: "Почему переименована функция?" })).toBeVisible();
    const codeQuestion = agentCLI(["next"]);
    expect(codeQuestion.target.kind).toBe("file");
    expect(codeQuestion.target.path).toBe("server.go");
    const codeResponse = join(mkdtempSync(join(tmpdir(), "toudocu-agent-response-")), "response.json");
    writeFileSync(codeResponse, JSON.stringify({ schemaVersion: 1, deliveryId: codeQuestion.deliveryId, discussionId: codeQuestion.discussion.id, outcome: "answered", message: "Имя отражает новое поведение.", evidence: [{ path: "server.go" }], changedPaths: [] }));
    agentCLI(["respond", "--input", codeResponse]);
    await page.locator("[data-discussions-close]").click();

    await page.locator('[data-file-list] [data-path="image.bin"]').click();
    await expect(page.locator("[data-detail] [data-file-comment]")).toBeVisible();
    await expect(page.locator("[data-tab-panel] .changes-error")).toBeVisible();

    await page.goto(origin + "/changes/?target=HEAD");
    await expect(page.locator("[data-discussions-toggle]")).toBeHidden();
    await expect(page.locator("[data-file-comment]")).toHaveCount(0);

    const touchContext = await browser.newContext({ hasTouch: true, viewport: { width: 390, height: 844 } });
    const touchPage = await touchContext.newPage();
    try {
      await touchPage.goto(origin + "/changes/?path=docs%2Farchitecture%2Foverview.md");
      const touchPlus = touchPage.locator(".diff-line-added .diff-comment").first();
      await expect(touchPlus).toBeVisible();
      await expect(touchPlus).toHaveCSS("opacity", "1");
      await touchPage.locator("[data-discussions-toggle]").click();
      const mobilePanel = touchPage.locator("[data-discussions-panel]");
      await expect(mobilePanel).toHaveAttribute("role", "dialog");
      await expect(mobilePanel).toHaveAttribute("aria-modal", "true");
      await expect.poll(async () => {
        const box = await mobilePanel.boundingBox();
        return box && { x: box.x, y: box.y, width: box.width, height: box.height };
      }).toEqual({ x: 0, y: 0, width: 390, height: 844 });
    } finally {
      await touchContext.close();
    }
  } finally {
    await stopChild(child);
  }
});
