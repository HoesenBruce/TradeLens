# SBI Securities Import

## Status

Implemented for SBI Securities `約定履歴照会` CSV exports.

## Goal

Import Japanese equity trades exported by SBI Securities into TraderMemos.

## Non-goals for v1

- Dividends
- Investment trusts
- NISA-specific tax handling
- Automated SBI login
- SBI website scraping
- Automatic account synchronization

## Architecture Principle

Reuse the current importer pipeline:

```text
SBI CSV
  → SBI-specific detection and normalization
  → importer.ParsedExecution
  → existing validation and deduplication
  → importer.Commit
  → existing persistence and trade grouping
```

Broker-specific code should live under `api/internal/importer/`, with real fixtures in
that package's `testdata/`. Prefer an isolated SBI parser when normalization cannot be
expressed safely as a `BrokerPreset`; keep any preset/dispatch registration to one small
integration point.

Do not write SBI rows directly to database tables unless a later design review proves
that the canonical importer cannot represent required data.

## Candidate v1 Fields

- `symbol`
- `side`
- `quantity`
- `price`
- `executed_at`
- `fees`
- `account`

Japanese security codes are strings, not numbers. Valid examples include `6501`, `285A`,
`200A`, and `584A`; parsing must preserve them verbatim.

## Japanese Margin Trading

The design must be able to represent:

- `現物買`
- `現物売`
- `信用新規買`
- `信用新規売`
- `信用返済買`
- `信用返済売`

The verified export headers are `約定日`, `銘柄コード`, `取引`, `約定数量`,
`約定単価`, and `手数料/諸経費等`. Cash, margin-long, and margin-short fills are
grouped separately. `現引` becomes a margin-long close followed by a cash open at the
same price; its fee is charged once on the margin close.

## Encoding

The verified export uses CP932. It is decoded before header detection with the project's
existing `golang.org/x/text` dependency; UTF-8 fixtures remain accepted for tests.

## Timezone

Treat offset-less SBI timestamps as `Asia/Tokyo`, then follow the existing importer's
timezone normalization behavior. Timestamps that include an explicit offset remain
authoritative.

## Deduplication

Reuse `importer.DedupHash` and `importer.Commit`. Do not create an SBI-only duplicate
detection or persistence path.

## Testing

The anonymized implementation fixture covers:

- cash buy and cash sell;
- margin long open and margin short open;
- margin repayment in both applicable directions;
- an alphanumeric Japanese ticker;
- duplicate rows;
- a malformed row;
- the verified SBI encoding;
- offset-less timestamps interpreted in `Asia/Tokyo`.

The fixture must be real and anonymized. Tests should prove canonical parsed executions,
duplicate handling, and commit/regroup behavior without weakening upstream tests.
