import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtemp, readFile, readdir, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { test } from "node:test";

const generated = new URL("../../internal/site/assets/generated/", import.meta.url);
const repo = resolve(fileURLToPath(new URL("../..", import.meta.url)));

test("generated project portal stays static and read-only", async (context) => {
  const temporary = await mkdtemp(join(tmpdir(), "toudocu-contract-"));
  context.after(() => rm(temporary, { recursive: true, force: true }));
  const output = join(temporary, "project-docs");
  const result = spawnSync("go", [
    "run", "./cmd/toudocu", "build", "./docs",
    "--output", output,
    "--repository-root", ".",
    "--clean",
    "--strict",
    "--stale-days", "0",
  ], { cwd: repo, encoding: "utf8" });
  assert.equal(result.status, 0, `portal build failed\n${result.stdout}\n${result.stderr}`);
  const portal = pathToFileURL(`${output}/`);
  const pages = ["index.html", "reference/configuration.html"];
  for (const page of pages) {
    const source = await readFile(new URL(page, portal), "utf8");
    assert.equal(source.includes('"runtime":"static"'), true, `${page} is not a static runtime`);
    assert.equal(source.includes('"runtime":"serve"'), false, `${page} leaked serve runtime`);
    assert.equal(source.includes("data-server-rebuild"), false, `${page} leaked rebuild control`);
    assert.equal(source.includes('href="/_toudocu/editor/'), false, `${page} leaked editor action`);
    assert.equal(source.includes('href="/changes/'), false, `${page} leaked changes action`);
  }

  const assets = new Set(await readdir(new URL("assets/", portal)));
  for (const forbidden of ["serve.js", "serve.css", "editor.js", "editor.css", "changes.js", "changes.css", "codemirror.js", "api-docs.js"]) {
    assert.equal(assets.has(forbidden), false, `${forbidden} leaked into static portal`);
  }
});

test("manifest separates static and serve assets", async () => {
  const manifest = JSON.parse(await readFile(new URL("manifest.json", generated), "utf8"));
  const viteManifest = JSON.parse(await readFile(new URL("vite-manifest.json", generated), "utf8"));
  const licenses = JSON.parse(await readFile(new URL("licenses.json", generated), "utf8"));
  assert.equal(manifest.schemaVersion, 1);
  assert.ok(Object.values(viteManifest).some((entry) => entry.name === "portal" && entry.isEntry));
  assert.ok(licenses.some((license) => license.name === "codemirror"));
  assert.ok(licenses.some((license) => license.name === "@xterm/xterm"));
  assert.ok(manifest.runtimes.static.includes("appearance.js"));
  assert.ok(manifest.runtimes.serve.includes("appearance.js"));
  assert.ok(manifest.runtimes.static.includes("portal.js"));
  assert.ok(manifest.runtimes.serve.includes("editor.js"));
  assert.ok(manifest.runtimes.serve.includes("changes.js"));
  for (const forbidden of ["editor.js", "changes.js", "serve.js", "codemirror.js", "api-docs.js"]) {
    assert.equal(manifest.runtimes.static.includes(forbidden), false, `${forbidden} leaked into static runtime`);
  }
  const terminalAssets = manifest.runtimes.serve.filter((file) => /terminal/i.test(file));
  assert.ok(terminalAssets.some((file) => file.endsWith(".js")), "serve runtime misses lazy terminal chunk");
  assert.ok(terminalAssets.some((file) => file.endsWith(".css")), "serve runtime misses terminal CSS");
  assert.equal(manifest.runtimes.static.some((file) => /terminal/i.test(file)), false, "terminal assets leaked into static runtime");
  for (const file of manifest.runtimes.static) {
    if (!file.endsWith(".js")) continue;
    const source = await readFile(new URL(file, generated), "utf8");
    assert.equal(source.includes("/assets/"), false, `${file} contains an absolute asset URL`);
  }
});

