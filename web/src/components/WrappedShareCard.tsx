import { useLingui } from "@lingui/react";
import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { useMemo, useState } from "react";
import { buildWrappedShareCard } from "@/lib/shareCard";
import type { YearWrapped } from "@/lib/wrapped";
import { ShareCardModal } from "@/components/ShareCard";

export function WrappedShareModal({
  wrapped,
  currency,
  fxRate,
  inProgress,
  open,
  onOpenChange,
}: {
  wrapped: YearWrapped;
  currency: string;
  fxRate: number;
  inProgress: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t: localize } = useSecondaryLingui();

  const { i18n } = useLingui();
  const locale = i18n.locale;
  const [showAmounts, setShowAmounts] = useState(false);

  const data = useMemo(
    () =>
      buildWrappedShareCard(wrapped, {
        showAmounts,
        locale,
        currency,
        fxRate,
        inProgress,
      }),
    [wrapped, showAmounts, currency, fxRate, inProgress, locale],
  );

  return (
    <ShareCardModal
      data={data}
      filename={`tradermemos-${wrapped.year}-wrapped.png`}
      privacyHint={localize({
        id: "wrapped.offSharesOnlyWinRateAndRatiosAccountSizeStaysPrivate",
        message: "Off shares only win rate and ratios — account size stays private.",
      })}
      showAmounts={showAmounts}
      onShowAmountsChange={setShowAmounts}
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}
