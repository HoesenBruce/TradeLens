import { expect, test, type Page, type APIRequestContext, type Locator } from "@playwright/test";

import { seedStatCards } from "./fixtures/seed-stat-cards";

// Run against a disposable API; this suite seeds its own synthetic records.
const api = process.env.E2E_API_URL ?? "http://localhost:8094/api/v1";
test.use({ serviceWorkers: "block", actionTimeout: 8000 });
test.beforeAll(async ({ request }) => {
  await seedStatCards(request, api);
});

async function login(page: Page, request: APIRequestContext) {
  const r = await request.post(`${api}/auth/login`, {
    data: { email: "qa333@synthetic.invalid", password: "synthetic-333-password" },
  });
  expect(r.ok()).toBeTruthy();
  const { access_token } = await r.json();
  const accounts = await (
    await request.get(`${api}/accounts`, { headers: { Authorization: `Bearer ${access_token}` } })
  ).json();
  const accountId = accounts.find((a: { name: string }) => a.name === "Synthetic JPY card QA").id;
  await page.addInitScript(
    ({ token, api, accountId }) => {
      localStorage.setItem("tm_token", token);
      localStorage.setItem("tm_api_base", api);
      if (!localStorage.getItem("tm_filters"))
        localStorage.setItem(
          "tm_filters",
          JSON.stringify({ state: { accountIds: [accountId] }, version: 1 }),
        );
      if (!localStorage.getItem("tm-locale")) localStorage.setItem("tm-locale", "en");
    },
    { token: access_token, api, accountId },
  );
  return access_token;
}

async function assertGeometry(page: Page, scope: Locator = page.getByTestId("surface")) {
  const failures = await scope.evaluate((root) => {
    const failures: string[] = [];
    for (const el of root.querySelectorAll<HTMLElement>("section,p,h1,h2,h3,div")) {
      if (el.closest("svg,table,[aria-hidden=true]")) continue;
      let scrollParent: Element | null = el.parentElement;
      while (
        scrollParent &&
        scrollParent !== root &&
        !["auto", "scroll"].includes(getComputedStyle(scrollParent).overflowX)
      )
        scrollParent = scrollParent.parentElement;
      if (scrollParent && scrollParent !== root) continue;
      if (
        el.tagName === "SECTION" &&
        el.scrollWidth <= el.clientWidth + 8 &&
        [...el.children].some((c) => getComputedStyle(c).overflowX === "auto")
      )
        continue;
      const style = getComputedStyle(el);
      if (style.visibility === "hidden" || el.clientWidth === 0 || style.overflowX !== "visible")
        continue;
      if (el.scrollWidth > el.clientWidth + 4)
        failures.push(
          `${el.tagName}: ${el.textContent?.slice(0, 100)} (${el.scrollWidth}/${el.clientWidth})`,
        );
    }
    for (const el of root.querySelectorAll<HTMLElement>("[data-money-format]")) {
      const b = el.getBoundingClientRect();
      const localScroll = el.closest("div.overflow-x-auto");
      const card = localScroll ? undefined : el.closest("section")?.getBoundingClientRect();
      if (b.width && card && (b.left < card.left - 1 || b.right > card.right + 1))
        failures.push(`money outside card: ${el.textContent}`);
      if (b.width && el.dataset.moneyFormat === "compact" && !el.getAttribute("aria-label"))
        failures.push("missing full value");
      if (
        el.querySelector("[tabindex='0']") ||
        (el.closest("button") && el.getAttribute("tabindex") === "0")
      )
        failures.push("nested control");
    }
    for (const grid of root.querySelectorAll<HTMLElement>("div.grid")) {
      const cells = [...grid.children].filter((el) => el.tagName === "SECTION");
      for (let i = 0; i < cells.length; i++)
        for (let j = i + 1; j < cells.length; j++) {
          const a = cells[i].getBoundingClientRect(),
            b = cells[j].getBoundingClientRect();
          if (
            Math.min(a.right, b.right) - Math.max(a.left, b.left) > 1 &&
            Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top) > 1
          )
            failures.push("overlapping cards");
        }
    }
    if (document.documentElement.scrollWidth > innerWidth)
      failures.push("page horizontal overflow");
    return failures;
  });
  expect(failures, `${page.url()} / ${page.viewportSize()?.width}`).toEqual([]);
}

