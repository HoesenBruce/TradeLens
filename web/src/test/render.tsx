import { I18nProvider } from "@lingui/react";
import { i18n } from "@/i18n";
import { render as renderReact, type RenderOptions } from "@testing-library/react";
import type { ReactNode } from "react";

export function render(ui: ReactNode, options: RenderOptions = {}) {
  const Wrapper = options.wrapper;
  return renderReact(ui, {
    ...options,
    wrapper: ({ children }) => (
      <I18nProvider i18n={i18n}>{Wrapper ? <Wrapper>{children}</Wrapper> : children}</I18nProvider>
    ),
  });
}
