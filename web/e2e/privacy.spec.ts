import { expect, test, type Page } from "@playwright/test";

// Use E2E_BASE_URL for a served production build, backed by a disposable seeded API.
// The account must have closed trades and funding in at least two currencies.
test.skip(!process.env.E2E_EMAIL || !process.env.E2E_PASSWORD, "Requires seeded E2E credentials");

async function moneyState(page: Page) {
  return page.evaluate(() => {
    const text = document.body.innerText;
    // Recharts measures ticks in an off-screen span; it is not displayed money.
    const measure = document.getElementById("recharts_measurement_span")?.innerText ?? "";
    const count = (s: string, re: RegExp) => (s.match(re) ?? []).length;
    return {
      amounts: count(text, /[$¥￥€£]\s*\d/g) - count(measure, /[$¥￥€£]\s*\d/g),
      masks: count(text, /••••/g) - count(measure, /••••/g),
    };
  });
}

async function flipPrivacy(page: Page) {
  await expect.poll(async () => (await moneyState(page)).amounts).toBeGreaterThan(0);
  await page.getByRole("button", { name: "Hide sensitive amounts", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Show sensitive amounts", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await expect.poll(async () => (await moneyState(page)).amounts).toBe(0);
  await expect.poll(async () => (await moneyState(page)).masks).toBeGreaterThan(0);
  await page.getByRole("button", { name: "Show sensitive amounts", exact: true }).click();
  await expect.poll(async () => (await moneyState(page)).amounts).toBeGreaterThan(0);
}

test("live privacy flip remasks compiled money surfaces and survives navigation", async ({
  page,
}) => {
  await page.goto("/home");
  await page.locator("#username").fill(process.env.E2E_EMAIL!);
  await page.locator("#password").fill(process.env.E2E_PASSWORD!);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Historical account value", exact: true }),
  ).toBeVisible();
  const dismiss = page.getByRole("button", { name: "Dismiss", exact: true });
  if (await dismiss.isVisible()) await dismiss.click();
  await flipPrivacy(page);

  await page.getByRole("link", { name: "Reports", exact: true }).click();
  for (const name of ["Overview", "Win / Loss", "Detailed", "Risk", "Behavior"]) {
    await page.getByRole("tab", { name, exact: true }).click();
    await expect(page.getByRole("tabpanel", { name, exact: true })).toBeVisible();
    await flipPrivacy(page);
  }
  await page.getByRole("link", { name: "Trades", exact: true }).click();
  await expect(page.getByRole("button", { name: "Add filter", exact: true })).toBeVisible();
  await flipPrivacy(page);
  await page.locator("tbody tr").first().click();
  await page.getByRole("button", { name: "Open full page", exact: true }).click();
  await expect(page.getByRole("button", { name: "Back to trades", exact: true })).toBeVisible();
  await flipPrivacy(page);

  await page.getByRole("button", { name: "Hide sensitive amounts", exact: true }).click();
  await page.getByRole("button", { name: "Trade actions", exact: true }).click();
  await page.getByRole("menuitem", { name: "Share card", exact: true }).click();
  await page.getByRole("switch", { name: /Show dollar amounts/ }).click();
  const card = page.getByRole("dialog", { name: "Share card", exact: true }).getByRole("img");
  await expect(card).toContainText("••••");
  await page.getByRole("button", { name: "Close", exact: true }).click();
  await page.getByRole("button", { name: "Show sensitive amounts", exact: true }).click();

  await page.getByRole("link", { name: "Calendar", exact: true }).click();
  await expect(page.getByRole("button", { name: "Year", exact: true })).toBeVisible();
  await flipPrivacy(page);
  await page.getByRole("button", { name: "Year", exact: true }).click();
  await flipPrivacy(page);
  await page.getByRole("link", { name: "Settings", exact: true }).click();
  await page.getByRole("link", { name: "Accounts", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Accounts & funding", exact: true }),
  ).toBeVisible();
  await flipPrivacy(page);

  await page.getByRole("button", { name: "Hide sensitive amounts", exact: true }).click();
  await page.getByRole("link", { name: "Home", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Historical account value", exact: true }),
  ).toBeVisible();
  await expect.poll(async () => (await moneyState(page)).amounts).toBe(0);
  await expect(
    page.getByRole("button", { name: "Show sensitive amounts", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
});