test("portal bundle has no server-only endpoint", async () => {
  const staticBundles = ["portal.js", "screen-map.js", "playable-flow.js"];
  for (const name of staticBundles) {
    const source = await readFile(new URL(name, generated), "utf8");
    for (const forbidden of ["/_toudocu/api/editor", "/_toudocu/api/changes", "/_toudocu/api/version", "/__toudocu/rebuild", "localhost"]) {
      assert.equal(source.includes(forbidden), false, `${forbidden} leaked into ${name}`);
    }
  }
  const portal = await readFile(new URL("portal.js", generated), "utf8");
  assert.equal(portal.includes("search-index.json"), true);
});

test("bootstrap source uses stable page kinds and explicit failure states", async () => {
  const source = await readFile(new URL("../src/core/bootstrap.ts", import.meta.url), "utf8");
  for (const kind of ["document", "architecture", "module", "use-case", "flow", "screen", "standard", "runbook", "task"]) {
    assert.equal(source.includes(`\"${kind}\"`), true, `missing stable kind ${kind}`);
  }
  for (const state of ["bootstrap unavailable", "unsupported schema", "invalid bootstrap"]) {
    assert.equal(source.includes(state), true, `missing state ${state}`);
  }
  assert.equal(source.includes("querySelector(\"h1\")"), false);
});

test("React UI layers keep their dependency boundary", async () => {
  const ui = await readFile(new URL("../src/ui/index.tsx", import.meta.url), "utf8");
  const docsUI = await readFile(new URL("../src/docs-ui/index.tsx", import.meta.url), "utf8");
  for (const component of ["Button", "IconButton", "Badge", "Separator", "Spinner", "EmptyState", "Diagnostic", "Dialog", "Tabs", "Tooltip", "Popover", "Menu", "Select"]) {
    assert.equal(ui.includes(component), true, `missing React component ${component}`);
  }
  async function layerSources(directory) {
    const sources = [];
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const target = new URL(`${entry.name}${entry.isDirectory() ? "/" : ""}`, directory);
      if (entry.isDirectory()) sources.push(...await layerSources(target));
      else if (/\.tsx?$/.test(entry.name)) sources.push([target, await readFile(target, "utf8")]);
    }
    return sources;
  }
  const sources = await layerSources(new URL("../src/ui/", import.meta.url));
  sources.push(...await layerSources(new URL("../src/docs-ui/", import.meta.url)));
  for (const [file, source] of sources) {
    for (const forbidden of ["PageBootstrap", "ToudocuPage", "fetch", "XMLHttpRequest", "/_toudocu/", "/core/", "../core", "locale", "text(\""]) {
      assert.equal(source.includes(forbidden), false, `${file.pathname} contains ${forbidden}`);
    }
  }
  assert.equal(docsUI.includes("translate: Translator"), true, "docs-ui has no injected translator contract");
});

