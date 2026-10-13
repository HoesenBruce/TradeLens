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
    // Coach responses must recheck server policy and never replay legacy bytes.
    {
      matcher: ({ url }) => /\/(?:capabilities\/coach|settings\/coach(?:\/|$)|trades\/[^/]+\/coach(?:\/|$)|news\/[^/]+\/analyze$)/.test(url.pathname),
      handler: new NetworkOnly(),
    },
    // Trade/report data: prefer fresh network, fall back to last-seen cache when offline.
    {
      matcher: ({ url }) => url.pathname.startsWith("/api/"),
      handler: new NetworkFirst({
        cacheName: "api-cache",
        networkTimeoutSeconds: 4,
        plugins: [
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
        if (/\/(?:capabilities\/coach|settings\/coach(?:\/|$)|trades\/[^/]+\/coach(?:\/|$)|news\/[^/]+\/analyze$)/.test(new URL(request.url).pathname)) {
          await cache.delete(request);
        }
      }
    }
  })());
});

serwist.addEventListeners();
