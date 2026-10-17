# Issue #300: structured Coach contract for approval

Status: **REVIEW REQUIRED**, 2026-10-13. This is the B approval packet for
[#300](https://github.com/HoesenBruce/TradeLens/issues/300), based on main
`10c3be793691d029cb4fb1d8fe2dd401635f604a`. It specifies future behavior, not
implemented protection. No independent reviewer or named boundary owner has yet
accepted this revision. **B_APPROVED=NO; #301 remains blocked on B approval.**

This document makes the [architecture C](coach-secure-output-design.md) and
[migration plan](coach-security-migration-plan.md) precise. On overlapping v1
schema, facts, catalog and lifecycle details this packet is authoritative once
approved; the other documents retain the threat model and migration policy.
No production code, database, configuration or deployment changes belong to B.

## Decisions requiring explicit acceptance

Only four templates ship in catalog `coach-1`. No model prose, symbols, money,
URLs, model name, journal text, setup names, private prompts or configurable text
can enter rendering. The provider may select eligible templates, but cannot
supply display values. Personalized narrative coaching is intentionally lost.
Private custom prompts remain stored but inactive; no fallback invokes them.

The provider knows its authentication credential and receives a minimal subset
of owned financial facts. Template choice, omissions, order, failure and timing
remain indirect channels. This design does **not** guarantee noninterference or
provider-side confidentiality/retention. A stakeholder requiring zero such
channel must reject this proposal and choose deterministic server findings or
an independently justified provider architecture. Quotas do not prove secrecy.

## Exact model grammar

UTF-8 JSON only. Validate decoded member names for duplicates, including escaped
aliases. Reject invalid UTF-8, lone surrogate escapes, duplicate/unknown keys at
all depths, nulls, absent keys, trailing non-whitespace, root arrays, nonfinite
numbers and coercions. Integer means a JSON integer token: `1.0`, `1e0`, `true`
and `"1"` are not version 1. Whitespace is allowed; string IDs match exactly.

A finding has **exactly** these required fields:

| Key | Type and allowed value |
|---|---|
| `schema_version` | integer `1` |
| `template_id` | one of the four catalog IDs below |
| `category` | string equal to that template's category |
| `priority` | string equal to that template's `warn` or `tip` |
| `fact_refs` | array of exactly one string: that template's advertised request-local ref |
| `action` | string equal to that template's action |
| `params` | object with zero members; no numeric or string parameters |

Streaming uses a sequence of complete objects after the provider adapter
assembles its deltas; the adapter owns framing, never the provider's event name.
The last object is exactly `{"schema_version":1,"kind":"end"}`. Zero to four
findings may precede it. All different eligible templates may coexist except
`review_execution` and `record_review`, which are mutually exclusive. Reject
repeat templates before emission. No object, trailing prose or second end may
follow end. Wait for normal provider completion too: end alone cannot commit.
EOF/error/truncation without end fails; refusal and output-limit finishes fail.

Blocking mode (and a permitted fallback) has exactly
`{"schema_version":1,"findings":[...finding objects...]}` with zero to four
entries, the same validation, and normal provider completion. No end object
inside that array. Validate the entire blocking envelope before emitting any of
its findings. Empty success persists an empty completed review with null next
action; it is not evidence that the trade is safe or well executed.

| Bound | Exact ceiling and failure rule |
|---|---|
| Assembled finding/end | 4,096 UTF-8 bytes including whitespace |
| Model output | 32,768 bytes across all attempts; blocking root included |
| Provider wire | 1,048,576 bytes across all attempts, including framing; also bound decompressed bytes to this ceiling |
| Nesting | four containers from root (root depth 1), including blocking envelope |
| Findings | four maximum, unique template IDs |
| Ref registry | four refs maximum, server allocates `f0` through `f3` for this request only |
| SSE | 65,536 encoded bytes including framing; reserve room for one terminal error before each nonterminal write |
| Deadline | min(valid configured timeout, 120 seconds); unset uses 60 seconds; nonpositive/invalid config is unavailable |
| Heartbeat | at most one progress repeat per five seconds, at most 24 in one run, within the same deadline |
| History computation | at most 10,000 candidate prior trades; excess makes session fact unavailable, never silently truncates it |

Count before append/allocation in C. Deadline includes lookup, fallback, decoding
and transaction; a DB/driver commit race is reconciled as described below. No
arbitrary provider retries in v1. At most one blocking fallback is allowed only
for a classified unsupported-stream/schema-mode response, before any finding,
under the same budgets/deadline. Malformed or rejected output never triggers
fallback. Unsupported JSON-mode providers fail; no free-text recovery.

## Fact allowlist and provenance

Build a separate immutable DTO; do not serialize `TradeContext`, which currently
contains arbitrary notes/tags/emotions, or `VisionConfig`. The renderer accepts
only a validated selection, typed snapshot and static catalog. Authentication,
transport and settings accessors are not renderer arguments.

The model receives only public instructions/catalog IDs and eligible ref/type/
value entries. Bind user/account/trade/snapshot internally, never send these IDs
upstream. Ref numbering is request-local in catalog order, skipping ineligible
facts. A ref from a different run, owner, account, trade or snapshot is invalid,
even if its token happens to be `f0`. Persist its binding with the snapshot.

| Fact type | Authoritative source, domain and missingness | Template predicate |
|---|---|---|
| `stop_recorded` | Owned `tradeDetailDTO.StopPrice`; successful journal lookup plus null means false, finite positive value means true; zero/negative/nonfinite means unavailable | `define_stop`: exactly false |
| `execution_grade` | Owned `tradeDetailDTO.TradeQuality` / `TradeContext.ExecGrade`; integer 1..5; null means no grade, invalid value means unavailable | `review_execution`: 1..5 |
| `loss_streak` | Successful current-account `coachHistoryFor` / `BuildSessionContext`; integer 0..10,000; derive from `StreakKind=loss` and `StreakLen`, else 0 for known non-loss; failed/incomplete lookup is unavailable | `review_loss_streak`: >=2 |
| `review_grade_absent` | Same owned `TradeQuality` lookup; null means true, valid 1..5 means false; invalid value or failed lookup is unavailable | `record_review`: exactly true |

`stop_recorded` deliberately replaces the draft's ambiguous `risk_defined`:
`InitialRisk > 0` is an amount, not proof a stop price is recorded. No analytics
or accounting definition changes. `record_review` refers to the existing trade
quality grade, not a new DB field, setup confidence or existence of a Coach row.
Absent journal after a successful owned query is known absence; query failure
is not absence. Missing/ineligible facts are not advertised and cannot be
selected. No numeric fact is displayed in catalog 1; numeric grades/counts only
control eligibility. No model-selected digits or numeric placeholders exist.

Session semantics remain the existing 180-day window, strictly closed before
entry, current account and existing currency exclusion/market timezone rules;
scratch trades end a streak. Wording explicitly says recorded, eligible history,
not all-time performance. Validate finite inputs before computation. Missing
entry/timezone or incomplete history makes this fact unavailable. Resolve equal
close timestamps deterministically by server trade ID before the existing stable
sort; never let provider order or DB row order choose the result. C must test
same-time loss/win/scratch fixtures without changing shared accounting behavior.

Authenticate JWT/PAT and resolve owned trade/account before upstream access.
Recheck authoritative user existence/access policy and trade/account ownership
before each client emission and inside the persistence transaction. Policy
revocation/disable must serialize with emission authorization (not merely poll
a stale token); the emission authorized before revocation is the last permitted
one. Cancel registered in-flight runs on disable. Already delivered bytes cannot
be recalled. Deleted or reassigned trade/account prevents subsequent readback;
snapshot validity never substitutes for current authorization.

## Finite catalog `coach-1`

Metadata and server next-action precedence are fixed in this order:

| Template | Category | Priority | Action |
|---|---|---|---|
| `define_stop` | risk | warn | `record_stop` |
| `review_loss_streak` | session | warn | `pause_review` |
| `review_execution` | execution | tip | `review_journal` |
| `record_review` | process | tip | `grade_trade` |

Each entry below is literal **headline / detail / action label**. No interpolation,
HTML or Markdown. The action is a suggestion to use existing journal controls,
never an order or automatic write. Completion selects the action from the first
selected template in the precedence above, independent of provider order.

| Template | Locale | Headline / detail / action label |
|---|---|---|
| define_stop | en | Record a stop / No stop price is recorded for this trade. / Record a stop price |
| define_stop | zh-CN | 记录止损 / 这笔交易尚未记录止损价。 / 记录止损价 |
| define_stop | zh-HK | 記錄止蝕 / 這筆交易尚未記錄止蝕價。 / 記錄止蝕價 |
| define_stop | ja | 損切り価格を記録 / この取引には損切り価格が記録されていません。 / 損切り価格を記録する |
| define_stop | ko | 손절 가격 기록 / 이 거래에는 손절 가격이 기록되어 있지 않습니다. / 손절 가격 기록하기 |
| review_loss_streak | en | Review the loss streak / Eligible recorded trades show consecutive losses before this entry within the review window. / Pause and review |
| review_loss_streak | zh-CN | 复盘连续亏损 / 在复盘时间范围内，符合条件的已记录交易显示本次入场前有连续亏损。 / 暂停并复盘 |
| review_loss_streak | zh-HK | 檢視連續虧損 / 在檢視時段內，符合條件的已記錄交易顯示本次入場前有連續虧損。 / 暫停並檢視 |
| review_loss_streak | ja | 連続損失を振り返る / 振り返り期間内の対象となる記録済み取引では、このエントリー前に損失が続いています。 / 一度立ち止まって振り返る |
| review_loss_streak | ko | 연속 손실 복기 / 복기 기간 내 조건에 맞는 기록된 거래에서 이번 진입 전에 연속 손실이 있었습니다. / 잠시 멈추고 복기하기 |
| review_execution | en | Review execution / This trade has a recorded execution grade. Review it alongside your journal. / Review the journal |
| review_execution | zh-CN | 复盘交易执行 / 这笔交易已有执行评分，请结合交易日志复盘。 / 查看交易日志 |
| review_execution | zh-HK | 檢視交易執行 / 這筆交易已有執行評分，請結合交易日誌檢視。 / 查看交易日誌 |
| review_execution | ja | 取引の実行を振り返る / この取引には実行評価が記録されています。取引日誌と合わせて振り返りましょう。 / 取引日誌を確認する |
| review_execution | ko | 거래 실행 복기 / 이 거래에는 실행 평가가 기록되어 있습니다. 거래 일지와 함께 복기하세요. / 거래 일지 확인하기 |
| record_review | en | Grade this trade / This trade has no recorded execution grade. / Record an execution grade |
| record_review | zh-CN | 为交易评分 / 这笔交易尚未记录执行评分。 / 记录执行评分 |
| record_review | zh-HK | 為交易評分 / 這筆交易尚未記錄執行評分。 / 記錄執行評分 |
| record_review | ja | 取引を評価する / この取引には実行評価が記録されていません。 / 実行評価を記録する |
| record_review | ko | 거래 평가 / 이 거래에는 실행 평가가 기록되어 있지 않습니다. / 실행 평가 기록하기 |

Withheld history uses the following literal headline/detail; no action:

| Locale | Headline / detail |
|---|---|
| en | Previous review unavailable / This review used an older output format and cannot be displayed safely. You can request a new review when Coach is available. |
| zh-CN | 历史复盘不可用 / 这份复盘使用旧版输出格式，无法安全展示。Coach 可用时，你可以请求新的复盘。 |
| zh-HK | 歷史覆盤無法顯示 / 這份覆盤使用舊版輸出格式，無法安全顯示。Coach 可用時，你可以請求新的覆盤。 |
| ja | 過去のレビューは表示できません / このレビューは古い出力形式のため、安全に表示できません。Coach が利用可能なときに、新しいレビューを依頼できます。 |
| ko | 이전 복기를 표시할 수 없음 / 이 복기는 이전 출력 형식으로 작성되어 안전하게 표시할 수 없습니다. Coach를 사용할 수 있을 때 새 복기를 요청할 수 있습니다. |

Exact supported locale IDs: `en`, `zh-CN`, `zh-HK`, `ja`, `ko`; unsupported/missing
preference uses `en`. Locale is client/server selected, never model selected.
Pin catalog version on stored rows, retain historical catalogs. Missing locale
entry uses that catalog's reviewed English; missing template/catalog withholds
the row. Translation changes require a new catalog version and review. Status,
progress and error messages use the existing UI translation system with fixed
keys, never provider strings; C's localization owner must supply/review those
control translations before UI acceptance. The five-language wording above
also requires localization sign-off; presence is not proof of translation quality.

## API and SSE state machine

Existing routes are POST `/api/v1/trades/:id/coach`, POST the same path plus
`/stream`, and GET the same path plus `/reviews`. `output_version=2` is a query
parameter on all three. Reject repeated/unknown versions with generic 400 before
provider access. Auth/ownership errors retain 401/404 with no provider call.
Disabled v2 generation returns generic 503 `unavailable` before opening SSE;
history still returns safe placeholders. Runtime failures after SSE starts use
one terminal event, not a changed HTTP status. Blocking failures use generic
502 generation_failed, 504 timeout, 500 save_failed, or 503 unavailable.

Every event has required `version:2`, server UUID `run_id` and integer `sequence`
starting at 0 and increasing by one, plus exactly the fields below. SSE event
name is fixed by the server. UUIDs/time/sequence are typed server metadata, not
model strings. The client rejects extra/missing keys, bad type, foreign run,
sequence gap/duplicate, unknown event and EOF without a terminal event.

| Event | Additional required keys |
|---|---|
| started | none |
| progress | `stage`: preparing, analyzing or saving |
| finding | `finding`: object below; `provisional`: true |
| completed | `review_id`: server UUID, `created_at`: server UTC RFC3339 timestamp, `catalog_version`: coach-1, `findings`: array, `next_action`: object or null, `provisional`: false |
| error | `code`: generation_failed, timeout, canceled, save_failed or unavailable; `message`: fixed localized server message |

Finding exact keys: `id` (server UUID), `template_id`, `category`, `priority`,
`action` (catalog enums), `headline`, `detail` (literal catalog text). No refs,
params, private binding, raw provider model, usage/finish reason or arbitrary
metadata in the public finding. `next_action` is exactly `{action,label}` from
the catalog, or null when there are no findings. Completed findings reuse the
same IDs/text as previews; replay/history may rerender in a newly requested
locale. Blocking success returns the completed payload with version/run metadata
and sequence 0; it never returns completion without a persisted review ID.

`started → preparing → analyzing → (finding | analyzing)* → saving → completed`
is the success order (progress entries may be omitted, heartbeat repeats only
the current stage). Any nonterminal state may end in error. Saving occurs only
after explicit model end and normal transport completion. Each finding is
fully safe **before flush**; provisional means not yet committed, never
provisionally confidential. On error, clear previews, preserve previous completed
history and display the generic status. No review is saved for invalid output.

| Transition | Required behavior |
|---|---|
| Cancel/navigation/disconnect before commit | Cancel upstream, stop emission, roll back; send canceled only if connection remains writable. UI cancellation does not wait for an event on a dead socket. |
| Bad later record | Earlier safe findings may have been seen; clear previews, terminal generation_failed, zero completed writes. |
| Save failure | Roll back, terminal save_failed; no success-shaped unsaved review. |
| Disconnect/timeout racing commit | Commit may already have succeeded; never infer absence from socket error or ambiguous commit error. Reconcile by key on a fresh authorized read. |
| Retry after failure | New run and fresh snapshot; no prior preview accumulation; explicit user retry only. |
| Disable/revocation | Cancel and stop further authorized emissions under the serialized policy; rollback if uncommitted. |
| EOF after completed | Success already established; no further event accepted. |

Generation requires `Idempotency-Key`: client-generated UUID, bound to current
user and trade, never echoed into prompts/logs. One in-flight run per key;
concurrent duplicate returns generic 409 `in_progress`, never another provider
call. Completed duplicate returns the stored completion (new authorized stream
may send started then completed without previews/provider call). A definitively
failed run releases its key; retry may reuse it. Unknown commit outcome must
first query the unique `(user_id,trade_id,idempotency_key)` row; do not blindly
retry the provider. Keep the in-flight reservation until commit outcome is known;
a restart reconciles it against durable state. C must implement a DB-backed
reservation/lease with fencing for multi-process safety: an expired worker must
not emit or commit after a successor acquires its reservation. Lease loss cancels
the run; expiry alone never proves an ambiguous commit failed. No replay based on
Last-Event-ID, no automatic EventSource/reconnect or billable retry.

## Storage, history, old clients and rollback

Approve the additive `coach_reviews_v2` decision in the migration plan. Persist
only exact validated selections and minimal immutable typed snapshot, catalog
version, server identity/bindings/time, finding IDs and idempotency key; no rendered model
text or transport configuration. Snapshot stores known absence explicitly;
unknown is never reconstructed as false. Readback validates exact types,
predicates, cross-fact consistency, unique selections, versions and current
ownership, then renders. No fallback to legacy JSON parsing on corruption.

History v2 root: `{version:2,reviews:[...],next_cursor:null|string}`. Page size
20, maximum request limit 100 (invalid limit → 400), merged descending server
`created_at,id`; cursor is bounded, server-validated and bound to user/trade,
not a raw SQL fragment. A complete entry contains exactly `id,created_at,status,
catalog_version,findings,next_action`; status is complete. Withheld/unavailable
entry contains exactly `id,created_at,status,message`; no model/notes/snippet.
Invalid legacy ID/time never enters a response: omit that corrupt row and emit
only a fixed diagnostic code, not its contents. Paging still advances over the
row to avoid loops. `legacy_withheld` covers legacy rows; `unavailable` covers
corrupt/unknown v2. Future pagination implementation must test ties across tables.

Absent version invokes only a safe adapter, never the old generator. To avoid
old clients missing the required idempotency key, **absent-version generation
returns existing `source:off,notes:[]` (JSON) or fixed unavailable error (SSE)**
with no provider call. This deliberately chooses the migration plan's explicit
unavailable compatibility option instead of emulating v1 generation. History
still adapts validated v2 findings or a fixed placeholder into existing notes;
warn maps to tone warn/priority 1, tip to tone tip/priority 2; placeholder uses
tone tip/priority 2 and a server note ID. Omit model and absent next_action.
Old clients must upgrade for new generation. Mobile is preserved at the contract
level and unvalidated, outside B's scope.

Legacy rows remain byte-for-byte unchanged. Authenticated history/export/share
paths cannot expose old prose to owners either. No silent migration/regeneration.
Web clears stale Coach caches on upgrade/user change/logout, uses no-store and
shows withheld entries as unavailable, not completed advice. C and E must audit
service-worker/open-tab behavior; previously copied bytes cannot be recalled.

SQLite/PostgreSQL migration and restore rehearsals must preserve unrelated
financial/journal data and legacy hashes. Rollback retains v2 tables and uses a
binary with E's containment/history fixes. Disable/cancel first; never roll back
to unsafe legacy generation/readback, destructively down-migrate, or automatically
restore a backup. Restoring production data requires separate authorization.

## Boundary owners and independent acceptance

Role accountability below is assigned by boundary; named people and acceptance
are **pending nomination**, not inferred from repository ownership. A single
implementer may fill several roles, but the security approver must review
independently. B cannot be marked approved until the table and sign-offs are
completed with a reviewed commit SHA.

| Boundary | Accountable role | Implementation/acceptance surface |
|---|---|---|
| Input exclusion and transport | C backend owner (TBD) | coach generate/stream/prompt; no settings accessor in renderer or model DTO |
| Ownership, facts and predicates | C API/domain owner (TBD) | coach handlers/history; JWT/PAT, revocation and snapshot tests |
| Decoder, catalog renderer, budgets | C backend owner (TBD) | exact parser, duplicate-key/fuzz tests, per-event provenance |
| SSE, cancellation, idempotency | C API owner (TBD) | real HTTP flush, races, reconnect and commit recovery |
| Storage, readback and rollback | C persistence owner (TBD) | both SQL dialects, sqlc, legacy hashes and rollback rehearsal |
| Five locales and compatibility loss | C product/localization owner (TBD) | catalog wording, withheld UX, private prompt retirement, old-client unavailable |
| Web/parser/caches | C Web owner (TBD) | trades API/types/useTradeCoach; real API browser transitions |
| Availability and shared News isolation | E/Core owner (TBD) | #299 policy and separately accepted News error/success/log/persistence boundary |
| Independent contract acceptance | B security reviewer (TBD) | traceability, adversarial escape review, residual-channel acceptance |
| Independent implementation acceptance | D2 security reviewer (TBD) | #303 exact candidate proof, distinct from this design sketch |

News `AnalyzeNews` currently shares transport and returns arbitrary successful
text. This catalog grants it **no** approval. E contains that path unless a
separate Core confidentiality acceptance covers success/error, credential,
log and storage paths. Manual News workflows remain available. News recovery
and Core D1 do not depend on C/D2; Coach re-enablement does.

Required B sign-off record: reviewer identity, exact PR commit SHA, each of the
five Issue acceptance items PASS/FAIL with evidence, named owners, product and
locale acceptance, residual-channel acceptance, and unresolved objections.
Any contract revision invalidates affected approvals. A merged planning document,
local sketch, CI pass or author self-review is not independent approval. No
release, merge, Issue closure or operational enablement is authorized here.

## Traceability and verification limits

| Issue acceptance | Review anchor | Future C/D2 proof |
|---|---|---|
| 1: all renderable values have provenance | facts/catalog and event exact keys above | config canaries absent from all frames/logs/history; only catalog/owned typed metadata |
| 2: fail-closed schema and references | model grammar, fact domains, budgets | real parser fuzz/boundaries; cross-user/account/run refs; invalid/missing facts |
| 3: safe before flush and lifecycle | state machine and transition table | real socket finding before held end; late failure, cancel, duplicate key and ambiguous commit |
| 4: locales, legacy and rollback | finite catalog, storage/old clients section | native wording review; five-locale rerender; two DB rehearsals; Web reset/readback/cache |
| 5: explicit limitations | decisions and News boundary | security/product acceptance of choice/timing channel and external input privacy |

[Design sketch and adversarial ledger](../validation/issue300/README.md) provide
reproducible synthetic schema checks. They do not prove actual HTTP flush,
production parser resource safety, authorization races, SQL transaction semantics,
translation quality or deployment behavior. #301 implements those; #303 verifies
the integrated candidate independently. B does not require full product test
reruns or browser/mobile E2E for this documentation-only change.
