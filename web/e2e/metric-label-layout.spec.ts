import { expect, test, type Locator, type Page } from "@playwright/test";
import { seedStatCards } from "./fixtures/seed-stat-cards";

const api = process.env.E2E_API_URL ?? "http://localhost:8097/api/v1";
test.use({ serviceWorkers: "block", hasTouch: true });

async function attachedLabels(root: Locator) {
  const groups = root
    .locator("[data-metric-label]")
    .filter({ has: root.page().locator("[data-metric-help]") });
  expect(await groups.count()).toBeGreaterThan(0);
  const problems = await groups.evaluateAll((elements) =>
    elements.flatMap((el) => {
      if (!el.getClientRects().length) return [];
      const text = el.firstElementChild!;
      const trigger = el.querySelector("[data-metric-help]")!;
      const a = text.getBoundingClientRect();
      const b = trigger.getBoundingClientRect();
      const group = el.getBoundingClientRect();
      const scale = b.width / 28;
      const parentWidth = el.parentElement!.getBoundingClientRect().width;
      return (b.left - a.right) / scale < 3 ||
        (b.left - a.right) / scale > 5 ||
        b.right > group.right + 1 ||
        (parentWidth > 0 && group.width > parentWidth + 1) ||
        el.scrollWidth > el.clientWidth + 1 ||
        b.width < 28 ||
        b.height < 28
        ? [
            {
              label: text.textContent,
              gap: b.left - a.right,
              width: group.width,
              parent: parentWidth,
            },
          ]
        : [];
    }),
  );
  expect(problems).toEqual([]);
}

async function exercise(page: Page, trigger: Locator) {
  await trigger.press("Enter");
  const dialog = page.locator('[data-slot="popover-popup"][data-open]');
  await expect(dialog).toBeVisible();
  await expect(dialog).toHaveAttribute("aria-labelledby", /.+/);
  await page.keyboard.press("Escape");
  await expect(dialog).not.toBeVisible();
  await expect(trigger).toBeFocused();
  await trigger.press("Space");
  await expect(dialog).toBeVisible();
  await dialog.locator('[data-slot="popover-close"]').click();
  await expect(trigger).toBeFocused();
  await trigger.tap();
  await expect(dialog).toBeVisible();
  const bounds = await dialog.boundingBox();
  expect(bounds!.x).toBeGreaterThanOrEqual(0);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(page.viewportSize()!.width);
  await dialog.locator('[data-slot="popover-close"]').click();
  await expect(dialog).not.toBeVisible();
}

test("short and long labels stay attached across locales, narrow widths and 200% zoom", async ({
  page,
}, info) => {
  test.setTimeout(90_000);
  await page.goto("/e2e/fixtures/metric-help.html");
  for (const locale of ["en", "zh-CN", "zh-HK", "ja", "ko"]) {
    await page.getByRole("combobox", { name: "Locale", exact: true }).selectOption(locale);
    for (const width of [1440, 390, 320]) {
      await page.setViewportSize({ width, height: 900 });
      await attachedLabels(page.locator("main"));
      await expect(page.getByTestId("no-help").locator("[data-metric-help]")).toHaveCount(0);
      await exercise(page, page.getByTestId("long-label").locator("[data-metric-help]"));
      await page.screenshot({ path: info.outputPath(`${locale}-${width}.png`) });
    }
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.evaluate(() => (document.documentElement.style.zoom = "2"));
  await attachedLabels(page.locator("main"));
  await page.screenshot({ path: info.outputPath("zoom-200.png") });
});

test("real API dashboard, table, compact list, trade page/drawer and reports use attached help", async ({
  page,
  request,
}, info) => {
  test.setTimeout(180_000);
  await seedStatCards(request, api);
  const login = await request.post(`${api}/auth/login`, {
    data: { email: "qa333@synthetic.invalid", password: "synthetic-333-password" },
  });
  expect(login.ok()).toBeTruthy();
  const { access_token: token } = await login.json();
  const headers = { Authorization: `Bearer ${token}` };
  const accounts = await (await request.get(`${api}/accounts`, { headers })).json();
  const account = accounts.find((a: { name: string }) => a.name === "Synthetic JPY card QA");
  const trades = await (
    await request.get(`${api}/trades?account_id=${account.id}`, { headers })
  ).json();
  const before = JSON.stringify(trades);
  await page.addInitScript(
    ({ token, api, id }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      localStorage.setItem("tm-locale", "en");
      localStorage.setItem(
        "tm_filters",
        JSON.stringify({ state: { accountIds: [id] }, version: 1 }),
      );
    },
    { token, api, id: account.id },
  );
  for (const path of [
    "/home",
    "/trades",
    `/accounts/${account.id}`,
    `/trades/${trades[0].id}`,
    "/reports",
  ]) {
    await page.goto(path);
    await expect(page.locator("main [data-metric-help]").first()).toBeVisible();
    for (const width of [1440, 390, 320]) {
      await page.setViewportSize({ width, height: 1000 });
      await attachedLabels(page.locator("body"));
      await exercise(page, page.locator("main [data-metric-help]").first());
      await page.screenshot({
        path: info.outputPath(
          `${path.startsWith("/trades/") ? "trade-detail" : path.split("/")[1]}-${width}.png`,
        ),
      });
    }
  }
  await page.goto("/trades");
  await page.setViewportSize({ width: 1440, height: 1000 });
  const sorted = await page.locator("th[aria-sort]").allTextContents();
  await exercise(page, page.locator("th [data-metric-help]").first());
  expect(await page.locator("th[aria-sort]").allTextContents()).toEqual(sorted);
  await page.setViewportSize({ width: 390, height: 1000 });
  await page.getByText("SYNTH12", { exact: true }).first().click();
  const drawer = page.getByRole("dialog");
  await expect(drawer).toBeVisible();
  await attachedLabels(drawer);
  await exercise(page, drawer.locator("[data-metric-help]").first());
  await page.screenshot({ path: info.outputPath("drawer-390.png") });
  await page.keyboard.press("Escape");
  await expect(drawer).not.toBeVisible();
  expect(
    JSON.stringify(
      await (await request.get(`${api}/trades?account_id=${account.id}`, { headers })).json(),
    ),
  ).toBe(before);
});
