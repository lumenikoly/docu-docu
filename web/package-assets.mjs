import { createHash } from "node:crypto";
import { copyFile, mkdir, readdir, readFile, rename, rm, writeFile } from "node:fs/promises";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = dirname(fileURLToPath(import.meta.url));
const repositoryRoot = resolve(webRoot, "..");
const viteOutput = join(webRoot, "dist");
const output = join(repositoryRoot, "internal", "site", "assets", "generated");
const staging = `${output}.staging`;
const expected = join(repositoryRoot, "internal", "site", "assets", "generated");

if (output !== expected || !output.endsWith(`${sep}internal${sep}site${sep}assets${sep}generated`)) {
  throw new Error(`Unsafe generated asset path: ${output}`);
}

const vendorFiles = [
  "mermaid.tiny.js", "mermaid.LICENSE.txt", "swagger-ui.css",
  "swagger-ui-bundle.js", "swagger-ui-standalone-preset.js",
  "swagger-ui.LICENSE.txt", "swagger-ui-bundle.LICENSE.txt",
  "swagger-ui-standalone-preset.LICENSE.txt", "swagger-ui.checksums.txt",
  "codemirror.LICENSE.txt",
];
const staticRoots = [
  "appearance", "portal", "portal-style", "screen-map", "screen-map-style",
  "playable-flow", "playable-flow-style",
];
const serveRoots = [
  ...staticRoots, "serve", "serve-style", "editor", "editor-style", "changes",
  "changes-style", "codemirror", "api-docs",
];
const staticVendor = ["manifest.json", "mermaid.tiny.js", "mermaid.LICENSE.txt", "favicon.svg"];
const serveVendor = ["manifest.json", ...vendorFiles, "codemirror.checksums.txt", "favicon.svg"];

const manifest = {
  ...JSON.parse(await readFile(join(viteOutput, ".vite", "manifest.json"), "utf8")),
  ...JSON.parse(await readFile(join(viteOutput, ".vite", "appearance-manifest.json"), "utf8")),
};
const entries = new Map(Object.entries(manifest).filter(([, value]) => value.isEntry).map(([key, value]) => [value.name, key]));

function closure(roots) {
  const found = new Set();
  const visit = (key) => {
    const chunk = manifest[key];
    if (!chunk) throw new Error(`Vite manifest entry not found: ${key}`);
    if (found.has(chunk.file)) return;
    found.add(chunk.file);
    for (const file of [...(chunk.css || []), ...(chunk.assets || [])]) found.add(file);
    for (const dependency of [...(chunk.imports || []), ...(chunk.dynamicImports || [])]) visit(dependency);
  };
  for (const root of roots) {
    const key = entries.get(root);
    if (!key) throw new Error(`Vite entry not found: ${root}`);
    visit(key);
  }
  return [...found].sort();
}

async function filesBelow(directory, base = directory) {
  const result = [];
  for (const entry of (await readdir(directory, { withFileTypes: true })).sort((a, b) => a.name.localeCompare(b.name))) {
    const absolute = join(directory, entry.name);
    if (entry.isDirectory()) result.push(...await filesBelow(absolute, base));
    else if (entry.isFile()) result.push(relative(base, absolute).split(sep).join("/"));
  }
  return result;
}

async function copy(source, destination) {
  await mkdir(dirname(destination), { recursive: true });
  await copyFile(source, destination);
}

async function sha256(path) {
  return createHash("sha256").update(await readFile(path)).digest("hex");
}

await rm(staging, { recursive: true, force: true });
await mkdir(staging, { recursive: true });
const viteFiles = new Set([...closure(serveRoots), "licenses.json"]);
for (const file of [...viteFiles].sort()) await copy(join(viteOutput, file), join(staging, file));
await writeFile(join(staging, "vite-manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`);
await copy(join(webRoot, "public", "favicon.svg"), join(staging, "favicon.svg"));
for (const file of vendorFiles) await copy(join(webRoot, "vendor", file), join(staging, file));

const codeMirrorChecksums = ["codemirror.js", "codemirror.LICENSE.txt"];
const checksumLines = [];
for (const file of codeMirrorChecksums) checksumLines.push(`${await sha256(join(staging, file))}  ${file}`);
await writeFile(join(staging, "codemirror.checksums.txt"), `${checksumLines.join("\n")}\n`);

const generated = (await filesBelow(staging)).filter((file) => file !== "manifest.json");
const assets = {};
for (const file of generated) assets[file] = { file, sha256: await sha256(join(staging, file)) };
const runtimeManifest = {
  schemaVersion: 1,
  assets,
  runtimes: {
    static: [...new Set([...closure(staticRoots), ...staticVendor])].sort(),
    serve: [...new Set([...closure(serveRoots), ...serveVendor])].sort(),
  },
};
await writeFile(join(staging, "manifest.json"), `${JSON.stringify(runtimeManifest, null, 2)}\n`);
await rm(output, { recursive: true, force: true });
await rename(staging, output);
