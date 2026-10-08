export const appName = 'TradeLens';
// Absolute origin for metadata routes (sitemap.xml, robots.txt, OG images, canonical URLs).
// Configure each environment with NEXT_PUBLIC_SITE_URL. Local preview is the default;
// Set NEXT_PUBLIC_SITE_URL to your own origin before publication; no hosted TradeLens is implied.
export const siteUrl = process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3000';
export const docsRoute = '/docs';
export const docsImageRoute = '/og/docs';
export const docsContentRoute = '/llms.mdx/docs';

export const gitConfig = {
  user: 'HoesenBruce',
  repo: 'TradeLens',
  branch: 'main',
};

// Optional upstream-only demo; never an official TradeLens installation.
export const demoConfig = {
  url: 'https://tradermemos.netlify.app',
  user: 'tradermemosdemo',
  password: 'demopassword',
};