test("legacy UI foundation and compatibility bridges stay removed", async () => {
  const portal = await readFile(new URL("../src/core/portal.ts", import.meta.url), "utf8");
  const editor = await readFile(new URL("../src/features/editor/app.tsx", import.meta.url), "utf8");
  const changes = await readFile(new URL("../src/styles/changes.css", import.meta.url), "utf8");
  const gallery = await readFile(new URL("../src/entries/dev-ui.tsx", import.meta.url), "utf8");
  const site = await readFile(new URL("../../internal/app/site.go", import.meta.url), "utf8");
  const taskSite = await readFile(new URL("../../internal/app/task_site.go", import.meta.url), "utf8");
  const screenSite = await readFile(new URL("../../internal/app/screen_site.go", import.meta.url), "utf8");
  const workspaceShell = await readFile(new URL("../../internal/app/workspace_shell.go", import.meta.url), "utf8");
  const docsCore = await readFile(new URL("../../internal/app/docs_core.go", import.meta.url), "utf8");
  const localeEN = JSON.parse(await readFile(new URL("../../internal/site/i18n/en.json", import.meta.url), "utf8"));
  const localeRU = JSON.parse(await readFile(new URL("../../internal/site/i18n/ru.json", import.meta.url), "utf8"));
  const portalCSS = await readFile(new URL("../src/styles/portal.css", import.meta.url), "utf8");
  const tokens = await readFile(new URL("../src/styles/tokens.css", import.meta.url), "utf8");
  for (const marker of ["initializeDocumentReview", "createDiscussionPanel", "createDialog", "createTabs", "installTooltip", "createCommandMenu", 'from "../components"']) assert.equal(portal.includes(marker), false, `legacy UI bridge remains: ${marker}`);
  for (const token of ["--bg:", "--surface:", "--text:", "--border:", "--accent:", "--radius:"]) assert.equal(tokens.includes(token), false, `legacy token alias remains: ${token}`);
  for (const icon of ["▾", "☰", "⌄", "✓"]) assert.equal(`${portal}\n${editor}\n${changes}\n${gallery}`.includes(icon), false, `Unicode UI icon remains: ${icon}`);
  for (const icon of ["⌂", "◐", "→", "✦", "✎", "↻", "◎", "▦", "◇", "⇄", "◆", "⇢", "⌗", "▣", "◫", "☐", "☑", "✓", "•", "▾", "☰", "⎙", "↗", "↙", "←", "›", "↑", "↓", "○", "◷", "Ⅱ", "×", "↪", "⌁", "≈"]) assert.equal(`${site}\n${taskSite}\n${screenSite}\n${workspaceShell}\n${docsCore}\n${portalCSS}`.includes(icon), false, `Go/CSS Unicode UI icon remains: ${icon}`);
  for (const key of ["play.openDiagnostics", "screen.openCatalog", "screen.openInCatalog", "screenMap.openDocument"]) for (const locale of [localeEN, localeRU]) assert.equal(locale[key].includes("→"), false, `localized Unicode UI icon remains: ${key}`);
});

test("React UI dependencies are pinned with license metadata", async () => {
  const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
  const lock = JSON.parse(await readFile(new URL("../package-lock.json", import.meta.url), "utf8"));
  for (const [group, names] of Object.entries({
    dependencies: ["react", "react-dom", "@base-ui/react"],
    devDependencies: ["vitest", "@testing-library/react", "@testing-library/user-event"],
  })) {
    for (const name of names) {
      assert.match(manifest[group][name], /^\d+\.\d+\.\d+$/, `${name} is not pinned`);
      assert.ok(lock.packages[`node_modules/${name}`].license, `${name} has no license metadata`);
    }
  }
});

test("semantic tokens and icons expose one accessibility contract", async () => {
  const tokens = await readFile(new URL("../src/styles/tokens.css", import.meta.url), "utf8");
  for (const token of ["--td-surface", "--td-text", "--td-border", "--td-accent", "--td-status-success", "--td-focus", "--td-selection", "--td-space-1", "--td-radius-control", "--td-control-height", "--td-motion-normal"]) {
    assert.equal(tokens.includes(token), true, `missing semantic token ${token}`);
  }
  for (const variant of ['data-site-theme="paper"', 'data-site-theme="terminal"', 'data-density="compact"']) {
    assert.equal(tokens.includes(variant), true, `missing token variant ${variant}`);
  }
  const iconSource = await readFile(new URL("../src/design/icons.ts", import.meta.url), "utf8");
  for (const contract of ['role\", \"img', "aria-label", "aria-hidden"]) {
    assert.equal(iconSource.includes(contract), true, `missing icon contract ${contract}`);
  }
  const componentCSS = await readFile(new URL("../src/styles/components.css", import.meta.url), "utf8");
  assert.equal(componentCSS.includes("stroke-width: 2"), true, "icons do not share Lucide stroke width");
  const portalCSS = await readFile(new URL("../src/styles/portal.css", import.meta.url), "utf8");
  const editorCSS = await readFile(new URL("../src/styles/editor.css", import.meta.url), "utf8");
  assert.equal(portalCSS.includes("--td-surface-canvas:"), true, "themes do not set semantic tokens");
  assert.equal(editorCSS.includes("--editor-"), false, "Editor keeps a feature-specific palette");
});

