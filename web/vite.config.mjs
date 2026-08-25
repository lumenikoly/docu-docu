import { resolve } from "node:path";
import { defineConfig } from "vite";

const entries = [
  "appearance", "portal", "serve", "editor", "changes", "screen-map",
  "playable-flow", "api-docs", "codemirror",
];
const styles = ["portal", "serve", "editor", "changes", "screen-map", "playable-flow"];

export default defineConfig(({ mode }) => ({
  base: "./",
  build: {
    outDir: "dist",
    emptyOutDir: mode !== "appearance",
    target: "es2020",
    manifest: mode === "appearance" ? ".vite/appearance-manifest.json" : true,
    license: mode === "appearance" ? false : { fileName: "licenses.json" },
    rolldownOptions: {
      input: mode === "appearance"
        ? { appearance: resolve(import.meta.dirname, "src/entries/appearance.ts") }
        : Object.fromEntries([
        ...entries.filter((name) => name !== "appearance").map((name) => [name, resolve(import.meta.dirname, `src/entries/${name}.ts`)]),
        ...styles.map((name) => [`${name}-style`, resolve(import.meta.dirname, `src/styles/${name}.css`)]),
      ]),
      output: {
        // ponytail: keep the pre-CSS appearance entry classic; merge its manifest after this one-entry build.
        codeSplitting: mode !== "appearance",
        entryFileNames: "[name].js",
        chunkFileNames: "chunks/[name]-[hash].js",
        assetFileNames: ({ names }) => {
          const style = names.find((name) => name.endsWith("-style.css"));
          return style ? style.replace("-style.css", ".css") : "assets/[name]-[hash][extname]";
        },
      },
    },
  },
}));
