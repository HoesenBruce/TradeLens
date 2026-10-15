import { expect, test, request as clients } from "@playwright/test";
import type { TradeDetail } from "../src/lib/api/types";

const api = process.env.E2E_API_URL ?? "http://127.0.0.1:8099/api/v1";
test.use({ serviceWorkers: "block" });

test("HTML journals reject ambiguous notes before PATCH and preserve bytes through independent readback", async ({
  page,
  request,
}, info) => {
  test.setTimeout(120_000);
  const credentials = { email: "review273@example.com", password: "Review273-test-pass" };
  await request.post(`${api}/setup`, { data: credentials });
  async function login() {
    const response = await request.post(`${api}/auth/login`, { data: credentials });
    expect(response.ok()).toBeTruthy();
    return (await response.json()).access_token as string;
  }
  const token = await login();
  const a = await clients.newContext({ extraHTTPHeaders: { Authorization: `Bearer ${token}` } });
  const b = await clients.newContext({
    extraHTTPHeaders: { Authorization: `Bearer ${await login()}` },
  });
  async function post(path: string, data: unknown) {
    const response = await a.post(`${api}${path}`, { data });
    expect(response.ok(), await response.text()).toBeTruthy();
    return response.json();
  }
  const account = await post("/accounts", {
    name: `HTML boundary ${Date.now()}`,
    broker: "manual",
    account_type: "cash",
    base_currency: "USD",
  });
  const close = Date.now() - 86400000;
  for (const [side, offset, price] of [
    ["buy", -3600000, 100],
    ["sell", 0, 110],
  ] as const) {
    await post("/executions", {
      account_id: account.id,
      symbol: "HTMLBOUNDARY273",
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
  async function readB(): Promise<TradeDetail> {
    const response = await b.get(url);
    expect(response.ok()).toBeTruthy();
    return response.json();
  }
  async function open(raw: string) {
    const response = await b.patch(url, { data: { notes: raw, trade_quality: 0 } });
    expect(response.ok()).toBeTruthy();
    expect((await readB()).notes).toBe(raw);
    await page.goto("/review");
    await expect(page.getByRole("heading", { name: /HTMLBOUNDARY273/ })).toBeVisible();
    return readB();
  }
  const writes: unknown[] = [];
  page.on("request", (r) => {
    if (r.url() === url && r.method() === "PATCH") writes.push(r.postDataJSON());
  });
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
  const save = page.getByRole("button", { name: "Save and next", exact: true });
  const notes = page.getByRole("textbox", { name: "Review notes" });
  const grade = page.getByRole("button", { name: "A+", exact: true });
  const documents = [
    "<pre>\n## Review notes\nexample text\n</pre>\n\n## Unknown\nkeep",
    "<PRE>\n## Review notes\nexample\n</PRE>",
    "Legacy paragraph\n##Review notes\nnot an actual Markdown heading\n## Unknown\nkeep",
    '<PrE class="code" data-example=">">\n## Review notes\n  example  \n</pRe>',
    '<pre\n class="code"\n>\n## Review notes\nexample\n</pre>',
    "<pre>\n<pre>\n## Review notes\nexample\n</pre>\n</pre>",
    "<pre class=\n## Review notes\nexample\n</PRE",
    "<pre>\n## Review notes\nexample",
    "## Review notes\nold\n</pre>",
    "<pre>\n## Review notes\nexample\n## Review notes\nexample two\n</pre>",
    "<pre>\n## Review notes\nexample\n</pre>\n\n## Review notes\nreal\n\n## Planned direction\nSHORT",
    "<pre>\nexample only\n</pre>\n\n## Planned direction\nSHORT\nexplanation",
    "<!--\n## Review notes\nexample\n-->\n\n## Unknown\nkeep",
    '```bad`\n<PRE class="sample">\n## Review notes\nexample text\n</PRE>\n```\n## Review notes\nunrelated code sample\n## Unknown\nkeep',
    "```bad`\n<pre>\n```\n## Review notes\nexample text\n```bad`\n</pre>\n```\n## Unknown\nkeep",
    "## Review notes\nold\n\nUnknown section\n==============\nkeep exact original",
    "## Review notes\nold\n\nUnknown section\n--------------\nkeep exact original",
    "- example\n  ## Review notes\n  unrelated example",
    '[example]: /url "title\n## Review notes\nexample"',
    "~~~\n<pre>\n~~~\u00a0\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
    "## Review notes\nold\r## Unknown\rkeep",
    "```\n<pre>\n```\u00a0\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
    "```\n<pre>\n```\u000b\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
    "```\n<pre>\n```\u000c\n## Review notes\nunrelated code sample\n## Unknown\nkeep",
  ];
  try {
    for (const eol of ["\n", "\r\n"]) {
      for (const document of documents) {
        const raw = document.replaceAll("\n", eol);
        const before = await open(raw);
        writes.length = 0;
        await grade.click();
        await notes.fill("new");
        await save.click();
        try {
          await expect(page.getByText(/The Review notes section is ambiguous/)).toBeVisible();
        } catch (error) {
          const persisted = await readB();
          await info.attach("HTML-boundary-persistence-on-failure", {
            body: JSON.stringify({ raw, persisted: persisted.notes, writes }),
            contentType: "application/json",
          });
          console.log(`PERSISTED_ORIGINAL_UNCHANGED=${persisted.notes === raw}`);
          throw error;
        }
        expect(writes).toEqual([]);
        expect(await readB()).toEqual(before);
        await expect(notes).toHaveValue("new");
        await expect(grade).toHaveAttribute("aria-pressed", "true");
        if (document === documents[0] && eol === "\n") {
          await page.screenshot({
            path: info.outputPath("html-journal-rejected-draft-retained.png"),
            fullPage: true,
          });
        }
        await page.getByRole("button", { name: "Cancel edits", exact: true }).click();
        await expect(notes).toHaveValue("");
        await grade.click();
        const response = page.waitForResponse(
          (r) => r.url() === url && r.request().method() === "PATCH",
        );
        await save.click();
        expect((await response).ok()).toBeTruthy();
        await expect(page.getByText(/^Queue complete/)).toBeVisible();
        expect(writes).toEqual([{ trade_quality: 5 }]);
        const after = await readB();
        expect(after.notes).toBe(raw);
        expect(after.trade_quality).toBe(5);
        const { trade_quality: _beforeGrade, ...beforeOther } = before;
        const { trade_quality: _afterGrade, ...afterOther } = after;
        expect(afterOther).toEqual(beforeOther);
      }

      // Empty ATX headings are real section boundaries too.
      for (const marker of ["#", "   ##"]) {
        const prefix = "## Review notes" + eol;
        const suffix = eol + eol + marker + eol + "keep unrelated  " + eol;
        const before = await open(prefix + "old" + suffix);
        writes.length = 0;
        await notes.fill("new");
        const response = page.waitForResponse(
          (r) => r.url() === url && r.request().method() === "PATCH",
        );
        await save.click();
        expect((await response).ok()).toBeTruthy();
        expect(writes).toEqual([{ notes: prefix + "new" + suffix }]);
        const after = await readB();
        expect(after).toEqual({ ...before, notes: prefix + "new" + suffix });
      }

      // HTML is literal inside known closed Markdown fences; retain existing editing/append behavior.
      for (const fence of ["```", "~~~~"]) {
        const prefix =
          `Legacy  \n\n${fence}html\n<PRE class="example">\n## Review notes\nexample\n</PRE>\n${fence}\n\n## Planned direction\nSHORT\nexplanation\n\n`.replaceAll(
            "\n",
            eol,
          );
        for (const actualSection of [false, true]) {
          const suffix = eol + eol + "## Unknown" + eol + " keep  ";
          const raw = actualSection ? prefix + "## Review notes" + eol + "old" + suffix : prefix;
          await open(raw);
          writes.length = 0;
          await grade.click();
          await notes.fill("new");
          const response = page.waitForResponse(
            (r) => r.url() === url && r.request().method() === "PATCH",
          );
          await save.click();
          expect((await response).ok()).toBeTruthy();
          await expect(page.getByText(/^Queue complete/)).toBeVisible();
          const expected = prefix + "## Review notes" + eol + "new" + (actualSection ? suffix : "");
          expect(writes).toEqual([{ trade_quality: 5, notes: expected }]);
          expect((await readB()).notes).toBe(expected);
        }
      }
    }
    // The guard must inspect the latest server Journal, not only A's opening snapshot.
    await open("## Review notes\nold\n\n## Unknown\nbefore");
    writes.length = 0;
    await grade.click();
    await notes.fill("local edit retained");
    const remoteHTML = documents[0];
    const remoteWrite = await b.patch(url, { data: { notes: remoteHTML } });
    expect(remoteWrite.ok()).toBeTruthy();
    await save.click();
    await expect(page.getByText(/The Review notes section is ambiguous/)).toBeVisible();
    expect(writes).toEqual([]);
    expect((await readB()).notes).toBe(remoteHTML);
    await expect(notes).toHaveValue("local edit retained");
    await page.getByRole("button", { name: "Cancel edits", exact: true }).click();
    await grade.click();
    const gradeOnly = page.waitForResponse(
      (r) => r.url() === url && r.request().method() === "PATCH",
    );
    await save.click();
    expect((await gradeOnly).ok()).toBeTruthy();
    await expect(page.getByText(/^Queue complete/)).toBeVisible();
    expect(writes).toEqual([{ trade_quality: 5 }]);
    expect((await readB()).notes).toBe(remoteHTML);
  } finally {
    await a.dispose();
    await b.dispose();
  }
});