test("strict TypeScript has no file-level bypass", async () => {
  const root = new URL("../src/", import.meta.url);
  async function sources(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    const files = [];
    for (const entry of entries) {
      const target = new URL(`${entry.name}${entry.isDirectory() ? "/" : ""}`, directory);
      if (entry.isDirectory()) files.push(...await sources(target));
      else if (entry.name.endsWith(".ts")) files.push(target);
    }
    return files;
  }
  for (const file of await sources(root)) {
    const source = await readFile(file, "utf8");
    assert.equal(source.includes("@ts-nocheck"), false, `${file.pathname} bypasses strict checking`);
  }
});

test("serve navigation replaces the versioned bootstrap", async () => {
  const source = await readFile(new URL("../src/core/serve-navigation.ts", import.meta.url), "utf8");
  for (const required of ["validatedBootstrap", "syncBootstrap", "parseBootstrap", "window.ToudocuPage = nextBootstrap.value", "toudocu:pagebeforechange", 'islandHost.unmountAll(["agent-console"])', "islandHost.discover()"]) {
    assert.equal(source.includes(required), true, `serve navigation misses ${required}`);
  }
  assert.ok(source.indexOf("validatedBootstrap(nextDocument)") < source.indexOf("toudocu:pagebeforechange"), "target bootstrap is validated after commit begins");
  assert.ok(source.indexOf("toudocu:pagebeforechange") < source.indexOf("currentLayout.replaceWith"), "layout changes before pagebeforechange");
});

test("serve refreshes portal content without reloading the agent console", async () => {
  const runtime = await readFile(new URL("../src/core/serve-runtime.ts", import.meta.url), "utf8");
  const navigation = await readFile(new URL("../src/core/serve-navigation.ts", import.meta.url), "utf8");
  assert.equal(runtime.includes("new CustomEvent('toudocu:contentrefresh')"), true);
  assert.equal(runtime.includes("window.location.reload()"), false);
  assert.ok(runtime.indexOf("etag = next") < runtime.indexOf("refreshContent();"));
  assert.equal(navigation.includes("document.addEventListener('toudocu:contentrefresh'"), true);
  assert.equal(navigation.includes('islandHost.unmountAll(["agent-console"])'), true);
});

test("Changes keeps optional islands out of its workspace layout", async () => {
  const serve = await readFile(new URL("../src/entries/serve.ts", import.meta.url), "utf8");
  const styles = await readFile(new URL("../src/styles/changes.css", import.meta.url), "utf8");
  assert.equal(serve.includes("document.querySelector(\"[data-td-island-instance='project-discussions']\") && sessionStorage"), true);
  assert.equal(styles.includes('.changes-body > [data-td-island="agent-console"] { display: contents; }'), true);
});

