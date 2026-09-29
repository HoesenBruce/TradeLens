import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vite-plus/test";
import { I18nProvider, loadLocale } from "@/i18n";
import { LoginScreen } from "./LoginScreen";

afterEach(async () => {
  cleanup();
  await loadLocale("en");
  localStorage.clear();
  vi.restoreAllMocks();
});

const showLogin = () =>
  render(
    <I18nProvider>
      <LoginScreen />
    </I18nProvider>,
  );

it("uses browser language only without a saved preference, with English fallback", async () => {
  for (const [browser, expected] of [
    ["zh-CN", "zh-CN"],
    ["ja-JP", "ja"],
    ["fr-FR", "en"],
  ]) {
    localStorage.clear();
    vi.spyOn(navigator, "languages", "get").mockReturnValue([browser]);
    const view = showLogin();
    await waitFor(() => expect(screen.getByRole("combobox")).toHaveValue(expected));
    view.unmount();
  }
  localStorage.setItem("tm-locale", "ja");
  vi.spyOn(navigator, "languages", "get").mockReturnValue(["zh-CN"]);
  showLogin();
  await waitFor(() => expect(screen.getByRole("combobox")).toHaveValue("ja"));
});

it("switches immediately, preserves credentials and restores manual selection on re-entry", async () => {
  localStorage.setItem("tm-locale", "en");
  const user = userEvent.setup();
  const view = showLogin();
  expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual([
    "English",
    "中文",
    "日本語",
  ]);
  await user.type(screen.getByLabelText("Username"), "qa-user");
  await user.type(screen.getByLabelText("Password"), "qa-password");
  for (const [locale, heading, note] of [
    ["zh-CN", "登录", "等待回测确认。"],
    ["ja", "ログイン", "再テストを待った。"],
    ["en", "Sign in", "Waited for the retest."],
  ]) {
    await user.selectOptions(screen.getByRole("combobox"), locale);
    await waitFor(() =>
      expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(heading),
    );
    expect(screen.getByText(note)).toBeInTheDocument();
    expect(localStorage.getItem("tm-locale")).toBe(locale);
  }
  expect(screen.getByLabelText("Username")).toHaveValue("qa-user");
  expect(screen.getByLabelText("Password")).toHaveValue("qa-password");
  await user.selectOptions(screen.getByRole("combobox"), "ja");
  await waitFor(() => expect(localStorage.getItem("tm-locale")).toBe("ja"));
  view.unmount();
  showLogin();
  await waitFor(() => expect(screen.getByRole("combobox")).toHaveValue("ja"));
});
