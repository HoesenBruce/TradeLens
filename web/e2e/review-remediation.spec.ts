import { expect, test, request as clients } from "@playwright/test";
import type { TradeDetail } from "../src/lib/api/types";

// Browser editor A and independently authenticated HTTP editor B, disposable API only.
const api = process.env.E2E_API_URL ?? "http://127.0.0.1:8099/api/v1";
test.use({ serviceWorkers: "block" });

test("review edits preserve raw journals and untouched fields across independent editors", async ({
  page,
  request,
}, info) => {
  const credentials = { email: "review273@example.com", password: "Review273-test-pass" };
  await request.post(`${api}/setup`, { data: credentials });
  const login = await request.post(`${api}/auth/login`, { data: credentials });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const loginB = await request.post(`${api}/auth/login`, { data: credentials });
  expect(loginB.ok()).toBeTruthy();
  const { access_token: tokenB } = await loginB.json();
  const a = await clients.newContext({ extraHTTPHeaders: { Authorization: `Bearer ${token}` } });
  const b = await clients.newContext({ extraHTTPHeaders: { Authorization: `Bearer ${tokenB}` } });
  async function post(path: string, data: unknown) {
    const response = await a.post(`${api}${path}`, { data });
    expect(response.ok(), await response.text()).toBeTruthy();
    return response.json();
  }
  const account = await post("/accounts", {
    name: `Remediation ${Date.now()}`,
    broker: "manual",
    account_type: "cash",
    base_currency: "USD",
  });
  const mistake = await post("/tags", { name: `Local mistake ${Date.now()}`, kind: "mistake" });
  const remoteMistake = await post("/tags", {
    name: `Remote mistake ${Date.now()}`,
    kind: "mistake",
  });
  const custom = await post("/tags", { name: `Remote custom ${Date.now()}`, kind: "custom" });
  const close = Date.now() - 86400000;
  for (const [side, offset, price] of [
    ["buy", -3600000, 100],
    ["sell", 0, 110],
  ] as const) {
    await post("/executions", {
      account_id: account.id,
      symbol: "REMEDIATION273",
      instrument_type: "stock",
      side,
      quantity: 1,
      price,
      executed_at: new Date(close + offset).toISOString(),
    });
  }
  const list = await a.get(`${api}/trades?account_id=${account.id}`);
  expect(list.ok()).toBeTruthy();
  const [trade] = await list.json();
  const url = `${api}/trades/${trade.id}`;
  async function editB(data: unknown) {
    const response = await b.patch(url, { data });
    expect(response.ok(), await response.text()).toBeTruthy();
    return response.json();
  }
  async function read(): Promise<TradeDetail> {
    const response = await b.get(url);
    expect(response.ok()).toBeTruthy();
    return response.json();
  }
  await page.addInitScript(
    ({ token, api, account }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [account] }, version: 1 }),
      );
    },
    { token, api, account: account.id },
  );
  async function open(notes: string) {
    await editB({ notes, trade_quality: 0, tag_ids: [] });
    await page.goto("/review");
    await expect(page.getByRole("heading", { name: /REMEDIATION273/ })).toBeVisible();
  }
  async function save(expected: Record<string, unknown>) {
    const response = page.waitForResponse(
      (r) => r.url() === url && r.request().method() === "PATCH",
    );
    await page.getByRole("button", { name: "Save and next", exact: true }).click();
    const saved = await response;
    expect(saved.request().postDataJSON()).toEqual(expected);
    expect(saved.ok()).toBeTruthy();
    await expect(page.getByText(/^Queue complete/)).toBeVisible();
    return read();
  }
  try {
    // Exact byte preservation is checked after persistence, not after reparsing.
    for (const plan of ["short", "SHORT", "short\nReason: hedge existing exposure"]) {
      for (const eol of ["\n", "\r\n"]) {
        const prefix = `  legacy  \n\n## Planned direction\n${plan}\n\n## Unknown\n  keep  \n\n## Review notes  \n\n  `;
        const suffix = "  \n\n## Exit reason\nkeep exit\n\n";
        const original = (prefix + "old" + suffix).replaceAll("\n", eol);
        await open(original);
        const before = await read();
        await page.getByRole("button", { name: "A+", exact: true }).click();
        await page.getByRole("textbox", { name: "Review notes" }).fill("new");
        const notes = (prefix + "new" + suffix).replaceAll("\n", eol);
        const after = await save({ notes, trade_quality: 5 });
        expect(after.notes).toBe(notes);
        expect(after.direction).toBe("long");
        for (const key of Object.keys(before).filter(
          (k) => !["notes", "trade_quality"].includes(k),
        )) {
          expect(after[key as keyof TradeDetail], key).toEqual(before[key as keyof TradeDetail]);
        }
      }
    }
    const absent =
      "  Legacy without review\r\n\r\n## Planned direction\r\nSHORT\r\n  explanation  ";
    await open(absent);
    await page.getByRole("button", { name: "A+", exact: true }).click();
    await page.getByRole("textbox", { name: "Review notes" }).fill("appended");
    const appended = absent + "\r\n\r\n## Review notes\r\nappended";
    expect((await save({ notes: appended, trade_quality: 5 })).notes).toBe(appended);

    const plan = "## Planned direction\nSHORT\nexplanation";
    await open(plan);
    await page.getByRole("button", { name: "A+", exact: true }).click();
    await editB({
      notes: plan + "\n\n## Review notes\nremote note",
      tag_ids: [remoteMistake.id, custom.id],
    });
    const gradeOnly = await save({ trade_quality: 5 });
    expect(gradeOnly.notes).toBe(plan + "\n\n## Review notes\nremote note");
    expect(gradeOnly.tags.map((t) => t.id).sort()).toEqual(
      [remoteMistake.id, custom.id].sort((left, right) => left.localeCompare(right)),
    );

    await open(plan);
    await page.getByRole("button", { name: "A+", exact: true }).click();
    await save({ trade_quality: 5 });
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await expect(page.getByRole("button", { name: "A+", exact: true })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await page.getByRole("textbox", { name: "Review notes" }).fill("local notes only");
    await editB({
      trade_quality: 3,
      notes: plan + "\n\n## Unknown\nremote",
      tag_ids: [remoteMistake.id, custom.id],
    });
    const notesOnly = plan + "\n\n## Unknown\nremote\n\n## Review notes\nlocal notes only";
    const notesSaved = await save({ notes: notesOnly });
    expect(notesSaved.trade_quality).toBe(3);
    expect(notesSaved.tags.map((t) => t.id).sort()).toEqual(
      [remoteMistake.id, custom.id].sort((left, right) => left.localeCompare(right)),
    );

    await page.getByRole("button", { name: "Back", exact: true }).click();
    await expect(page.getByRole("button", { name: "A-", exact: true })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    await page.getByRole("checkbox", { name: mistake.name, exact: true }).check();
    await editB({ trade_quality: 4, notes: plan + "\n\n## Review notes\nremote mistake-only" });
    const mistakesOnly = await save({ tag_ids: [custom.id, remoteMistake.id, mistake.id] });
    expect(mistakesOnly.trade_quality).toBe(4);
    expect(mistakesOnly.notes).toBe(plan + "\n\n## Review notes\nremote mistake-only");

    await open(plan);
    await page.getByRole("button", { name: "A+", exact: true }).click();
    await page.getByRole("checkbox", { name: mistake.name, exact: true }).check();
    await editB({ notes: plan + "\n\n## Review notes\nremote latest", tag_ids: [custom.id] });
    let rejected = false;
    await page.route(url, async (route) => {
      if (route.request().method() === "PATCH" && !rejected) {
        rejected = true;
        await route.fulfill({
          status: 500,
          contentType: "application/json",
          body: '{"error":{"message":"synthetic failure"}}',
        });
      } else await route.continue();
    });
    await page.getByRole("button", { name: "Save and next", exact: true }).click();
    await expect(
      page.getByText("Save failed. Your draft is retained; retry or cancel."),
    ).toBeVisible();
    await expect(page.getByRole("checkbox", { name: mistake.name, exact: true })).toBeChecked();
    await editB({ notes: plan + "\n\n## Review notes\nremote after failure" });
    const retried = await save({ trade_quality: 5, tag_ids: [custom.id, mistake.id] });
    expect(retried.notes).toBe(plan + "\n\n## Review notes\nremote after failure");
    await page.unroute(url);

    const repeated =
      "## Review notes\nfirst\n\n## Review notes\nsecond\n\n## Planned direction\nSHORT";
    await open(repeated);
    await page.getByRole("button", { name: "A+", exact: true }).click();
    await page.getByRole("textbox", { name: "Review notes" }).fill("ambiguous edit");
    let writes = 0;
    page.on("request", (r) => {
      if (r.url() === url && r.method() === "PATCH") writes++;
    });
    await page.getByRole("button", { name: "Save and next", exact: true }).click();
    await expect(page.getByText(/The Review notes section is ambiguous/)).toBeVisible();
    expect(writes).toBe(0);
    expect((await read()).notes).toBe(repeated);
    await expect(page.getByRole("textbox", { name: "Review notes" })).toHaveValue("ambiguous edit");
    await page.screenshot({
      path: info.outputPath("ambiguous-journal-retained.png"),
      fullPage: true,
    });
    await page.getByRole("button", { name: "Cancel edits", exact: true }).click();
    await page.getByRole("button", { name: "A+", exact: true }).click();
    expect((await save({ trade_quality: 5 })).notes).toBe(repeated);
  } finally {
    await a.dispose();
    await b.dispose();
  }
});
