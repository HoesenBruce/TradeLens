import { expect, type APIRequestContext } from "@playwright/test";

/** Synthetic data only. Use a dedicated throwaway API/database for this suite. */
export async function seedStatCards(request: APIRequestContext, api: string) {
  const credentials = { email: "qa333@synthetic.invalid", password: "synthetic-333-password" };
  let login = await request.post(`${api}/auth/login`, { data: credentials });
  if (!login.ok()) {
    await request.post(`${api}/setup`, { data: { ...credentials, name: "Synthetic card QA" } });
    login = await request.post(`${api}/auth/login`, { data: credentials });
  }
  expect(login.ok()).toBeTruthy();
  const { access_token } = await login.json();
  const headers = { Authorization: `Bearer ${access_token}` };
  const accounts = await (await request.get(`${api}/accounts`, { headers })).json();
  if (accounts.some((a: { name: string }) => a.name === "Synthetic JPY card QA")) return;
  const tag = await request.post(`${api}/tags`, {
    headers,
    data: { name: "SYNTHETIC_MISTAKE_ABCDEFGHIJKLMNOPQRSTUVWXYZ", kind: "mistake" },
  });
  expect(tag.ok()).toBeTruthy();
  const { id: tagId } = await tag.json();
  for (const [name, currency, type] of [
    ["JPY", "JPY", "margin"],
    ["USD", "USD", "cash"],
    ["Prop", "JPY", "prop"],
  ]) {
    const response = await request.post(`${api}/accounts`, {
      headers,
      data: {
        name: `Synthetic ${name} card QA`,
        broker: "Manual",
        account_type: type,
        base_currency: currency,
        starting_balance: 123456789,
      },
    });
    expect(response.ok()).toBeTruthy();
    const { id } = await response.json();
    if (type === "prop")
      expect(
        (
          await request.put(`${api}/accounts/${id}/prop-settings`, {
            headers,
            data: {
              profit_target: 10000000,
              max_drawdown: 20000000,
              drawdown_mode: "static",
              daily_loss_limit: 1000000,
              consistency_pct: 0.5,
            },
          })
        ).ok(),
      ).toBeTruthy();
    for (let i = 0; i < 16; i++) {
      const symbol = i === 0 ? "SYNTHETIC_LONG_SYMBOL_ABCDEFGHIJKLMN" : `SYNTH${i}`;
      const day = String((i % 14) + 1).padStart(2, "0");
      const delta =
        i < 3
          ? 10000
          : i < 5
            ? -10000
            : i === 5
              ? 10000
              : i === 14 || i === 15
                ? 0
                : [123457, -98765, 10000, -10000, 0, 1000000, -123456][i % 7];
      for (const [side, price, hour] of [
        ["buy", 1234567, "01"],
        ["sell", 1234567 + delta, "02"],
      ]) {
        const fill = await request.post(`${api}/executions`, {
          headers,
          data: {
            account_id: id,
            symbol,
            instrument_type: "stock",
            side,
            quantity: i === 3 || i === 4 ? 400 : 100,
            price,
            fees: i === 14 ? 0 : 1234,
            executed_at:
              i < 6
                ? `2026-10-01T01:${String(i * 10 + (side === "sell" ? 5 : 0)).padStart(2, "0")}:00Z`
                : i === 15
                  ? side === "buy"
                    ? "2016-10-01T01:00:00Z"
                    : "2026-10-15T02:00:00Z"
                  : `2026-10-${day}T${hour}:00:00Z`,
          },
        });
        expect(fill.status()).toBe(201);
      }
    }
    const trades = await (await request.get(`${api}/trades?account_id=${id}`, { headers })).json();
    for (const trade of trades)
      expect(
        (
          await request.patch(`${api}/trades/${trade.id}`, {
            headers,
            data: {
              initial_risk: 1000000,
              target_price: 2345678,
              stop_price: 1000000,
              mae: 5000000,
              mfe: 20000000,
              tag_ids: [tagId],
            },
          })
        ).ok(),
      ).toBeTruthy();
  }
  expect(
    (
      await request.put(`${api}/settings/risk-rules`, {
        headers,
        data: {
          max_risk_per_trade: 1000000,
          max_daily_loss: 10000000,
          max_open_risk: 20000000,
          default_account_risk_pct: 1,
          max_trades_per_day: 5,
          max_consecutive_losses: 3,
        },
      })
    ).ok(),
  ).toBeTruthy();
  expect(
    (
      await request.put(`${api}/settings/annual-goal`, {
        headers,
        data: { year: 2026, amount: 100000000, currency: "JPY" },
      })
    ).ok(),
  ).toBeTruthy();
}
