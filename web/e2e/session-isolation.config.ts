import { resolve } from "node:path";
import { existsSync } from "node:fs";
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  testMatch: "session-isolation.spec.ts",
  workers: 1,
  fullyParallel: false,
  timeout: 30_000,
  reporter: "list",
  use: {
    ...devices["Desktop Chrome"],
    baseURL: "http://127.0.0.1:5189",
    launchOptions: {
      executablePath:
        process.env.E2E_CHROMIUM_PATH ??
        (existsSync("/usr/bin/chromium") ? "/usr/bin/chromium" : undefined),
    },
  },
  webServer: {
    command: "node scripts/serve-security-build.mjs",
    cwd: resolve(import.meta.dirname, ".."),
    url: "http://127.0.0.1:5189",
    reuseExistingServer: false,
  },
});
