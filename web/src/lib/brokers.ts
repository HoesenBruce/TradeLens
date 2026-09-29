import { t as localize } from "@lingui/core/macro";
import { t as tr } from "@lingui/core/macro";
/**
 * The broker catalogue behind the Connect flow.
 *
 * The server recognises broker exports by their header signature
 * (`api/internal/importer/brokers.go`) but that detection is invisible: the
 * import page asks for "a file" and silently guesses. This catalogue is the
 * user-facing half — it names the brokers, says how to get the file out of
 * each one, and routes to the connection method that broker supports.
 *
 * `key` matches the server preset key wherever one exists, so a card that
 * claims "we recognise this layout" is making a checkable promise.
 */

export type BrokerConnectKind =
  /** Credentials in the app; fills arrive on a schedule. */
  | "sync"
  /** The broker exports a file; we parse it. */
  | "file"
  /** No export at all — trades are typed in. */
  | "manual";

export interface BrokerDef {
  key: string;
  name: string;
  /** Stored on `account.broker` when the flow creates the account. */
  accountBroker: string;
  kind: BrokerConnectKind;
  /**
   * True when `importer.brokerPresets` carries a signature for this export —
   * the column mapping is then pre-filled instead of guessed.
   */
  recognised: boolean;
  /** Brand colour for the mark tile, used until a logo asset is dropped in. */
  brand: string;
  /** Monogram shown on the tile when no logo file is present. */
  monogram: string;
  /** File types the export produces. */
  formats?: string;
  /** How to get the data out, in the broker's own menu vocabulary. */
  steps: string[];
  /** A caveat worth knowing before starting. */
  note?: string;
  /** Extra search terms — old names, parent companies, platform names. */
  aliases?: string[];
}

