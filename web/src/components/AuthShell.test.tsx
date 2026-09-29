import { I18nProvider } from "@lingui/react";
import { i18n } from "@lingui/core";
import { render, screen } from "@testing-library/react";
import { expect, it } from "vite-plus/test";
import { AuthShell } from "./AuthShell";
import { BRAND } from "@/lib/brand";

it("keeps branding and form visible without an animation gate", () => {
  const { container } = render(
    <I18nProvider i18n={i18n}>
      <AuthShell>
        <input aria-label="Username" />
      </AuthShell>
    </I18nProvider>,
  );
  expect(screen.getAllByText(BRAND.tagline)).toHaveLength(2);
  expect(screen.getByRole("textbox", { name: "Username" })).toBeEnabled();
  expect(container.querySelector("[data-auth-surface]")?.className).not.toContain("animate");
  const lens = container.querySelector("[data-auth-lens]");
  expect(lens?.getAttribute("class")).toContain("motion-safe:animate-[auth-lens-focus");
  expect(lens?.querySelector("circle")?.getAttribute("stroke")).toBe("#E5E7EB");
  expect(lens?.querySelector("path")?.getAttribute("stroke")).toBe("#3B82F6");
  for (const delay of [900, 1800, 2700]) {
    expect(container.innerHTML).toContain(`animation-delay: ${delay}ms`);
  }
  expect(container.innerHTML).not.toContain("infinite");
});