test("Discussions uses the Agent Console panel lifecycle", async () => {
  const source = await readFile(new URL("../src/features/discussions/components.tsx", import.meta.url), "utf8");
  const agent = await readFile(new URL("../src/features/agent-console/island.tsx", import.meta.url), "utf8");
  const styles = await readFile(new URL("../src/styles/serve.css", import.meta.url), "utf8");
  assert.equal(source.includes("const [visible, setVisible] = useState(initiallyOpen)"), true, "Discussions has no closing lifecycle");
  assert.equal(source.includes("window.setTimeout(() => setVisible(false), 180)"), true, "Discussions disappears before its exit animation");
  assert.equal(source.includes("hidden={!visible}"), true, "Discussions still hides directly from open state");
  assert.equal(source.includes('new CustomEvent("toudocu:discussions-open")'), true, "Discussions does not replace Agent Console");
  assert.equal(source.includes('"toudocu:agent-console-open"'), true, "Discussions stays open under Agent Console");
  assert.equal(agent.includes('"toudocu:discussions-open"'), true, "Agent Console stays open over Discussions");
  assert.equal(source.includes('hidden={!visible || !narrow}'), true, "Discussions backdrop is shown on wide screens");
  assert.equal(source.includes('<button\n        type="button"\n        className="portal-review-scrim"'), false, "Discussion backdrop still closes the panel");
  assert.equal(source.includes("element.inert = open && narrow"), true, "Narrow Discussions leaves background content accessible");
  assert.equal(styles.includes("body.discussions-open"), true, "Wide Discussions overlays instead of reserving content space");
  for (const value of ["opacity var(--td-motion-normal) var(--td-motion-easing)", "transform var(--td-motion-normal) var(--td-motion-easing)"]) assert.equal(styles.includes(value), true, `Discussions misses shared motion: ${value}`);
});

test("changes review requests preserve the selected Git range", async () => {
  const app = await readFile(new URL("../src/features/changes/app.tsx", import.meta.url), "utf8");
  const discussions = await readFile(new URL("../src/features/discussions/components.tsx", import.meta.url), "utf8");
  assert.equal(app.includes("requestURL={requestURL}"), true, "Changes does not pass its Git-range URL resolver to Discussions");
  assert.equal(app.includes('for (const key of ["base", "branchBase", "target"])'), true, "Changes discussion resolver drops Git range fields");
  assert.equal(discussions.includes("useDiscussionState(endpoint, signal, requestURL)"), true, "shared Discussions API ignores the supplied URL resolver");
});

test("browser behavior reads user-facing copy from the locale catalog", async () => {
  const root = new URL("../src/", import.meta.url);
  const russian = JSON.parse(await readFile(new URL("../../internal/site/i18n/ru.json", import.meta.url), "utf8"));
  const english = JSON.parse(await readFile(new URL("../../internal/site/i18n/en.json", import.meta.url), "utf8"));
  const defined = new Set(Object.keys(english));
  assert.deepEqual(Object.keys(english).sort(), Object.keys(russian).sort());
  assert.equal(/[А-Яа-яЁё]/.test(JSON.stringify(english)), false, "English catalog contains Russian copy");
  for (const [key, value] of Object.entries(russian)) {
    const markers = (copy) => [...copy.matchAll(/\{\d+\}/g)].map((match) => match[0]).sort();
    assert.deepEqual(markers(english[key]), markers(value), `${key} placeholders differ`);
    assert.equal(/<\/?[a-z][^>]*>/i.test(english[key]), false, `${key} English value contains HTML`);
    assert.equal(/<\/?[a-z][^>]*>/i.test(value), false, `${key} Russian value contains HTML`);
  }
  async function sources(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    const files = [];
    for (const entry of entries) {
      const target = new URL(`${entry.name}${entry.isDirectory() ? "/" : ""}`, directory);
      if (entry.isDirectory()) files.push(...await sources(target));
      else if (entry.name.endsWith(".ts") && entry.name !== "locale.ts") files.push(target);
    }
    return files;
  }
  for (const file of await sources(root)) {
    const source = await readFile(file, "utf8");
    assert.equal(/[А-Яа-яЁё]/.test(source), false, `${file.pathname} contains copy outside the locale catalog`);
    for (const match of source.matchAll(/text\("([^"]+)"/g)) {
      assert.equal(defined.has(match[1]), true, `${file.pathname} uses missing locale key ${match[1]}`);
    }
  }
  const localeSource = await readFile(new URL("../src/core/locale.ts", import.meta.url), "utf8");
  assert.equal(localeSource.includes("registerMessages"), false);
});
