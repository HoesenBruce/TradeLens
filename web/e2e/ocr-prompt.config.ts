import { defineConfig } from "@playwright/test";
import base from "../playwright.config";

export default defineConfig({
  ...base,
  testDir: ".",
  testMatch: "ocr-prompt.spec.ts",
  projects: [
    { name: "chrome", use: { channel: "chrome" } },
    { name: "webkit", use: { browserName: "webkit" } },
  ],
});
