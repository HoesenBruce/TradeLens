import { expect, test } from "@playwright/test";

// Exercise the live API, without the service worker serving cached responses.
test.use({ serviceWorkers: "block" });

const api = process.env.E2E_API_URL ?? "http://localhost:8080/api/v1";
const web = process.env.E2E_WEB_URL ?? "http://localhost:5173";

test("manual predictions persist, cancel cleanly, validate and delete", async ({
  page,
  request,
}, info) => {
  const login = await request.post(`${api}/auth/login`, {
    data: { email: process.env.E2E_EMAIL, password: process.env.E2E_PASSWORD },
  });
  expect(login.ok()).toBeTruthy();
  const auth = await login.json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  const created = await request.post(`${api}/news`, {
    headers,
    data: {
      title: `Prediction E2E ${Date.now()}`,
      source: "E2E",
      published_at: "2026-09-20T00:00:00Z",
      assets: [{ asset_type: "stock", symbol: "285A", market: "JP" }],
    },
  });
  expect(created.status()).toBe(201);
  const news = await created.json();
  await page.addInitScript(
    ({ token, api }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
    },
    { token: auth.access_token, api },
  );
  try {
    await page.goto(`${web}/news`);
    await page.getByRole("button", { name: "Reset filters", exact: true }).first().click();
    await page
      .getByRole("row")
      .filter({ hasText: news.title })
      .getByRole("button", { name: "Predictions", exact: true })
      .click();
    const section = page.getByRole("region", { name: `Predictions for ${news.title}` });
    await section.getByRole("button", { name: "Add prediction" }).click();
    let dialog = page.getByRole("dialog");
    await dialog.getByRole("textbox", { name: "Reasoning" }).fill("Abandoned draft");
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await section.getByRole("button", { name: "Add prediction" }).click();
    await expect(dialog.getByRole("textbox", { name: "Reasoning" })).toHaveValue("");
    await dialog.getByRole("combobox", { name: "Direction" }).selectOption("bullish");
    await dialog.getByRole("spinbutton", { name: "Confidence (%)" }).fill("101");
    await dialog.getByRole("button", { name: "Save prediction" }).click();
    await expect(dialog).toBeVisible();
    expect(
      (await (await request.get(`${api}/news/${news.id}`, { headers })).json()).predictions,
    ).toHaveLength(0);
    await dialog.getByRole("spinbutton", { name: "Confidence (%)" }).fill("80");
    for (const label of ["Reasoning", "Catalysts", "Risks", "Invalidation"])
      await dialog
        .getByRole("textbox", { name: label, exact: true })
        .fill(`${label} at prediction time`);
    await dialog.getByRole("checkbox", { name: "5D", exact: true }).check();
    await dialog.getByRole("checkbox", { name: "20D", exact: true }).check();
    await page.screenshot({ path: info.outputPath("prediction-editor.png"), fullPage: true });
    await dialog.getByRole("button", { name: "Save prediction" }).click();
    await expect(dialog).toBeHidden();
    await expect(section.getByText("80%", { exact: true })).toBeVisible();
    await page.reload();
    await page
      .getByRole("row")
      .filter({ hasText: news.title })
      .getByRole("button", { name: "Predictions", exact: true })
      .click();
    await section.getByRole("button", { name: "Edit prediction" }).click();
    await expect(dialog.getByRole("spinbutton", { name: "Confidence (%)" })).toHaveValue("80");
    await expect(dialog.getByRole("checkbox", { name: "5D", exact: true })).toBeChecked();
    await dialog.getByRole("combobox", { name: "Direction" }).selectOption("bearish");
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await section.getByRole("button", { name: "Edit prediction" }).click();
    await expect(dialog.getByRole("combobox", { name: "Direction" })).toHaveValue("bullish");
    await dialog.getByRole("combobox", { name: "Direction" }).selectOption("bearish");
    await dialog.getByRole("spinbutton", { name: "Confidence (%)" }).fill("");
    for (const h of [1, 5, 20])
      await dialog.getByRole("checkbox", { name: `${h}D`, exact: true }).uncheck();
    await dialog.getByRole("button", { name: "Save prediction" }).click();
    await expect(dialog.getByRole("alert")).toContainText("Select at least one");
    for (const h of [3, 10])
      await dialog.getByRole("checkbox", { name: `${h}D`, exact: true }).check();
    await dialog.getByRole("button", { name: "Save prediction" }).click();
    await expect(dialog).toBeHidden();
    const saved = (await (await request.get(`${api}/news/${news.id}`, { headers })).json())
      .predictions[0];
    expect(saved).toMatchObject({
      direction: "bearish",
      confidence: null,
      horizons: [3, 10],
      source: "user",
      risks: "Risks at prediction time",
    });
    await page.screenshot({ path: info.outputPath("prediction-saved.png"), fullPage: true });
    await section.getByRole("button", { name: "Delete prediction" }).click();
    await dialog.getByRole("button", { name: "Cancel" }).click();
    await expect(section.getByText("bearish", { exact: true })).toBeVisible();
    await section.getByRole("button", { name: "Delete prediction" }).click();
    await dialog.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(section.getByText("No predictions yet.")).toBeVisible();
    expect(
      (await (await request.get(`${api}/news/${news.id}`, { headers })).json()).predictions,
    ).toHaveLength(0);
  } finally {
    await request.delete(`${api}/news/${news.id}`, { headers });
  }
});
