# Coach secure output architecture C

Status: proposed implementation contract, 2026-10-13; independent B approval and D2 acceptance remain required. AI Coach is retained as a planned, supported feature. No production redesign implemented. Baseline/head and full inventory are in [split assessment](pr295-split-assessment.md).

## Decision

Only server-approved bounded structured selections cross the model boundary. Only server-rendered text and verified, allowlisted user-owned facts cross the client boundary. JSON wrapping free text is insufficient. Remove arbitrary model text from headline, detail, next action, model metadata, error/progress and stored review display. Do not grow the recognizable-secret filter into the primary boundary.

The rejected design validates one note and flushes it before checking the aggregate. Moving cumulative checks earlier would address the published timing defect only within a finite recognition policy. It would still depend on guessing representations of private input. C instead makes each emitted finding independently safe under the declared data contract, including when generation later fails. Marking a note provisional never makes confidential disclosure acceptable.

## Trust and input construction

Current flow: `tradeContextFromDetail` → `coachHistoryFor` → `FormatTradeContext` → `systemPrompt` → `GenerateReview[Stream]`. Trade lookups use authenticated user; session history is filtered by user and current account. Journal query currently loads all the user's journals, then joins by trade. Preserve account scoping while narrowing retrieval where practical; do not promote account labels to authorization rules.

| Source | Current data | C treatment |
|---|---|---|
| Current owned trade | Direction/status, fills, entry/exit, P&L/fees, currency, duration, risk/stop/target, R, MAE/MFE | Server loads and computes facts; model receives only facts required by eligible templates. Stable finite enums, finite bounded numbers, explicit missingness. No account name/ID, broker credential, attachment or full DB row. |
| Current-account prior trades | Session/day/week totals, streak/drawdown, last three closes over bounded history | Prefer existing computed aggregate/execution axes; no other account. Do not send recent symbols/emotions by default. Preserve market timezone, currency exclusions and “as of entry” semantics. Missing history is explicit, never invented zero. |
| User journals | Notes, tags, setup names, emotion strings | Exclude arbitrary prose from v1 model input and renderable fact registry. Retain journals unchanged elsewhere. Use only validated grades or explicit finite enums; a DB origin does not make prose trustworthy. Loss of personalized prose coaching is an intentional compatibility tradeoff. |
| System instructions | Default or instance-owner custom prompt | Use public, versioned server instructions describing templates. Private custom prompt is never sent on v2. Existing owner configuration is retained but inactive for v2; no automatic semantic conversion. Later customization needs reviewed finite policy controls, not a private-text escape hatch. |
| Transport config | Base URL, API key, URL auth/query, HTTP client, model identifier | Transport adapter only. No inclusion in model messages, facts, display metadata, metrics labels or logs. Do not use string replacement as the input isolation proof. Provider sees the credential needed to authenticate; that is not a promise of secrecy from the provider. |
| Other app data | Users, PAT/JWT, account credentials, instance settings, news, files | Not loaded into review context. No model tools, arbitrary SQL/resource paths, retrieval or URL fetching. |

Use a request-local immutable fact snapshot with server-side user/account/trade binding. The renderer receives this snapshot and a static template catalog, never `VisionConfig`, private prompt or a general settings accessor. The model-facing object is separately constructed from selected facts; transport fields cannot accidentally enter it via whole-struct marshaling. Ownership grants access to a fact; it does not certify its economic accuracy.

`AnalyzeNews` uses the same Coach transport but has a separate fixed system prompt and free-form successful summary/reasoning/etc. It does not currently call trade `systemPrompt`. Error sanitization alone does not establish a safe successful-output contract for a malicious provider that knows its authentication headers. News analysis is a separately developable Core feature. Audit and isolate its shared credential/error/success-output paths independently; its availability requires its own accepted confidentiality boundary, not completion of Coach C. If its current shared path cannot be shown safe, temporarily contain that path while its independent remediation proceeds; manual news editing/review remains available. This design's initial templates cover trade Coach only, not arbitrary news prose.

## Versioned model selection schema (v1)

Each provider record is one complete JSON object; provider stream deltas are buffered internally until a complete record is available. No model-controlled event names or client envelopes.

