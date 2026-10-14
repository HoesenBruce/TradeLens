import { expect, test } from "@playwright/test";
const api = process.env.E2E_API_URL ?? "http://localhost:8099/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5189";
test("legacy AI news is unavailable through a real API", async ({ page, request }) => {
  const auth = await (
    await request.post(`${api}/auth/login`, {
      data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
    })
  ).json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  const created = await request.post(`${api}/news`, {
    headers,
    data: {
      title: "Synthetic unavailable AI",
      source: "Manual",
      published_at: "2026-10-12T00:00:00Z",
      summary: "Manual summary",
    },
  });
  expect(created.status()).toBe(201);
  const news = await created.json();
  try {
    expect((await request.post(`${api}/news/${news.id}/analyze`, { headers })).status()).toBe(503);
    await page.addInitScript(
      ({ token, api }) => {
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
      },
      { token: auth.access_token, api },
    );
    await page.goto(`${web}/news/${news.id}`);
    await expect(
      page.getByText(
        "AI news analysis is temporarily unavailable. Manual news review remains available.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(page.getByRole("button", { name: "Analyze with AI", exact: true })).toHaveCount(0);
    expect((await (await request.get(`${api}/news/${news.id}`, { headers })).json()).summary).toBe(
      "Manual summary",
    );
  } finally {
    await request.delete(`${api}/news/${news.id}`, { headers });
  }
});
