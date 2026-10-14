import { createServer } from "node:http";
import { readFileSync } from "node:fs";
import { resolve, extname } from "node:path";
import { test, expect, type Page } from "@playwright/test";

const api = process.env.E2E_API_URL ?? "http://127.0.0.1:8099/api/v1";
const password = "Synthetic-pr306-password-123!";
const owner = "pr306-owner@example.invalid";
const member = "pr306-member@example.invalid";
const dist = resolve(process.env.E2E_WORKER_DIST ?? "dist");
let ownerToken: string;
let memberToken: string;

test.beforeAll(async ({ request }) => {
  const status = await (await request.get(`${api}/setup/status`)).json();
  if (status.needs_setup)
    expect((await request.post(`${api}/setup`, { data: { email: owner, password } })).ok()).toBe(
      true,
    );
  const login = async (email: string) => {
    const response = await request.post(`${api}/auth/login`, { data: { email, password } });
    expect(response.status()).toBe(200);
    return (await response.json()).access_token as string;
  };
  ownerToken = await login(owner);
  const headers = { Authorization: `Bearer ${ownerToken}` };
  const users = await (await request.get(`${api}/admin/users`, { headers })).json();
  if (!users.some((user: { email: string }) => user.email === member)) {
    expect(
      (
        await request.post(`${api}/admin/users`, {
          headers,
          data: { email: member, password, is_admin: false },
        })
      ).ok(),
    ).toBe(true);
  }
  memberToken = await login(member);
});

// Return categories only, so assertion failures cannot print private bodies.
async function fetchStatus(page: Page, path: string, token: string) {
  return page.evaluate(
    async ({ path, token }) => {
      try {
        const response = await fetch(path, { headers: { Authorization: `Bearer ${token}` } });
        await response.arrayBuffer();
        return response.status;
      } catch {
        return 0;
      }
    },
    { path, token },
  );
}

function fixtureServer(prefix: string, legacy = false, scope = "/") {
  let upgraded = !legacy;
  const server = createServer(async (req, res) => {
    const url = new URL(req.url!, "http://localhost");
    const path = url.pathname;
    res.setHeader("Cache-Control", "no-store");
    if (path === `${scope}probe`) {
      res.setHeader("Content-Type", "text/html");
      res.end("<html><body>Synthetic worker probe</body></html>");
      return;
    }
    if (path === `${scope}sw.js` && !upgraded) {
      res.setHeader("Content-Type", "application/javascript");
      res.end(
        "self.addEventListener('install',()=>self.skipWaiting());self.addEventListener('activate',e=>e.waitUntil(self.clients.claim()));self.addEventListener('fetch',e=>e.respondWith(caches.match(e.request).then(r=>r||fetch(e.request))));",
      );
      return;
    }
    if (path.startsWith(`${prefix}/`)) {
      try {
        const response = await fetch(api + path.slice(prefix.length) + url.search, {
          headers: req.headers.authorization ? { Authorization: req.headers.authorization } : {},
        });
        res.writeHead(response.status, { "Content-Type": "application/json" });
        res.end(Buffer.from(await response.arrayBuffer()));
      } catch {
        res.writeHead(502).end();
      }
      return;
    }
    try {
      const file = resolve(
        dist,
        `.${path.startsWith(scope) ? "/" + path.slice(scope.length) : path}`,
      );
      if (!file.startsWith(dist + "/")) throw new Error("path");
      res.setHeader(
        "Content-Type",
        { ".js": "application/javascript", ".html": "text/html", ".css": "text/css" }[
          extname(file)
        ] ?? "application/octet-stream",
      );
      res.end(readFileSync(file));
    } catch {
      res.writeHead(404).end();
    }
  });
  return {
    server,
    upgrade: () => {
      upgraded = true;
    },
  };
}