```json
{
  "schema_version": 1,
  "template_id": "define_stop",
  "category": "risk",
  "priority": "warn",
  "fact_refs": ["f0"],
  "action": "record_stop",
  "params": {}
}
```

All seven keys are required; exactly these keys are allowed. Unknown keys at any depth, duplicate keys, missing/null fields, invalid UTF-8/unpaired surrogates, trailing data, arrays as root, unsupported versions and type coercion fail closed. Go `DisallowUnknownFields` alone is insufficient for duplicate keys or required presence: require explicit token/presence validation too. Strings must exactly match server-advertised finite IDs/enums; they are selections, never display text. Ref tokens are request-local keys allocated by the server (e.g. `f0`), not database IDs or paths. The model cannot create facts.

Initial approved catalog (all text maintained by the server):

| Template | Category / priority | Required fact and predicate | Action | Parameter contract |
|---|---|---|---|---|
| `define_stop` | risk / warn | `risk_defined` boolean = false for current trade | `record_stop` | Empty object |
| `review_execution` | execution / tip | Existing execution-grade integer 1..5 for current trade | `review_journal` | Empty object |
| `review_loss_streak` | session / warn | Current-account, entry-time loss-streak count >= 2 | `pause_review` | Empty object |
| `record_review` | process / tip | Current trade review grade absent | `grade_trade` | Empty object |

Category, priority and action must equal the catalog's values for that template; no independent covert string parameter. Model-selected confidence, monetary amounts, percentages, identifiers, prose, URLs, markdown and numeric digits to be printed are disallowed. v1 numeric parameters are deliberately an empty set: all displayed numbers are computed from server facts, with field-specific units, precision and domain validation. Any future numeric parameter needs a new schema/catalog version with finite integer range and demonstrated necessity, never an arbitrary float or decimal string. Reject NaN/infinity/overflow in fact computation; missing facts make the template ineligible. No template suggests or executes an order.

At most four findings, one per template. Duplicate selection rejects the run. Validate cumulative count/uniqueness and allowed combinations **before** each emission. Local hard limits: 4 KiB per assembled provider record, nesting <= 4, one fact reference per v1 template, 32 KiB assembled model output, 1 MiB provider wire, 64 KiB client event stream, existing configured request deadline capped by the server. Count bytes before allocation/append. Keep at most the current record and four validated findings; never unbounded concatenation. These are proposed limits needing boundary/fuzz/load tests, not measurements from this task.

Terminal provider record is exactly `{"schema_version":1,"kind":"end"}`; it is parsed through a separate exact-key terminal validator. No record after end. Provider transport EOF/error before explicit end is failure; provider adapter must also confirm normal completion. Non-streaming fallback returns exactly `{"schema_version":1,"findings":[...v1 records...]}` with the same local validation and limits. Providers without schema support may use JSON mode under the identical local contract; there is no arbitrary-text fallback.

## Fact resolver and server rendering

Authenticate JWT/PAT and load the trade with `user_id` before upstream access. Build the snapshot only from the requested trade and its authorized account. A reference lookup must match request user, account, trade/snapshot scope, fact type, template predicate and allowed field. Session aggregate references bind to the current account and requested trade's entry-time snapshot. Never dereference a model-provided global ID. Recheck authorization at persistence/readback; define revocation behavior for in-flight requests in implementation tests.

Render using static catalog templates, fixed enum labels and typed server facts; no `fmt` string from the provider. Plain text only, React text nodes, no HTML/markdown interpretation or model URLs. v1 templates need no arbitrary symbol/setup/journal strings: label the page using its independently authorized trade UI. Do not return raw provider model identifiers; use a server-defined public label such as “AI coach”. Server-generated finding/run IDs, sequence numbers and timestamps are separate from selections.

Locale is chosen from validated request preference with fallback `en`. Support the existing `en`, `zh-CN`, `zh-HK`, `ja`, `ko` locales. Add a small versioned backend catalog for Coach sentences; use the same catalog identifiers in Web tests and existing Lingui catalogs for UI status controls. Do not send translation instructions to the model. Missing template translation falls back to reviewed English text; missing template itself fails closed. Store template/catalog versions and typed facts, not locale-specific model prose. Locale switching rerenders a stored snapshot through that pinned catalog; keep historical catalogs. Number/currency formatting follows reviewed server locale rules and must not change accounting values. Web masking/privacy display remains an independent presentation requirement.