export const BROKERS: BrokerDef[] = [
  {
    key: "sbi",
    name: "SBI Securities",
    accountBroker: "SBI Securities",
    kind: "file",
    recognised: true,
    brand: "#E60012",
    monogram: "SBI",
    get formats() {
      return "約定履歴 CSV · 入出金明細 CSV";
    },
    get steps() {
      return [
        tr({
          id: "market.sbi1",
          message: `In SBI Securities, open ${"口座管理 → 取引履歴 → 約定履歴"}, select the import period, and export the CSV.`,
        }),
        tr({
          id: "market.sbi2",
          message: `Open ${"入出金 → 入出金明細"}, select the corresponding period, and export the CSV.`,
        }),
        tr({
          id: "market.sbi3",
          message:
            "Import both exports to reconstruct trades, cash flows, and contributed capital. Upload one original CSV at a time; CP932 encoding and report headers are handled automatically.",
        }),
      ];
    },
    get note() {
      return tr({
        id: "market.sbiNote",
        message: "Cash, margin, and 現引 position conversions are supported.",
      });
    },
    aliases: ["sbi", "sbisec", "sbi証券", "エスビーアイ"],
  },
  {
    key: "ibkr",
    name: "Interactive Brokers",
    accountBroker: "IBKR",
    kind: "sync",
    recognised: true,
    brand: "#D91F26",
    monogram: "IB",
    get formats() {
      return localize({
        id: "broker.flexWebServiceOrActivityStatementCsv",
        message: "Flex Web Service, or Activity Statement CSV",
      });
    },
    get steps() {
      return [
        localize({
          id: "broker.inClientPortalOpenPerformanceReportsFlexQueries",
          message: "In Client Portal open Performance & Reports → Flex Queries.",
        }),
        localize({
          id: "broker.createATradeConfirmationFlexQueryWithTheTradesExecutionsSectionFormat",
          message:
            "Create a Trade Confirmation Flex Query with the Trades → Executions section, format CSV, and save it.",
        }),
        localize({
          id: "broker.copyTheQueryIdShownNextToTheSavedQuery",
          message: "Copy the Query ID shown next to the saved query.",
        }),
        localize({
          id: "broker.underPerformanceReportsSettingsEnableTheFlexWebServiceAndCopyThe",
          message:
            "Under Performance & Reports → Settings, enable the Flex Web Service and copy the token.",
        }),
      ];
    },
    get note() {
      return localize({
        id: "broker.theTokenExpiresYearlyIbkrEmailsAReminderBeforeItDoes",
        message: "The token expires yearly — IBKR emails a reminder before it does.",
      });
    },
    aliases: ["ib", "tws", "flex", "interactive brokers"],
  },
  {
    key: "thinkorswim",
    name: "thinkorswim",
    accountBroker: "Charles Schwab",
    kind: "file",
    recognised: true,
    brand: "#00A0DF",
    monogram: "TS",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openTheMonitorTabAccountStatement",
          message: "Open the Monitor tab → Account Statement.",
        }),
        localize({
          id: "broker.setTheDateRangeYouWantToJournal",
          message: "Set the date range you want to journal.",
        }),
        localize({
          id: "broker.useTheMenuAtTheTopRightOfTheStatementToExport",
          message: "Use the menu at the top right of the statement to export it as CSV.",
        }),
        localize({
          id: "broker.uploadTheFileTheAccountTradeHistorySectionIsThePartWe",
          message: "Upload the file — the Account Trade History section is the part we read.",
        }),
      ];
    },
    aliases: ["tos", "schwab", "td ameritrade", "ameritrade"],
  },
  {
    key: "schwab",
    name: "Charles Schwab",
    accountBroker: "Charles Schwab",
    kind: "file",
    recognised: true,
    brand: "#00A0DF",
    monogram: "CS",
    formats: "CSV",
    get steps() {
      return [
        localize({ id: "broker.openAccountsHistory", message: "Open Accounts → History." }),
        localize({
          id: "broker.pickTheAccountAndTheDateRangeWithTransactionsSelected",
          message: "Pick the account and the date range, with Transactions selected.",
        }),
        localize({ id: "broker.exportTheResultAsCsv", message: "Export the result as CSV." }),
      ];
    },
    aliases: ["schwab"],
  },
  {
    key: "webull",
    name: "Webull",
    accountBroker: "Webull",
    kind: "file",
    recognised: true,
    brand: "#1D68F1",
    monogram: "WB",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openTheWebullDesktopAppThePhoneAppCannotExport",
          message: "Open the Webull desktop app (the phone app cannot export).",
        }),
        localize({
          id: "broker.goToOrdersAndPickTheDateRange",
          message: "Go to Orders and pick the date range.",
        }),
        localize({ id: "broker.exportTheOrdersAsCsv", message: "Export the orders as CSV." }),
      ];
    },
    get note() {
      return localize({
        id: "broker.onlyFilledOrdersImportCancelledAndPendingRowsAreSkipped",
        message: "Only filled orders import — cancelled and pending rows are skipped.",
      });
    },
    aliases: ["webull"],
  },
  {
    key: "tradovate",
    name: "Tradovate",
    accountBroker: "Tradovate",
    kind: "file",
    recognised: true,
    brand: "#0B8B3E",
    monogram: "TV",
    formats: "CSV",
    get steps() {
      return [
        localize({ id: "broker.openTheReportsSection", message: "Open the Reports section." }),
        localize({
          id: "broker.chooseFillsAndSetTheDateRange",
          message: "Choose Fills and set the date range.",
        }),
        localize({ id: "broker.downloadTheReportAsCsv", message: "Download the report as CSV." }),
      ];
    },
    get note() {
      return localize({
        id: "broker.timesAreReadAsExchangeChicagoTime",
        message: "Times are read as exchange (Chicago) time.",
      });
    },
    aliases: ["futures", "tradovate"],
  },
  {
    key: "ninjatrader",
    name: "NinjaTrader",
    accountBroker: "NinjaTrader",
    kind: "file",
    recognised: true,
    brand: "#F58220",
    monogram: "NT",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.inTheControlCenterOpenTradePerformance",
          message: "In the Control Center open Trade Performance.",
        }),
        localize({
          id: "broker.selectTheExecutionsTabAndTheDateRange",
          message: "Select the Executions tab and the date range.",
        }),
        localize({
          id: "broker.rightClickTheGridAndExportItAsCsv",
          message: "Right-click the grid and export it as CSV.",
        }),
      ];
    },
    aliases: ["ninja", "futures"],
  },
  {
    key: "ctrader",
    name: "cTrader",
    accountBroker: "cTrader",
    kind: "file",
    recognised: true,
    brand: "#1E6FD9",
    monogram: "cT",
    formats: "CSV",
    get steps() {
      return [
        localize({ id: "broker.openTheHistoryTab", message: "Open the History tab." }),
        localize({
          id: "broker.setThePeriodYouWantToJournal",
          message: "Set the period you want to journal.",
        }),
        localize({ id: "broker.exportTheHistoryToCsv", message: "Export the history to CSV." }),
      ];
    },
    get note() {
      return localize({
        id: "broker.rowsAreWholePositionsAndQuantitiesAreLotsCheckTheTimezoneIn",
        message:
          "Rows are whole positions, and quantities are lots — check the timezone in the preview before confirming.",
      });
    },
    aliases: ["forex", "cfd", "prop"],
  },
  {
    key: "dxtrade",
    name: "DXtrade",
    accountBroker: "DXtrade",
    kind: "file",
    recognised: true,
    brand: "#2A6DF4",
    monogram: "DX",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openOrderHistoryInTheTradingPortal",
          message: "Open Order History in the trading portal.",
        }),
        localize({ id: "broker.setTheDateRange", message: "Set the date range." }),
        localize({ id: "broker.exportItAsCsv", message: "Export it as CSV." }),
      ];
    },
    get note() {
      return localize({
        id: "broker.quantitiesAreLotsCommonOnPropFirmPortals",
        message: "Quantities are lots. Common on prop-firm portals.",
      });
    },
    aliases: ["prop", "forex", "funded"],
  },
  {
    key: "matchtrader",
    name: "Match-Trader",
    accountBroker: "Match-Trader",
    kind: "file",
    recognised: true,
    brand: "#0EA5A0",
    monogram: "MT",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openThePositionsOrHistoryView",
          message: "Open the Positions or History view.",
        }),
        localize({ id: "broker.setTheDateRange", message: "Set the date range." }),
        localize({ id: "broker.exportItAsCsv", message: "Export it as CSV." }),
      ];
    },
    get note() {
      return localize({
        id: "broker.rowsArePositionsStillOpenOnesImportWithTheirEntryFillOnly",
        message: "Rows are positions; still-open ones import with their entry fill only.",
      });
    },
    aliases: ["prop", "forex", "funded"],
  },
  {
    key: "metatrader",
    name: "MetaTrader 4 / 5",
    accountBroker: "MetaTrader",
    kind: "file",
    recognised: true,
    brand: "#0B7CBF",
    monogram: "M5",
    get formats() {
      return localize({ id: "broker.xlsxOrHtmlStatement", message: "XLSX or HTML statement" });
    },
    get steps() {
      return [
        localize({
          id: "broker.openTheToolboxMt5OrTerminalMt4AndSelectTheHistoryTab",
          message: "Open the Toolbox (MT5) or Terminal (MT4) and select the History tab.",
        }),
        localize({
          id: "broker.setThePeriodThenRightClickTheGrid",
          message: "Set the period, then right-click the grid.",
        }),
        localize({
          id: "broker.chooseReportXlsxMt5OrSaveAsReportMt4",
          message: "Choose Report → XLSX (MT5) or Save as Report (MT4).",
        }),
      ];
    },
    get note() {
      return localize({
        id: "broker.dealsImportInBrokerServerTimeEetByDefaultSetTheTimezone",
        message:
          "Deals import in broker server time (EET by default) — set the timezone in the preview if your broker differs.",
      });
    },
    aliases: ["mt4", "mt5", "metaquotes", "forex"],
  },
  {
    key: "tastytrade",
    name: "tastytrade",
    accountBroker: "tastytrade",
    kind: "file",
    recognised: false,
    brand: "#F04E23",
    monogram: "tt",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openHistoryInTheDesktopOrWebPlatform",
          message: "Open History in the desktop or web platform.",
        }),
        localize({
          id: "broker.setTheDateRangeAndFilterToFilledTransactions",
          message: "Set the date range and filter to filled transactions.",
        }),
        localize({ id: "broker.downloadTheCsv", message: "Download the CSV." }),
      ];
    },
    aliases: ["tasty", "options"],
  },
  {
    key: "fidelity",
    name: "Fidelity",
    accountBroker: "Fidelity",
    kind: "file",
    recognised: false,
    brand: "#368727",
    monogram: "Fi",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openAccountsTradePortfolioActivityOrders",
          message: "Open Accounts & Trade → Portfolio → Activity & Orders.",
        }),
        localize({ id: "broker.setTheDateRange", message: "Set the date range." }),
        localize({
          id: "broker.useDownloadToSaveTheActivityAsCsv",
          message: "Use Download to save the activity as CSV.",
        }),
      ];
    },
    aliases: ["fidelity"],
  },
  {
    key: "etrade",
    name: "E*TRADE",
    accountBroker: "E*TRADE",
    kind: "file",
    recognised: false,
    brand: "#6633CC",
    monogram: "E*",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openAccountsTransactions",
          message: "Open Accounts → Transactions.",
        }),
        localize({
          id: "broker.setTheAccountAndDateRange",
          message: "Set the account and date range.",
        }),
        localize({
          id: "broker.downloadTheTransactionsAsCsv",
          message: "Download the transactions as CSV.",
        }),
      ];
    },
    aliases: ["etrade", "morgan stanley"],
  },
  {
    key: "moomoo",
    name: "moomoo / Futu",
    accountBroker: "Moomoo",
    kind: "file",
    recognised: false,
    brand: "#FF6C00",
    monogram: "mm",
    formats: "CSV",
    get steps() {
      return [
        localize({
          id: "broker.openTheMoomooOrFutuDesktopApp",
          message: "Open the moomoo or FUTU desktop app.",
        }),
        localize({
          id: "broker.goToTheAccountSOrderOrTradeHistoryAndSetThe",
          message: "Go to the account's order or trade history and set the date range.",
        }),
        localize({ id: "broker.exportHistoryCsv", message: "Export the history as CSV." }),
      ];
    },
    aliases: ["futu", "niuniu", "moo moo", "hk"],
  },
  {
    key: "robinhood",
    name: "Robinhood",
    accountBroker: "Robinhood",
    kind: "file",
    recognised: false,
    brand: "#00C805",
    monogram: "RH",
    get formats() {
      return localize({
        id: "broker.csvYouBuildFromTheStatement",
        message: "CSV you build from the statement",
      });
    },
    get steps() {
      return [
        localize({
          id: "broker.openAccountSettingsStatementsHistory",
          message: "Open Account → Settings → Statements & History.",
        }),
        localize({
          id: "broker.downloadTheMonthlyAccountStatementsCoveringYourTrades",
          message: "Download the monthly account statements covering your trades.",
        }),
        localize({
          id: "broker.copyTheTradeRowsIntoASpreadsheetWithSymbolSideQuantityPrice",
          message:
            "Copy the trade rows into a spreadsheet with symbol, side, quantity, price and time columns, and save it as CSV.",
        }),
      ];
    },
    get note() {
      return localize({
        id: "broker.robinhoodOnlyPublishesPdfStatementsSoThisOneNeedsASpreadsheetStep",
        message:
          "Robinhood only publishes PDF statements, so this one needs a spreadsheet step. Logging trades by hand is often faster for a light month.",
      });
    },
    aliases: ["rh", "hood"],
  },
  {
    key: "generic",
    name: "Other broker",
    accountBroker: "Other",
    kind: "file",
    recognised: false,
    brand: "#64748B",
    monogram: "CSV",
    get formats() {
      return localize({
        id: "broker.anyCsvOrATradermemosJsonBackup",
        message: "Any CSV, or a TraderMemos JSON backup",
      });
    },
    get steps() {
      return [
        localize({
          id: "broker.exportYourTradeOrderOrExecutionHistoryFromTheBrokerAsCsv",
          message: "Export your trade, order or execution history from the broker as CSV.",
        }),
        localize({
          id: "broker.keepOneRowPerFillWithSymbolSideQuantityPriceAndA",
          message: "Keep one row per fill, with symbol, side, quantity, price and a timestamp.",
        }),
        localize({
          id: "broker.uploadItYouMapTheColumnsOntoThoseFieldsInTheNext",
          message: "Upload it — you map the columns onto those fields in the next step.",
        }),
      ];
    },
    get note() {
      return localize({
        id: "broker.roundTripExportsOneRowPerClosedPositionWorkTooMapThe",
        message:
          "Round-trip exports (one row per closed position) work too — map the open/close columns instead.",
      });
    },
    aliases: ["csv", "custom", "unknown", "spreadsheet"],
  },
  {
    key: "manual",
    get name() {
      return localize({ id: "broker.manualAccount", message: "Manual account" });
    },
    accountBroker: "Manual",
    kind: "manual",
    recognised: false,
    brand: "#1264B2",
    monogram: "✎",
    get steps() {
      return [
        localize({
          id: "broker.nameTheAccountAndPickItsCurrency",
          message: "Name the account and pick its currency.",
        }),
        localize({
          id: "broker.logTradesAsYouTakeThemOrFillInYesterdaySAt",
          message: "Log trades as you take them, or fill in yesterday's at review time.",
        }),
      ];
    },
    get note() {
      return localize({
        id: "broker.screenshotScanningAndTheTradeFormBothWriteHereNothingAboutThe",
        message:
          "Screenshot scanning and the trade form both write here — nothing about the journal needs a broker file.",
      });
    },
    aliases: ["by hand", "paper", "prop", "backtest", "no broker"],
  },
];

const BY_KEY = new Map(BROKERS.map((b) => [b.key, b]));

export function findBroker(key: string | undefined): BrokerDef | undefined {
  return key ? BY_KEY.get(key) : undefined;
}

/** Case- and punctuation-insensitive match over name, aliases and format. */
export function searchBrokers(query: string): BrokerDef[] {
  const q = query.trim().toLowerCase();
  if (!q) return BROKERS;
  return BROKERS.filter((b) => {
    const haystack = [b.name, b.accountBroker, b.formats ?? "", ...(b.aliases ?? [])]
      .join(" ")
      .toLowerCase();
    return haystack.includes(q);
  });
}

export const KIND_LABEL: Record<BrokerConnectKind, string> = {
  get sync() {
    return tr({ id: "market.autoSync", message: "Auto-sync" });
  },
  get file() {
    return tr({ id: "market.fileImport", message: "File import" });
  },
  get manual() {
    return tr({ id: "market.manual", message: "Manual" });
  },
};

/** Ordered so the picker's groups read best-effort-first. */
export const KIND_ORDER: BrokerConnectKind[] = ["sync", "file", "manual"];
