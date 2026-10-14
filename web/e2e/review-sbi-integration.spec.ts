/// <reference types="node" />
import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import type { Trade, TradeDetail } from "../src/lib/api/types";

test.use({ viewport: { width: 1280, height: 1200 } });

// Run only against a disposable API with synthetic market data enabled.
const api = process.env.E2E_API_URL ?? "http://localhost:8096/api/v1";

test("SBI review grade and journal stay linked across same-minute reverse reimport", async ({
  page,
  request,
}, info) => {
  const credentials = { email: "review273@example.com", password: "Review273-test-pass" };
  const setup = await request.post(`${api}/setup`, { data: credentials });
  const auth = setup.ok()
    ? await setup.json()
    : await (await request.post(`${api}/auth/login`, { data: credentials })).json();
  const headers = { Authorization: `Bearer ${auth.access_token}` };
  const post = async (path: string, data: unknown) => {
    const response = await request.post(`${api}${path}`, { headers, data });
    expect(response.ok(), await response.text()).toBeTruthy();
    return response.json();
  };
  const account = await post("/accounts", {
    name: `Synthetic Review SBI ${Date.now()}`,
    broker: "SBI Securities",
    account_type: "margin",
    base_currency: "JPY",
  });
  const fixture = readFileSync("../api/internal/importer/testdata/sbi-review-direction.csv");
  const commit = async (buffer: Buffer, inserted: number, skipped: number) => {
    const file = { name: "synthetic-review-sbi.csv", mimeType: "text/csv", buffer };
    const preview = await request.post(`${api}/imports`, {
      headers,
      multipart: { account_id: account.id, file },
    });
    expect(preview.ok(), await preview.text()).toBeTruthy();
    const hints = await preview.json();
    expect(hints.detected_broker).toContain("SBI");
    const response = await request.post(`${api}/imports/commit`, {
      headers,
      multipart: {
        account_id: account.id,
        file,
        column_mapping: JSON.stringify(hints.suggested_mapping),
        source_tz: hints.suggested_source_tz,
      },
    });
    expect(response.ok(), await response.text()).toBeTruthy();
    expect(await response.json()).toMatchObject({ inserted, skipped });
  };
  await commit(fixture, 9, 0);
  const list = await request.get(`${api}/trades?account_id=${account.id}`, { headers });
  expect(list.ok()).toBeTruthy();
  const trades: Trade[] = await list.json();
  expect(trades).toHaveLength(4);
  const margin = trades.find((trade) => trade.qty_opened === 20)!;
  expect(margin).toMatchObject({ direction: "long", status: "closed" });
  expect(trades.find((trade) => trade.direction === "short")).toMatchObject({
    status: "open",
    qty_remaining: 10,
  });
  const plans = await Promise.all(
    ["one", "two"].map((name) =>
      post("/setups", { name: `SBI planned short ${name} ${Date.now()}`, direction: "short" }),
    ),
  );
  const notes =
    "SBI legacy preserved\n\n## Session\nAsia\n\n## Planned direction\nshort\n\n## Entry reason\nSBI entry\n\n## Exit reason\nSBI exit\n\n## Review notes\nold SBI lesson";
  const prepared = await request.patch(`${api}/trades/${margin.id}`, {
    headers,
    data: {
      notes,
      setup_ids: plans.map((plan) => plan.id),
      initial_risk: 20,
      target_price: 150,
      stop_price: 80,
      confidence: 4,
    },
  });
  expect(prepared.ok()).toBeTruthy();
  const detail = async (): Promise<TradeDetail> => {
    const response = await request.get(`${api}/trades/${margin.id}`, { headers });
    expect(response.ok()).toBeTruthy();
    return response.json();
  };
  const before = await detail();
  await page.addInitScript(
    ({ token, api, accountId }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [accountId] }, version: 1 }),
      );
    },
    { token: auth.access_token, api, accountId: account.id },
  );
  await page.goto("/review");
  await expect(page.getByRole("button", { name: "Recent (0)", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Backlog (3)", exact: true }).click();
  // Synthetic microseconds collapse to a millisecond in Web; locate the saved
  // journal rather than treating ID tie order as execution chronology.
  for (let skipped = 0; skipped < 3; skipped++) {
    const field = page.getByRole("textbox", { name: "Review notes" });
    await expect(field).toBeVisible();
    if ((await field.inputValue()) === "old SBI lesson") break;
    const next = page.waitForResponse(
      (response) =>
        /\/trades\/[^/?]+$/.test(response.url()) && response.request().method() === "GET",
    );
    await page.getByRole("button", { name: "Skip", exact: true }).click();
    expect((await next).ok()).toBeTruthy();
  }
  await expect(page.getByRole("heading", { name: /733A · long/ })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Review notes" })).toHaveValue("old SBI lesson");
  await page.getByRole("button", { name: "A+", exact: true }).click();
  await page.getByRole("textbox", { name: "Review notes" }).fill("SBI grade persisted");
  await page.screenshot({
    path: info.outputPath("sbi-review-planned-short-actual-long.png"),
    fullPage: true,
  });
  const saved = page.waitForResponse(
    (response) =>
      response.url().endsWith(`/trades/${margin.id}`) && response.request().method() === "PATCH",
  );
  await page.getByRole("button", { name: "Save and next", exact: true }).click();
  expect((await saved).ok()).toBeTruthy();
  const after = await detail();
  expect(after).toMatchObject({
    direction: "long",
    trade_quality: 5,
    setup_ids: plans.map((plan) => plan.id),
  });
  expect(after.notes).toBe(notes.replace("old SBI lesson", "SBI grade persisted"));
  const unrelated = ({
    notes: _notes,
    trade_quality: _grade,
    tags: _tags,
    ...fields
  }: TradeDetail) => fields;
  expect(unrelated(after)).toEqual(unrelated(before));
  const [title, columns, ...rows] = fixture.toString("utf8").trim().split("\n");
  await commit(Buffer.from([title, columns, ...rows.reverse(), ""].join("\n")), 0, 9);
  expect(await detail()).toEqual(after);
  await page.reload();
  await expect(page.getByRole("button", { name: "Backlog (2)", exact: true })).toBeVisible();
});
