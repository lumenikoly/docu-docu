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
  assert.ok(manifest.runtimes.static.includes("appearance.js"));
  assert.ok(manifest.runtimes.serve.includes("appearance.js"));
  assert.ok(manifest.runtimes.static.includes("portal.js"));
  assert.ok(manifest.runtimes.serve.includes("editor.js"));
  assert.ok(manifest.runtimes.serve.includes("changes.js"));
  for (const forbidden of ["editor.js", "changes.js", "serve.js", "codemirror.js", "api-docs.js"]) {
    assert.equal(manifest.runtimes.static.includes(forbidden), false, `${forbidden} leaked into static runtime`);
  }
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

test("design primitives are model-independent", async () => {
  const source = await readFile(new URL("../src/components/index.ts", import.meta.url), "utf8");
  for (const component of ["createButton", "createIconButton", "createBadge", "createTabs", "wireDisclosure", "createDialog", "installTooltip", "createCommandMenu", "createTree", "createDataTable", "createEmptyState", "createDiagnostic", "createDiffBlock"]) {
    assert.equal(source.includes(component), true, `missing component ${component}`);
  }
  for (const forbidden of ["ProjectModel", "task readiness", "semantic diff", "filesystem path"]) {
    assert.equal(source.includes(forbidden), false, `component layer contains project rule ${forbidden}`);
  }
  const portal = await readFile(new URL("../src/core/portal.ts", import.meta.url), "utf8");
  const editor = await readFile(new URL("../src/features/editor/index.ts", import.meta.url), "utf8");
  assert.equal(portal.includes('from "../components"'), true, "portal does not use component primitives");
  assert.equal(editor.includes('from "../../components"'), true, "editor does not use dialog primitive");
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
  for (const contract of ["stroke-width", 'role\", \"img', "aria-label", "aria-hidden"]) {
    assert.equal(iconSource.includes(contract), true, `missing icon contract ${contract}`);
  }
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
  for (const required of ["syncBootstrap", "parseBootstrap", "window.ToudocuPage = parsed.value"]) {
    assert.equal(source.includes(required), true, `serve navigation misses ${required}`);
  }
});

test("changes review requests preserve the selected Git range", async () => {
  const source = await readFile(new URL("../src/features/changes/index.ts", import.meta.url), "utf8");
  assert.equal(source.includes("fetch(`${REVIEW}${endpoint}`"), false);
  assert.equal(source.includes("fetch(`${REVIEW}/discussions`"), false);
  for (const required of ["fetch(reviewURL(endpoint)", "fetch(reviewURL('/discussions')"]) {
    assert.equal(source.includes(required), true, `review request bypasses range query: ${required}`);
  }
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
