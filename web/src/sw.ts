/// <reference lib="esnext" />
/// <reference lib="webworker" />
import { defaultCache } from "@serwist/vite/worker";
import type { PrecacheEntry, SerwistGlobalConfig } from "serwist";
import { CacheableResponsePlugin, ExpirationPlugin, NetworkFirst, NetworkOnly, Serwist } from "serwist";

declare global {
  interface WorkerGlobalScope extends SerwistGlobalConfig {
    __SW_MANIFEST: (PrecacheEntry | string)[] | undefined;
  }
}

declare const self: ServiceWorkerGlobalScope;

// Owner configuration and old provider diagnostics must also disappear on upgrade.
const confidentialApiPath = /\/(?:capabilities\/coach|settings\/(?:coach|ocr|alert-channels)(?:\/|$)|trades\/[^/]+\/coach(?:\/|$)|accounts\/[^/]+\/flex-sync(?:\/|$)|flex-sync(?:\/|$)|news\/[^/]+\/analyze$)/;

const serwist = new Serwist({
  precacheEntries: self.__SW_MANIFEST,
  precacheOptions: { cleanupOutdatedCaches: true },
  // Security containment must replace an old API-caching worker without waiting for a tab to close.
  skipWaiting: true,
  clientsClaim: true,
  navigationPreload: false,
  runtimeCaching: [
    // Release checks must never fall back to an obsolete offline update prompt.
    {
      matcher: ({ url }) => url.origin === "https://api.github.com" && url.pathname.includes("/releases/"),
      handler: new NetworkOnly(),
    },
    // Exports must recheck ownership and current records, never download an offline snapshot.
    {
      matcher: ({ url }) => /\/news\/(?:[^/]+\/)?export$/.test(url.pathname),
      handler: new NetworkOnly(),
    },
    // Confidential responses must recheck server policy and never replay legacy bytes.
    {
      matcher: ({ url }) => confidentialApiPath.test(url.pathname),
      handler: new NetworkOnly(),
    },
    // Trade/report data: prefer fresh network, fall back to last-seen cache when offline.
    {
      matcher: ({ url, request }) => url.pathname.startsWith("/api/") && !request.headers.has("Authorization"),
      handler: new NetworkOnly(),
    },
    {
      matcher: ({ url }) => url.pathname.startsWith("/api/"),
      handler: new NetworkFirst({
        cacheName: "api-cache",
        networkTimeoutSeconds: 4,
        plugins: [
          {
            // Cache API matching ignores Authorization unless the server sets Vary.
            async cacheKeyWillBeUsed({ request }: { request: Request }) {
              const bytes = new TextEncoder().encode(request.headers.get("Authorization") ?? "");
              const digest = await crypto.subtle.digest("SHA-256", bytes);
              const scope = Array.from(new Uint8Array(digest), (b) => b.toString(16).padStart(2, "0")).join("");
              const url = new URL(request.url);
              url.searchParams.set("__tm_auth_scope", scope);
              return new Request(url, { method: request.method });
            },
          },
          new CacheableResponsePlugin({ statuses: [0, 200] }),
          new ExpirationPlugin({ maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 }),
        ],
      }),
    },
    ...defaultCache,
  ],
});

self.addEventListener("message", (event) => {
  if (event.data && event.data.type === "SKIP_WAITING") {
    void self.skipWaiting();
  }
});

self.addEventListener("activate", (event) => {
  event.waitUntil((async () => {
    for (const name of await caches.keys()) {
      const cache = await caches.open(name);
      for (const request of await cache.keys()) {
        const url = new URL(request.url);
        if (confidentialApiPath.test(url.pathname) || (url.pathname.startsWith("/api/") && !url.searchParams.has("__tm_auth_scope"))) {
          await cache.delete(request);
        }
      }
    }
  })());
});

serwist.addEventListeners();