## Incremental SSE contract (client v2)

Negotiate `output_version=2` on existing trade Coach routes; new Web uses v2. Unknown versions return a generic 400 before provider access. Absent version uses only the safe compatibility adapter described in migration, never the old free-text generator.

| Event | Server payload and rule |
|---|---|
| `started` | version=2, server run ID, sequence=0; after authentication, access policy and owned trade lookup |
| `progress` | fixed stage: `preparing`, `analyzing`, `saving`; optional fixed-cadence heartbeat has no provider data. No invented completion percentage or echoed model status. |
| `finding` | server finding ID/sequence, public template/category/priority/action, server-rendered headline/detail, provisional=true; emit only after complete-record/schema/reference/predicate/cumulative validation |
| `completed` | persisted review ID, created timestamp, authoritative validated findings, fixed next action selected by server catalog priority, provisional=false; after successful transaction only |
| `error` | allowlisted code (`generation_failed`, `timeout`, `canceled`, `save_failed`, `unavailable`) and catalog message; terminal, no raw cause, headers or diagnostics |

Flow: `started → progress* → (finding/progress)* → completed OR error`, one terminal event. A finding can arrive while the mock provider withholds end. No partial JSON strings, unknown field names, provider errors, raw finish reasons or usage metadata go to clients. Set SSE no-cache/no-store and buffering controls; test deployed proxy flushing. Bound heartbeat count by deadline.

Findings are provisional in **review completion/persistence** only; confidentiality approval is final before flush. If a later record fails, previous approved text remains safe; UI clears provisional findings and shows a generic failure, with no completed review saved. No transactional fiction that the error retracts bytes. Global validation cannot be used to approve confidentiality after emission. If future templates require whole-review facts, hold those findings until their own prerequisites are established, while server progress continues.

Cancel upstream on client disconnect/write failure; stop emitting and do not persist an incomplete run. Timeout covers retries, fallback, parsing and persistence budget. Do not restart provider generation after any finding has been emitted within the same run; a user retry gets a new run. Use a user/trade-scoped idempotency key for safe commit/readback recovery when a connection dies after a transaction commits. No automatic repeated billable call on SSE reconnect; no Last-Event-ID replay in v2 initially. UI may read owned history to reconcile a completed commit. Authorization/revocation failure before a future emission terminates safely; already authorized delivered facts cannot be recalled.

## Guarantees and limitations

Under correct authorization, immutable typed fact construction and reviewed catalogs, arbitrary model bytes cannot populate client text, stored display fields or log diagnostics. Unknown/forged references cannot expose another user's/account's values. Private configuration has no renderable data path. Each emitted event obeys the same rule, including pre-error findings. Stored rows are not trusted merely because they were once validated; readback repeats version/reference/type/catalog checks.

This is not full information-theoretic noninterference. A malicious provider knows its transport credential and can choose among allowed findings, omit them, vary timing or fail; those finite choices may form an indirect channel over repeated calls. Minimize this channel with no private prompt input, deterministic catalog metadata, strict predicates, count/dedup bounds, fixed server progress and request quotas. Rate limits are mitigation, not proof. A stronger requirement forbidding all provider-dependent leakage would require deterministic server findings or a provider architecture in which the model cannot access confidential state; C does not claim that property.

User financial facts intentionally disclosed to the model remain subject to provider trust/retention. Fact accuracy, misleading-but-valid selection, coaching usefulness and translation quality are quality issues requiring separate tests. A compromised DB/host/operator/catalog or authorized user's own data sharing is outside the client isolation guarantee. No current design claim covers vendor privacy, arbitrary deployment proxies or unvalidated storage engines.

## Disposable validation

`python3 /private/tmp/tradelens-295-poc/poc.py`: 16 tests PASS. The sketch implements exact fields, template predicate, owner/account/trade reference binding, server text, event ordering, failure and incremental iteration. It has no real SSE socket or persistence; its iteration evidence must not be substituted for future HTTP flush, transactional or race acceptance. [Roadmap](phase1-security-roadmap.md) defines those gates.
