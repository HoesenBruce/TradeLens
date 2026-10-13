import { createServer } from "node:http";
import { expect, test } from "@playwright/test";

test.use({ serviceWorkers: "block" });
const api = process.env.E2E_API_URL ?? "http://localhost:8098/api/v1";

test("provider error redaction through real API and Coach browser", async ({
  page,
  request,
}, info) => {
  const password = "Synthetic295-acceptance-password";
  const setup = await request.post(`${api}/setup`, {
    data: { email: "owner295@example.invalid", password },
  });
  const owner = setup.ok()
    ? await setup.json()
    : await (
        await request.post(`${api}/auth/login`, {
          data: { email: "owner295@example.invalid", password },
        })
      ).json();
  const memberEmail = `member295-${Date.now()}@example.invalid`;
  expect(
    (await request.post(`${api}/auth/register`, { data: { email: memberEmail, password } })).ok(),
  ).toBeTruthy();
  const member = await (
    await request.post(`${api}/auth/login`, { data: { email: memberEmail, password } })
  ).json();
  const ownerHeaders = { Authorization: `Bearer ${owner.access_token}` };
  const key = "synthetic-owner-key-295";
  const prompt = "synthetic-private-prompt-295";
  let calls = 0;
  let fail = true;
  let finishSuccess = () => {};
  const provider = createServer(async (req, res) => {
    let body = "";
    for await (const chunk of req) body += chunk.toString();
    const payload = JSON.parse(body) as { stream?: boolean; messages: { content: string }[] };
    calls++;
    expect(req.headers.authorization).toBe(`Bearer ${key}`);
    expect(payload.messages.some((message) => message.content.includes(prompt))).toBeTruthy();
    expect(payload.messages.every((message) => !message.content.includes(key))).toBeTruthy();
    if (fail) {
      const content = JSON.stringify({
        notes: [
          {
            tone: "warn",
            headline: "Provider rejection",
            detail: `${key}; Authorization: Bearer ${key}; ${prompt}`,
          },
        ],
      });
      if (payload.stream) {
        res.writeHead(200, { "Content-Type": "text/event-stream" });
        // Split the secret across actual provider chunks, then fail after a complete note.
        for (let i = 0; i < content.length; i += 5)
          res.write(
            `data: ${JSON.stringify({ choices: [{ delta: { content: content.slice(i, i + 5) } }] })}\n\n`,
          );
        res.end(`data: ${JSON.stringify({ error: { message: key } })}\n\n`);
      } else {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ choices: [{ message: { content } }] }));
      }
    } else {
      res.writeHead(200, { "Content-Type": "text/event-stream" });
      const content = JSON.stringify({
        notes: [{ tone: "tip", headline: "Safe review", detail: "Synthetic successful review" }],
      });
      res.write(
        `data: ${JSON.stringify({ choices: [{ delta: { content: content.slice(0, -1) + "," } }] })}\n\n`,
      );
      await new Promise<void>((resolve) => {
        finishSuccess = resolve;
      });
      res.end(
        `data: ${JSON.stringify({ choices: [{ delta: { content: '"next_action":"Record the next plan."}' } }] })}\n\ndata: [DONE]\n\n`,
      );
    }
  });
  await new Promise<void>((resolve) => provider.listen(0, "127.0.0.1", resolve));
  const { port } = provider.address() as { port: number };
  try {
    expect(
      (
        await request.put(`${api}/settings/coach`, {
          headers: ownerHeaders,
          data: {
            enabled: true,
            base_url: `http://127.0.0.1:${port}`,
            api_key: key,
            model: "fixture",
            custom_prompt: prompt,
          },
        })
      ).ok(),
    ).toBeTruthy();
    async function trade(token: string) {
      const headers = { Authorization: `Bearer ${token}` };
      const account = await (
        await request.post(`${api}/accounts`, {
          headers,
          data: {
            name: "Synthetic295",
            broker: "manual",
            account_type: "cash",
            base_currency: "USD",
          },
        })
      ).json();
      let id = "";
      for (const [side, price, offset] of [
        ["buy", 100, -3600000],
        ["sell", 110, 0],
      ] as const) {
        const r = await request.post(`${api}/executions`, {
          headers,
          data: {
            account_id: account.id,
            symbol: "SAFE295",
            side,
            quantity: 1,
            price,
            executed_at: new Date(Date.now() + offset).toISOString(),
          },
        });
        expect(r.ok()).toBeTruthy();
        id = (await r.json()).trade_id;
      }
      return id;
    }
    const memberTrade = await trade(member.access_token);
    const ownerTrade = await trade(owner.access_token);
    const pat = await (
      await request.post(`${api}/access-tokens`, {
        headers: { Authorization: `Bearer ${member.access_token}` },
        data: { name: "Synthetic295" },
      })
    ).json();
    for (const token of [member.access_token, pat.token]) {
      const headers = { Authorization: `Bearer ${token}` };
      expect((await request.get(`${api}/settings/coach`, { headers })).status()).toBe(403);
      for (const suffix of ["coach", "coach/stream"]) {
        const response = await request.post(`${api}/trades/${memberTrade}/${suffix}`, {
          headers,
          data: {},
        });
        expect(response.status()).toBe(200);
        const body = await response.text();
        expect(body).toContain("coach generation failed");
        for (const secret of [key, prompt, "Authorization: Bearer"])
          expect(body).not.toContain(secret);
      }
      expect(
        (
          await (
            await request.get(`${api}/trades/${memberTrade}/coach/reviews`, { headers })
          ).json()
        ).reviews,
      ).toEqual([]);
    }
    const errors: string[] = [];
    const consoleErrors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    await page.addInitScript(
      ({ token, api }) => {
        Reflect.deleteProperty(Navigator.prototype, "serviceWorker");
        localStorage.setItem("tm_token", token);
        localStorage.setItem("tm_api_base", api);
        localStorage.setItem("tm-locale", "en");
      },
      { token: owner.access_token, api },
    );
    await page.goto(`/trades/${ownerTrade}`);
    const ask = page.getByRole("button", { name: "Ask AI coach" });
    await expect(ask).toBeVisible();
    await ask.click();
    await expect(page.getByText("Could not reach the coach API", { exact: true })).toBeVisible();
    for (const secret of [key, prompt, "Authorization: Bearer"])
      expect(
        (await page.locator("body").innerText()) + errors.join() + consoleErrors.join(),
      ).not.toContain(secret);
    await page.screenshot({ path: info.outputPath("coach-error.png"), fullPage: true });
    fail = false;
    await page.getByRole("button", { name: "Regenerate AI coach" }).click();
    await expect(page.getByText("Tip · Safe review", { exact: true })).toBeVisible();
    await expect(page.getByText("Synthetic successful review", { exact: true })).toBeVisible();
    await expect(
      page.getByRole("button", { name: /^(Ask AI coach|Regenerate AI coach)$/ }),
    ).toBeDisabled();
    await page.screenshot({ path: info.outputPath("coach-incremental.png"), fullPage: true });
    finishSuccess();
    await expect(page.getByRole("button", { name: "Regenerate AI coach" })).toBeEnabled();
    await page.screenshot({ path: info.outputPath("coach-success.png"), fullPage: true });
    for (const secret of [key, prompt, "Authorization: Bearer"])
      expect(
        (await page.locator("body").innerText()) + errors.join() + consoleErrors.join(),
      ).not.toContain(secret);
    // The inherited mutation handler can emit this sanitized rejected-promise error.
    expect(errors.filter((error) => error !== "coach generation failed")).toEqual([]);
    await info.attach("browser-errors", {
      body: JSON.stringify({ errors, consoleErrors }),
      contentType: "application/json",
    });
    expect(calls).toBeGreaterThanOrEqual(6);
  } finally {
    finishSuccess();
    await new Promise<void>((resolve, reject) =>
      provider.close((err) => (err ? reject(err) : resolve())),
    );
  }
});
