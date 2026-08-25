import { defineConfig } from "vitest/config";

export default defineConfig({
  root: ".",
  cacheDir: "./node_modules/.vite/vitest",
  test: { environment: "jsdom", include: ["tests/ui.test.tsx"] },
});
