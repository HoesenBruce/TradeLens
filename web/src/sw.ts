/// <reference lib="esnext" />
/// <reference lib="webworker" />
import { defaultCache } from "@serwist/vite/worker";
import type { PrecacheEntry, SerwistGlobalConfig } from "serwist";
import { NetworkOnly, Serwist } from "serwist";
import { isNetworkOnlyRequest, isRetiredApiEntry, retiredApiCaches } from "./lib/workerCachePolicy";

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
    // Check authorization before any generic static/API fallback, for every API base.
    {
      matcher: ({ request }) => isNetworkOnlyRequest(request),
      handler: new NetworkOnly(),
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
      if (retiredApiCaches.has(name)) {
        await caches.delete(name);
        continue;
      }
      const cache = await caches.open(name);
      for (const request of await cache.keys()) {
        if (isRetiredApiEntry(request)) {
          await cache.delete(request);
        }
      }
    }
  })());
});

serwist.addEventListeners();
