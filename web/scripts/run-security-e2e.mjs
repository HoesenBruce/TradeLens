import { spawn, spawnSync } from "node:child_process";
import { randomBytes } from "node:crypto";
import { createServer } from "node:http";
import { mkdtempSync, openSync, closeSync, rmSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import { fileURLToPath } from "node:url";

const web = fileURLToPath(new URL("..", import.meta.url));
const temp = mkdtempSync(join(tmpdir(), "pr306-security-"));
const binary = join(temp, "api");
const apiLog = openSync(join(temp, "api.log"), "w");
let apiProcess;
let browserProcess;
const market = createServer((req, res) => {
  const query = new URL(req.url, "http://localhost").searchParams;
  res.setHeader("Content-Type", "application/json");
  res.end(
    JSON.stringify({
      symbol: query.get("symbol"),
      interval: query.get("interval"),
      bars: [],
      timezone: "America/New_York",
      adjustment_status: "unadjusted",
    }),
  );
});

async function stop(child) {
  if (!child || child.exitCode !== null || child.signalCode !== null) return;
  await new Promise((done) => {
    const timeout = setTimeout(() => child.kill("SIGKILL"), 5000);
    child.once("exit", () => {
      clearTimeout(timeout);
      done();
    });
    child.kill("SIGTERM");
  });
}
for (const signal of ["SIGINT", "SIGTERM"]) {
  process.on(signal, () => {
    browserProcess?.kill(signal);
    apiProcess?.kill(signal);
  });
}

try {
  // Refuse an existing service: every run must use its own disposable database.
  let occupied = false;
  try {
    await fetch("http://127.0.0.1:8099/api/v1/setup/status", { signal: AbortSignal.timeout(1000) });
    occupied = true;
  } catch {
    /* no API listening */
  }
  if (occupied) throw new Error("DisposableApiPortInUse");
  const build = spawnSync(
    process.env.E2E_GO_COMMAND ?? "go",
    ["build", "-o", binary, "./cmd/server"],
    { cwd: resolve(web, "../api"), stdio: "inherit", env: process.env },
  );
  if (build.status !== 0) throw new Error("DisposableApiBuildFailed");
  await new Promise((done, reject) => {
    market.once("error", reject);
    market.listen(8100, "127.0.0.1", done);
  });
  apiProcess = spawn(binary, [], {
    cwd: temp,
    stdio: ["ignore", apiLog, apiLog],
    env: {
      ...process.env,
      TM_HTTP_PORT: "8099",
      TM_DATABASE_URL: `sqlite:${join(temp, "qa.db")}`,
      TM_DB_PATH: join(temp, "qa.db"),
      TM_JWT_SECRET: randomBytes(32).toString("hex"),
      TM_JOBS_ENABLED: "false",
      TM_MARKET_DATA_PROVIDER: "http",
      TM_MARKET_DATA_HTTP_BASE_URL: "http://127.0.0.1:8100",
      TM_CORS_ORIGINS: "http://localhost:5189,http://127.0.0.1:5189",
      TM_BACKUP_DIR: join(temp, "backups"),
    },
  });
  let ready = false;
  let readinessStatus = 0;
  let setupRequired = false;
  let readinessError = "";
  for (let attempt = 0; attempt < 300; attempt++) {
    if (apiProcess.exitCode !== null) throw new Error("DisposableApiStartupFailed");
    try {
      const response = await fetch("http://127.0.0.1:8099/api/v1/setup/status", {
        signal: AbortSignal.timeout(1000),
      });
      readinessStatus = response.status;
      setupRequired = (await response.json()).needs_setup === true;
      if (response.ok && setupRequired) {
        ready = true;
        break;
      }
    } catch (error) {
      readinessError = error.name;
      /* wait for the newly built local service */
    }
    await new Promise((done) => setTimeout(done, 100));
  }
  if (!ready) {
    console.error("API readiness categories:", {
      status: readinessStatus,
      setupRequired,
      error: readinessError,
    });
    throw new Error("DisposableApiNotReady");
  }
  browserProcess = spawn(
    process.execPath,
    [
      resolve(web, "node_modules/@playwright/test/cli.js"),
      "test",
      "-c",
      "e2e/security.config.ts",
      ...process.argv.slice(2),
    ],
    {
      cwd: web,
      stdio: "inherit",
      env: { ...process.env, E2E_API_URL: "http://127.0.0.1:8099/api/v1" },
    },
  );
  process.exitCode = await new Promise((done, reject) => {
    browserProcess.once("error", reject);
    browserProcess.once("exit", (code) => done(code ?? 1));
  });
} catch (error) {
  console.error("Synthetic security harness failure:", error.message);
  if (error.message === "DisposableApiStartupFailed") {
    const diagnostic = readFileSync(join(temp, "api.log"), "utf8");
    const category = /address already in use/i.test(diagnostic)
      ? "PortInUse"
      : /permission denied|read-only/i.test(diagnostic)
        ? "FilesystemPermission"
        : /jwt|auth/i.test(diagnostic)
          ? "AuthConfiguration"
          : /market|provider/i.test(diagnostic)
            ? "MarketConfiguration"
            : /database|sqlite|migration/i.test(diagnostic)
              ? "DatabaseStartup"
              : "UnknownStartupError";
    console.error("API startup category:", category);
  }
  process.exitCode = 1;
} finally {
  await stop(browserProcess);
  await stop(apiProcess);
  if (market.listening) await new Promise((done) => market.close(done));
  closeSync(apiLog);
  rmSync(temp, { recursive: true, force: true });
}
