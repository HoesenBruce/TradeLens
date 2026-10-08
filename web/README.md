<div align="center">

# TradeLens web (SPA)

Vite + React journal UI. Point it at **your** Go API (parent monorepo `../api`).

<br/>

[![Deploy with Vercel](https://vercel.com/button)](<https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens&root-directory=web&project-name=tradelens&repository-name=tradelens&env=VITE_API&envDescription=Optional%20API%20base%20URL%20(e.g.%20https%3A%2F%2Fapi.example.com%2Fapi%2Fv1).%20Leave%20empty%20to%20set%20Server%20at%20login.&envLink=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens%2Fblob%2Fmain%2Fdocs%2Ffork-deploy.md>)
[![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https%3A%2F%2Fgithub.com%2FHoesenBruce%2FTradeLens%2Ftree%2Fmain%2Fweb)
[![Deploy to Netlify](https://www.netlify.com/img/deploy/button.svg)](https://app.netlify.com/start/deploy?repository=https://github.com/HoesenBruce/TradeLens)
[Railway API setup](https://github.com/HoesenBruce/TradeLens/blob/main/docs/fork-deploy.md#railway-api)

<br/>

[Full fork / deploy guide](../docs/fork-deploy.md)

</div>

## After the UI is live

1. Run the API — [Railway](https://github.com/HoesenBruce/TradeLens/blob/main/docs/fork-deploy.md#railway-api) (Volume at `/data`) or Docker/`make up` at repo root.
2. Allow your CDN origin:
   ```bash
   TM_CORS_ORIGINS=https://*.vercel.app,https://*.pages.dev,https://*.workers.dev,https://*.netlify.app,https://*.up.railway.app,http://localhost:5173
   ```
3. Open the site → login **Server** = your API origin (or set `VITE_API` at build time).

Vercel, Cloudflare and Netlify target the `web/` SPA only; Railway setup targets the Go API separately. Private repository access and provider authorization are required. These links do not deploy `marketing/` or a complete stack; provider deployment is unverified.
