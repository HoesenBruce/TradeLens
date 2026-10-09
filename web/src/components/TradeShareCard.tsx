import { useMoneyFormatters } from "@/lib/useMoneyFormatters";
import { useLingui } from "@lingui/react";
import { useLingui as useSecondaryLingui } from "@lingui/react/macro";
import { useMemo, useState } from "react";
import type { TradeDetail } from "@/lib/api/types";
import { buildTradeShareCard, type ShareCardData } from "@/lib/shareCard";
import type { TradeInsights } from "@/lib/tradeInsights";
import { ShareCardModal } from "@/components/ShareCard";

function cardFilename(data: ShareCardData): string {
  const date = data.dateLabel.replaceAll(/[^a-zA-Z0-9]+/g, "-").toLowerCase();
  return `tradermemos-${data.title.toLowerCase()}-${date}.png`;
}

export function TradeShareModal({
  trade,
  insights,
  open,
  onOpenChange,
}: {
  trade: TradeDetail;
  insights: TradeInsights;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t: localize } = useSecondaryLingui();

  const { i18n } = useLingui();
  const locale = i18n.locale;
  const { fmtSignedMoney } = useMoneyFormatters();
  const [showAmounts, setShowAmounts] = useState(false);

  const data = useMemo(
    () => buildTradeShareCard(trade, insights, { showAmounts, locale, fmtSignedMoney }),
    [trade, insights, showAmounts, locale, fmtSignedMoney],
  );

  return (
    <ShareCardModal
      data={data}
      filename={cardFilename(data)}
      privacyHint={localize({
        id: "wrapped.offSharesOnlyRMultiplesAndPercentagesAccountSizeStaysPrivate",
        message: "Off shares only R multiples and percentages — account size stays private.",
      })}
      showAmounts={showAmounts}
      onShowAmountsChange={setShowAmounts}
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}