test("all card families stay inside their tracks across locales and breakpoints", async ({
  page,
  request,
}, info) => {
  test.setTimeout(240_000);
  await login(page, request);
  await page.goto("/e2e/fixtures/stat-cards.html");
  await expect(page.getByTestId("api-proof")).toContainText("16 trades");
  for (const locale of ["en", "zh-CN", "zh-HK", "ja", "ko"]) {
    await page.getByLabel("Locale", { exact: true }).selectOption(locale);
    for (const group of ["home", "reports", "callers", "goals", "trade", "primitives", "export"]) {
      await page.getByLabel("Surface", { exact: true }).selectOption(group);
      for (const width of [320, 390, 639, 640, 1023, 1024, 1279, 1280, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        await expect
          .poll(() => page.getByTestId("surface").locator("[data-money-format]").count())
          .toBeGreaterThanOrEqual(0);
        await page.evaluate(
          () =>
            new Promise<void>((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
            ),
        );
        await assertGeometry(page);
      }
      if (locale === "en") {
        await page.setViewportSize({ width: 390, height: 900 });
        await page.getByTestId("surface").screenshot({ path: info.outputPath(`${group}.png`) });
      }
      if (group === "export") {
        const text = await page.locator("svg text").evaluateAll((elements) =>
          elements.map((el) => ({
            x: (el as SVGGraphicsElement).getBBox().x,
            w: (el as SVGGraphicsElement).getBBox().width,
            size: Number(el.getAttribute("font-size")),
          })),
        );
        expect(text.every((t) => t.x >= 0 && t.x + t.w <= 1200 && t.size >= 16)).toBeTruthy();
      }
    }
  }
});

test("boundary changes, full disclosure, masking and Reports display transitions", async ({
  page,
  request,
}) => {
  await login(page, request);
  await page.goto("/e2e/fixtures/stat-cards.html");
  await expect(page.getByTestId("api-proof")).toBeVisible();
  await page.getByLabel("Surface", { exact: true }).selectOption("primitives");
  await page.setViewportSize({ width: 320, height: 900 });
  for (const currency of ["JPY", "USD"]) {
    await page.getByLabel("Currency", { exact: true }).selectOption(currency);
    for (const amount of [0, 10000, -10000, 999949, 999950, 1000000, -9876543, 1e25, 10]) {
      await page.getByLabel("Amount", { exact: true }).fill(String(amount));
      await assertGeometry(page);
    }
  }
  await page.getByLabel("Amount", { exact: true }).fill("12345678");
  const value = page.getByTestId("boundary-money").locator("[data-money-format]");
  await expect(value).toHaveAttribute("data-money-format", "compact");
  const full = await value.getAttribute("aria-label");
  await value.hover();
  const popup = page.getByRole("tooltip");
  await expect(popup).toContainText(full!);
  await page.keyboard.press("Escape");
  await expect(popup).not.toBeVisible();
  await value.focus();
  await expect(popup).toContainText(full!);
  await page.keyboard.press("Escape");
  await value.click();
  await expect(popup).toContainText(full!);
  await page.getByRole("button", { name: "Privacy off", exact: true }).click();
  await expect(page.locator("body")).not.toContainText(full!);
  expect(await page.locator("body").innerHTML()).not.toContain(full!);
  await page.getByRole("button", { name: "Privacy on", exact: true }).click();
  await page.keyboard.press("Escape");
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(value).toHaveAttribute("data-money-format", "full");
  await page.setViewportSize({ width: 320, height: 900 });
  await expect(value).toHaveAttribute("data-money-format", "compact");
  await page.getByLabel("Surface", { exact: true }).selectOption("reports");
  const reportAmount = page.getByTestId("surface").locator("[data-money-format]").first();
  await expect(reportAmount).toContainText("+$");
  await page.getByLabel("Locale", { exact: true }).selectOption("zh-CN");
  await expect(reportAmount).toContainText("+US$");
  await page.getByLabel("Locale", { exact: true }).selectOption("en");
  await expect(reportAmount).toContainText("+$");
  const before = await page.getByTestId("surface").innerText();
  for (const name of ["P&L net", "Units abs", "Average mean"]) {
    await page.getByRole("button", { name, exact: true }).click();
    await assertGeometry(page);
  }
  expect(await page.getByTestId("surface").innerText()).not.toBe(before);
  for (const name of ["P&L gross", "Units pct", "Average median"])
    await page.getByRole("button", { name, exact: true }).click();
  await expect.poll(() => page.getByTestId("surface").innerText()).toBe(before);
});

test("real account, trade, day, recap, public pages and PNG export", async ({
  page,
  request,
}, info) => {
  test.setTimeout(180_000);
  const token = await login(page, request);
  const headers = { Authorization: `Bearer ${token}` };
  const accounts = await (await request.get(`${api}/accounts`, { headers })).json();
  const account = accounts.find((a: { name: string }) => a.name === "Synthetic JPY card QA");
  const trades = await (
    await request.get(`${api}/trades?account_id=${account.id}`, { headers })
  ).json();
  const trade = trades.find((t: { symbol: string }) => t.symbol.startsWith("SYNTHETIC_LONG"));
  const links = [];
  for (const show_amounts of [true, false]) {
    const response = await request.post(`${api}/share-links`, {
      headers,
      data: { account_id: account.id, show_amounts, currency: "JPY", expires_in_days: 1 },
    });
    expect(response.ok()).toBeTruthy();
    links.push(await response.json());
  }
  await page.goto("/home");
  for (const locale of ["en", "zh-CN", "zh-HK", "ja", "ko"]) {
    await page.evaluate((locale) => localStorage.setItem("tm-locale", locale), locale);
    for (const path of [
      `/accounts/${account.id}`,
      `/trades/${trade.id}`,
      "/day/2026-10-01",
      "/reports",
      "/wrapped?year=2026",
      `/s/${links[0].token}`,
      `/s/${links[1].token}`,
      "/news/performance",
    ]) {
      await page.goto(path);
      const root = page.locator("main").first();
      await expect(root).toBeVisible();
      if (path.includes("/s/") && path.includes(links[1].token)) {
        await expect.poll(() => root.locator("section").count()).toBeGreaterThanOrEqual(5);
        await expect(root.locator("[data-money-format]")).toHaveCount(0);
      } else if (!path.includes("/news/"))
        await expect(root.locator("[data-money-format]").first()).toBeVisible();
      for (const width of [320, 390, 640, 1024, 1280, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        await page.evaluate(
          () =>
            new Promise<void>((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
            ),
        );
        await assertGeometry(page, root);
      }
    }
  }
  await page.evaluate(() => localStorage.setItem("tm-locale", "en"));
  await page.goto("/wrapped?year=2025");
  await expect(page.getByText("No closed trades in 2025", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Share card", exact: true })).toBeDisabled();
  await page.goto("/trades");
  await page.setViewportSize({ width: 390, height: 900 });
  await page.getByText("SYNTH12", { exact: true }).first().click();
  const drawer = page.getByRole("dialog");
  await expect(drawer).toBeVisible();
  for (const width of [320, 390, 640, 1024]) {
    await page.setViewportSize({ width, height: 900 });
    await assertGeometry(page, drawer);
  }
  await page.getByRole("button", { name: "Open full page", exact: true }).click();
  await expect(page).toHaveURL(
    new RegExp(`/trades/${trades.find((t: { symbol: string }) => t.symbol === "SYNTH12").id}`),
  );
  await page.getByRole("button", { name: "Edit trade", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.setViewportSize({ width: 390, height: 900 });
  await assertGeometry(page, page.getByRole("dialog"));
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).not.toBeVisible();
  await page.goto("/wrapped?year=2026");
  await expect(page.getByRole("button", { name: "Share card", exact: true })).toBeEnabled();
  await page.getByRole("button", { name: "Share card", exact: true }).click();
  await expect(
    page.getByRole("img", { name: "2026 Wrapped share card", exact: true }),
  ).toBeVisible();
  await page.getByRole("switch").click();
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: "Download PNG", exact: true }).click(),
  ]);
  await download.saveAs(info.outputPath("synthetic-wrapped.png"));
  const svg = page.getByRole("img", { name: "2026 Wrapped share card", exact: true });
  expect(
    await svg.locator("text").evaluateAll((elements) =>
      elements.every((el) => {
        const b = (el as SVGGraphicsElement).getBBox();
        return b.x >= 0 && b.x + b.width <= 1200;
      }),
    ),
  ).toBeTruthy();
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Share card", exact: true }).click();
  await expect(page.getByRole("switch")).toBeChecked();
  await page.keyboard.press("Escape");
});

test("goal states and editor transitions, help, filters and calculator results", async ({
  page,
  request,
}) => {
  test.setTimeout(120_000);
  await login(page, request);
  await page.goto("/e2e/fixtures/stat-cards.html");
  await expect(page.getByTestId("api-proof")).toBeVisible();
  await page.getByLabel("Surface", { exact: true }).selectOption("goals");
  await page.setViewportSize({ width: 320, height: 900 });
  for (const amount of [0, -9000000, -10000000, -12000000, 12345678, 1e25, 0]) {
    await page.getByLabel("Amount", { exact: true }).fill(String(amount));
    await assertGeometry(page);
  }
  const editor = page.getByRole("button", { name: "Edit annual goal", exact: true }).first();
  const input = page.getByRole("textbox", { name: "Annual P&L goal amount", exact: true });
  await editor.click();
  await input.fill("20000000");
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await editor.click();
  await expect(input).toHaveValue("10000000");
  await input.fill("20000000");
  await page.getByRole("button", { name: "Save goal", exact: true }).click();
  await editor.click();
  await expect(input).toHaveValue("20000000");
  await page.getByRole("button", { name: "Clear", exact: true }).click();
  await expect(page.getByRole("button", { name: "Set annual goal", exact: true })).toHaveCount(2);
  await page.getByRole("button", { name: "Set annual goal", exact: true }).first().click();
  await input.fill("10000000");
  await page.getByRole("button", { name: "Save goal", exact: true }).click();
  await assertGeometry(page);
  await page.goto("/home");
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(page.getByTestId("home-insights")).toBeVisible();
  await page.locator('[data-metric-help="profitFactor"]').last().press("Enter");
  await expect(page.locator('[data-slot="popover-popup"][data-open]')).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.locator('[data-slot="popover-popup"][data-open]')).toHaveCount(0);
  const winners = page.getByRole("button", { name: /^Wins/ });
  await winners.click();
  await expect(winners).toHaveAttribute("aria-pressed", "true");
  await page.getByRole("link", { name: "Trades", exact: true }).first().click();
  await page.getByRole("link", { name: "Home", exact: true }).first().click();
  await expect(winners).toHaveAttribute("aria-pressed", "true");
  await winners.click();
  await expect(winners).toHaveAttribute("aria-pressed", "false");
  await page.goto("/calculator");
  await page.getByRole("textbox", { name: "Capital", exact: true }).fill("1234567890123");
  await page.getByRole("textbox", { name: "Entry", exact: true }).fill("100");
  await page.getByRole("textbox", { name: "Stop", exact: true }).fill("99.99");
  for (const width of [320, 390, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await assertGeometry(page, page.locator("main"));
  }
  await page.setViewportSize({ width: 1440, height: 900 });
  for (const tool of ["Position size", "Kelly criterion"]) {
    await page.getByRole("button", { name: "Tools", exact: true }).click();
    await page.getByRole("button", { name: tool, exact: true }).click();
    const modal = page.getByRole("dialog");
    await expect(modal).toBeVisible();
    if (tool === "Position size") {
      await modal.locator("#ps-entry").fill("100");
      await modal.locator("#ps-stop").fill("99.999999");
      await modal.locator("#ps-equity").fill("1234567890123");
    } else await modal.locator("#kelly-payoff").fill("1234567890123");
    await page.setViewportSize({ width: 320, height: 900 });
    await assertGeometry(page, modal);
    await page.keyboard.press("Escape");
    await expect(modal).not.toBeVisible();
    await page.setViewportSize({ width: 1440, height: 900 });
  }
});

test("touch can reveal a compact amount and never exposes it after masking", async ({
  browser,
  request,
}) => {
  const context = await browser.newContext({
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
    serviceWorkers: "block",
  });
  const page = await context.newPage();
  try {
    await login(page, request);
    await page.goto(
      `${process.env.E2E_BASE_URL ?? "http://localhost:5173"}/e2e/fixtures/stat-cards.html`,
    );
    await expect(page.getByTestId("api-proof")).toBeVisible();
    await page.getByLabel("Surface", { exact: true }).selectOption("primitives");
    const amount = page.getByTestId("boundary-money").locator("[data-money-format]");
    await expect(amount).toHaveAttribute("data-money-format", "compact");
    const full = await amount.getAttribute("aria-label");
    await amount.tap();
    await expect(page.getByRole("tooltip")).toContainText(full!);
    await page.getByRole("button", { name: "Privacy off", exact: true }).tap();
    expect(await page.locator("body").innerHTML()).not.toContain(full!);
    await page.getByRole("button", { name: "Privacy on", exact: true }).tap();
    await amount.tap();
    await expect(page.getByRole("tooltip")).toContainText(full!);
  } finally {
    await context.close();
  }
});

test("native Reports display modes, card layout read-back/reset and day drawer", async ({
  page,
  request,
}) => {
  test.setTimeout(90_000);
  await login(page, request);
  await page.goto("/reports");
  await page.setViewportSize({ width: 1440, height: 900 });
  await expect(page.locator("main [data-money-format]").first()).toBeVisible();
  for (const [group, name] of [
    ["P&L basis", "Gross"],
    ["Unit", "%"],
    ["Average basis", "Median"],
  ])
    await page
      .getByRole("group", { name: group, exact: true })
      .getByRole("button", { name, exact: true })
      .click();
  await expect(page).toHaveURL(/pnl=gross.*unit=pct.*avg=median/);
  await page.reload();
  for (const [group, name] of [
    ["P&L basis", "Gross"],
    ["Unit", "%"],
    ["Average basis", "Median"],
  ])
    await expect(
      page
        .getByRole("group", { name: group, exact: true })
        .getByRole("button", { name, exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
  for (const [group, name] of [
    ["P&L basis", "Net"],
    ["Unit", "$"],
    ["Average basis", "Mean"],
  ])
    await page
      .getByRole("group", { name: group, exact: true })
      .getByRole("button", { name, exact: true })
      .click();
  const customize = page.getByRole("button", { name: "Customize cards", exact: true });
  await customize.click();
  const popup = page.locator('[data-slot="popover-popup"][data-open]');
  const first = popup.locator("button[aria-pressed]").first();
  const name = (await first.innerText()).trim();
  const sections = await page.locator("main section").count();
  await first.click();
  await expect(popup.getByRole("button", { name, exact: true })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await page.keyboard.press("Escape");
  await expect.poll(() => page.locator("main section").count()).toBeLessThan(sections);
  await customize.click();
  await expect(popup.getByRole("button", { name, exact: true })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await popup.getByRole("button", { name, exact: true }).click();
  await popup.getByRole("button", { name: `Move ${name} up`, exact: true }).click();
  await page.keyboard.press("Escape");
  await page.setViewportSize({ width: 390, height: 900 });
  await assertGeometry(page, page.locator("main"));
  await customize.click();
  await popup.getByRole("button", { name: "Reset to default", exact: true }).click();
  await page.keyboard.press("Escape");
  await expect.poll(() => page.locator("main section").count()).toBe(sections);
  await page
    .getByRole("button", { name: /^View trades for/ })
    .nth(1)
    .click();
  const drawer = page.getByRole("dialog");
  await expect(drawer).toBeVisible();
  await assertGeometry(page, drawer);
  await expect(drawer.locator("[data-money-format]").first()).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(drawer).not.toBeVisible();
});

test("heatmap detail amounts and trade actions remain accessible in a narrow popup", async ({
  page,
  request,
}) => {
  await login(page, request);
  await page.goto("/e2e/fixtures/stat-cards.html");
  await expect(page.getByTestId("api-proof")).toBeVisible();
  await page.getByLabel("Surface", { exact: true }).selectOption("reports");
  for (const locale of ["en", "zh-CN", "zh-HK", "ja", "ko"]) {
    await page.getByLabel("Locale", { exact: true }).selectOption(locale);
    await page.getByTestId("heatmap").locator("button.cursor-pointer[aria-label]").first().click();
    const popup = page.locator('[data-slot="popover-popup"][data-open]');
    await expect(popup).toBeVisible();
    for (const width of [320, 390, 640]) {
      await page.setViewportSize({ width, height: 900 });
      await page.evaluate(
        () =>
          new Promise<void>((resolve) =>
            requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
          ),
      );
      await expect
        .poll(async () => {
          const box = await popup.boundingBox();
          return box != null && box.x >= 0 && box.x + box.width <= width;
        })
        .toBeTruthy();
      await expect
        .poll(() => page.evaluate(() => document.documentElement.scrollWidth))
        .toBeLessThanOrEqual(width);
      await assertGeometry(page, popup);
    }
    expect(await popup.locator('button [tabindex="0"]').count()).toBe(0);
    await page.keyboard.press("Escape");
    await expect(popup).not.toBeVisible();
  }
});
