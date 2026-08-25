import { defineConfig } from "vitest/config";

const root = new URL(".", import.meta.url).pathname;

export default defineConfig({
  root,
  cacheDir: `${root}node_modules/.vite/vitest`,
  test: { environment: "jsdom", include: ["tests/ui.test.tsx", "tests/island-host.test.ts"] },
});
