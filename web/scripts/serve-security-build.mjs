import { createServer } from "node:http";
import { readFileSync } from "node:fs";
import { extname, resolve } from "node:path";

const dist = resolve(process.env.E2E_WORKER_DIST ?? "dist");
const api = process.env.E2E_API_URL ?? "http://localhost:8099/api/v1";
createServer(async (req, res) => {
  const path = new URL(req.url, "http://localhost").pathname;
  // The closed FX calculator mounts eagerly; keep this unrelated vendor synthetic.
  if (path === "/api/v1/market/fx") {
    const query = new URL(req.url, "http://localhost").searchParams;
    res.setHeader("Content-Type", "application/json");
    res.end(
      JSON.stringify({
        from: query.get("from"),
        to: query.get("to"),
        rate: 1,
        as_of: "2026-01-06",
        provider: "synthetic",
      }),
    );
    return;
  }
  if (path.startsWith("/api/v1/")) {
    try {
      const chunks = [];
      for await (const chunk of req) chunks.push(chunk);
      const response = await fetch(api.replace(/\/api\/v1$/, "") + req.url, {
        method: req.method,
        headers: {
          "Content-Type": "application/json",
          ...(req.headers.authorization ? { Authorization: req.headers.authorization } : {}),
        },
        ...(req.method !== "GET" && req.method !== "HEAD" ? { body: Buffer.concat(chunks) } : {}),
      });
      res.writeHead(response.status, {
        "Content-Type": response.headers.get("content-type") ?? "application/json",
      });
      res.end(Buffer.from(await response.arrayBuffer()));
    } catch (error) {
      console.error("Synthetic proxy failure", path, error.name);
      res.writeHead(502).end();
    }
    return;
  }
  let file = resolve(dist, path === "/" ? "index.html" : `.${path}`);
  if (!file.startsWith(dist + "/")) {
    res.writeHead(400).end();
    return;
  }
  try {
    if (!extname(path)) file = resolve(dist, "index.html");
    res.setHeader(
      "Content-Type",
      {
        ".js": "application/javascript",
        ".css": "text/css",
        ".html": "text/html",
        ".svg": "image/svg+xml",
        ".webmanifest": "application/manifest+json",
      }[extname(file)] ?? "application/octet-stream",
    );
    res.setHeader("Cache-Control", "no-store");
    res.end(readFileSync(file));
  } catch {
    res.writeHead(404).end();
  }
}).listen(5189, "127.0.0.1");