// The worker-security project repeats every case in three fresh Chromium contexts.
for (const [prefix, scope] of [
  ["/api/v1", "/"],
  ["/gateway/api/v1", "/"],
  ["/tenant/backend", "/"],
  ["/gateway/api/v1", "/app/"],
]) {
  test(`PREFIX_BEARER_X3 ${prefix} scope ${scope}`, async ({ page, context }) => {
    const { server } = fixtureServer(prefix, false, scope);
    await new Promise<void>((done) => server.listen(0, "127.0.0.1", done));
    const origin = `http://127.0.0.1:${(server.address() as { port: number }).port}`;
    try {
      await page.goto(`${origin}${scope}probe`);
      await page.evaluate(async (scope) => {
        await navigator.serviceWorker.register(`${scope}sw.js`);
        await navigator.serviceWorker.ready;
      }, scope);
      await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
      for (const route of ["/admin/users", "/admin/backup", "/me"]) {
        await context.setOffline(false);
        expect(await fetchStatus(page, prefix + route, ownerToken)).toBe(200);
        expect(await fetchStatus(page, prefix + route, memberToken)).toBe(
          route === "/me" ? 200 : 403,
        );
        await context.setOffline(true);
        expect(await fetchStatus(page, prefix + route, memberToken)).toBe(0);
        expect(await fetchStatus(page, prefix + route, ownerToken)).toBe(0);
      }
      expect(
        await page.evaluate(async (prefix) => {
          for (const name of await caches.keys()) {
            for (const key of await (await caches.open(name)).keys()) {
              if (new URL(key.url).pathname.startsWith(`${prefix}/`)) return true;
            }
          }
          return false;
        }, prefix),
      ).toBe(false);
    } finally {
      await context.setOffline(false);
      await new Promise<void>((done) => server.close(() => done()));
    }
  });
}

test("PREFIX_CACHE_MIGRATION", async ({ page, context }) => {
  const { server, upgrade } = fixtureServer("/gateway/api/v1", true);
  await new Promise<void>((done) => server.listen(0, "127.0.0.1", done));
  const origin = `http://127.0.0.1:${(server.address() as { port: number }).port}`;
  const paths = [
    "/api/v1/me",
    "/gateway/api/v1/me",
    "/gateway/api/v1/admin/users",
    "/gateway/api/v1/admin/backup",
    "/gateway/api/v1/settings/ocr",
    "/gateway/api/v1/trades/fixture/coach/reviews",
    "/gateway/api/v1/export.json",
    "/tenant/backend/me?__tm_auth_scope=TEST_USER_A",
  ];
  try {
    await page.goto(`${origin}/probe`);
    await page.evaluate(async () => {
      await navigator.serviceWorker.register("/sw.js");
      await navigator.serviceWorker.ready;
    });
    await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
    await page.evaluate(
      async ({ paths }) => {
        for (const name of ["api-cache", "apis", "others", "custom-old-cache"]) {
          const cache = await caches.open(name);
          for (const path of paths) await cache.put(path, new Response("TEST_CONFIG_A"));
          if (name !== "custom-old-cache")
            await cache.put("/tenant/backend/admin/users", new Response("TEST_CONFIG_A"));
        }
        const publicCache = await caches.open("synthetic-public-cache");
        await publicCache.put("/public-probe.txt", new Response("TEST_PUBLIC_ASSET"));
        await publicCache.put(
          new Request(location.origin + "/tenant/backend/admin/backup", {
            headers: { Authorization: "Bearer TEST_USER_A" },
          }),
          new Response("TEST_CONFIG_A"),
        );
      },
      { paths },
    );
    expect(
      await page.evaluate(
        async () => (await (await fetch("/gateway/api/v1/admin/users")).text()) === "TEST_CONFIG_A",
      ),
    ).toBe(true);
    upgrade();
    await page.evaluate(async () => {
      await (await navigator.serviceWorker.getRegistration())!.update();
    });
    await expect
      .poll(() =>
        page.evaluate(async () => {
          const names = await caches.keys();
          if (["api-cache", "apis", "others"].some((name) => names.includes(name))) return false;
          for (const name of names) {
            for (const key of await (await caches.open(name)).keys()) {
              if (
                key.url.includes("/api/") ||
                key.url.includes("__tm_auth_scope") ||
                key.headers.has("Authorization")
              )
                return false;
            }
          }
          return true;
        }),
      )
      .toBe(true);
    expect(
      await page.evaluate(
        async () =>
          (await (await caches.open("synthetic-public-cache")).match("/public-probe.txt")) !==
          undefined,
      ),
    ).toBe(true);
    expect(await page.evaluate(async () => (await fetch("/favicon.ico")).status)).toBe(200);
    await context.setOffline(true);
    for (const path of paths) expect(await fetchStatus(page, path, memberToken)).toBe(0);
    expect(await fetchStatus(page, "/tenant/backend/admin/users", memberToken)).toBe(0);
    expect(await page.evaluate(async () => (await fetch("/favicon.ico")).status)).toBe(200);
  } finally {
    await context.setOffline(false);
    await new Promise<void>((done) => server.close(() => done()));
  }
});
