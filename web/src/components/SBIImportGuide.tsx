import { useLingui } from "@lingui/react/macro";
import {
  Accordion,
  AccordionItem,
  AccordionPanel,
  AccordionTrigger,
} from "@/components/ui/accordion";

/** SBI menu labels stay Japanese so users can find them on the broker site. */
export function SBIImportGuide() {
  const { t } = useLingui();
  return (
    <div className="text-sm leading-relaxed text-muted-foreground">
      <p>
        {t({
          id: "imports.sbiFiles",
          message: `Account reconstruction uses both ${"約定履歴 CSV"} and ${"入出金明細 CSV"}. Import each file separately.`,
        })}
      </p>
      <Accordion>
        <AccordionItem value="sbi-files">
          <AccordionTrigger className="py-2 text-primary">
            {t({
              id: "imports.sbiGuide",
              message: "Which CSV should I export? What is it used for?",
            })}
          </AccordionTrigger>
          <AccordionPanel className="space-y-4 pt-2 text-[13px] leading-relaxed">
            <section className="space-y-1">
              <h3 className="font-medium text-foreground" lang="ja">
                約定履歴 CSV
              </h3>
              <p>
                {t({
                  id: "imports.sbiExecutionsPurpose",
                  message:
                    "Required for trade history. Creates executions to reconstruct buys, sells, quantities, prices and dates. Include opening trades before the first closing trade you want to review; missing openings can leave reconstruction incomplete.",
                })}
              </p>
              <p lang="ja">口座管理 → 取引履歴 → 約定履歴</p>
              <p>
                {t({
                  id: "imports.sbiExecutionsExport",
                  message:
                    "Select the execution-date period and export the original CSV. Import this first, into the account that will receive any supplemental P&L data.",
                })}
              </p>
            </section>
            <section className="space-y-1">
              <h3 className="font-medium text-foreground" lang="ja">
                入出金明細 CSV
              </h3>
              <p>
                {t({
                  id: "imports.sbiCashPurpose",
                  message:
                    "Required for cash-flow and contributed-capital reconstruction; optional for a trade-only journal. Creates supported cash movements such as deposits and withdrawals, not another set of trades.",
                })}
              </p>
              <p lang="ja">入出金 → 入出金明細</p>
              <p>
                {t({
                  id: "imports.sbiCashExport",
                  message:
                    "Select the corresponding cash-movement period and export the original CSV. Upload separately, before or after execution history; include earlier funding if you need the full account history.",
                })}
              </p>
            </section>
            <section className="space-y-1">
              <h3 className="font-medium text-foreground" lang="ja">
                実現損益詳細 CSV
              </h3>
              <p>
                {t({
                  id: "imports.sbiPnlPurpose",
                  message:
                    "Optional, after importing executions. Supplements existing SBI closes with broker-reported pretax realized P&L and acquisition basis, even when trade history was imported earlier. Creates no new executions or cash movements.",
                })}
              </p>
              <p lang="ja">My資産 → 実現損益詳細 → 国内株式（現物） / 国内株式（信用）</p>
              <p>
                {t({
                  id: "imports.sbiPnlExport",
                  message: `Select ${"約定日"}, the period and account, then ${"照会"}. Open the domestic-stock detail and use ${"CSVダウンロード"}; use detail rows, not summary totals. Supported transactions: ${"売却 / 返済売 / 返済買 / 現渡"}.`,
                })}
              </p>
              <p>
                {t({
                  id: "imports.sbiPnlPeriod",
                  message:
                    "Cover the same closing dates as the imported executions in this account. SBI CSV downloads cover executions through the previous day, with up to 5,000 detail rows per download; split longer periods. Matching uses date, symbol, transaction, quantity, price and available settlement evidence, not an assumed intraday order.",
                })}
              </p>
              <p>
                {t({
                  id: "imports.sbiPnlCaveats",
                  message:
                    "Review the enrichment preview before confirming. Repeated or overlapping exports do not add trades or apply the same enrichment twice. Unmatched rows create nothing; check the account and date coverage. Invalid, ambiguous or conflicting rows need review and may block confirmation; existing values are not silently overwritten.",
                })}
              </p>
              <p>
                {t({
                  id: "imports.sbiPnlValues",
                  message:
                    "Reconstructed estimates can differ from broker-reported values. Reported P&L is pretax and includes broker costs; do not deduct fees again. Tax cash movements remain separate. SBI may revise provisional basis and costs after settlement; a changed report can produce a conflict rather than an automatic replacement.",
                })}
              </p>
              <a
                className="text-primary underline underline-offset-4"
                href="https://search.sbisec.co.jp/v2/popwin/guide/tool/assets/03_profits_loss/profits_loss_domestic.html"
                target="_blank"
                rel="noreferrer"
              >
                {t({ id: "imports.sbiOfficialHelp", message: "SBI export help (Japanese)" })}
              </a>
            </section>
            <p>
              {t({
                id: "imports.sbiPreviewDuplicates",
                message:
                  "Upload one original file at a time and check its preview. Execution and cash imports skip recognized duplicates in overlapping periods; inspect the result before uploading the next file.",
              })}
            </p>
            <p>
              {t({
                id: "imports.sbiHistoryGuide",
                message:
                  "Import history records uploads. Reprocess is offered only for supported imports with retained source rows; it reparses those rows with current rules and can change historical results. Legacy imports without retained rows, cash-only imports and P&L enrichment cannot be replayed there. Reprocessing does not fetch missing broker data or replace P&L enrichment.",
              })}
            </p>
          </AccordionPanel>
        </AccordionItem>
      </Accordion>
    </div>
  );
}
