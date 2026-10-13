# Coach security migration and compatibility plan

Status: proposed, no migration or feature setting executed. Core D1 and Coach D2 are separate gates; see the roadmap. Applies to main `25fe2f97d41053f2eb8e84362e28ad950c35f55e`; [architecture C contract](coach-secure-output-design.md).

## Current contracts and affected surfaces

Current `coach_reviews` stores `id,user_id,trade_id,model,notes,next_action,created_at`; notes and next action are free text. SQL exists for SQLite and PostgreSQL. `saveCoachReview` writes after successful generation; the current handler can return success without a stored ID when persistence fails. Current history deserializes notes; #295 additionally checks against current configuration, which cannot prove safety after secret rotation.

Current Web `trades.ts` consumes `note/done/error`; `trades.coach.types.ts` describes headline/detail/priority/next_action and optional model metadata. `useTradeCoach.ts` clears previews on settled and reads latest history; settings are currently owner-only and the hook uses settings.enabled as availability. Introduce a non-sensitive effective capability response for authenticated users, rather than broadening access to private settings. Update UI history handling so a withheld legacy record is not mistaken for a completed review. News AI shares Coach transport and needs explicit gating. Existing mobile contract stays usable through the safe legacy adapter or explicit unavailable response; mobile behavior remains unvalidated/outside scope.

## Storage decision

Prefer a separate additive `coach_reviews_v2` table to prevent older binaries from automatically reading v2 data as free text. Fields: server ID, user_id, account_id, trade_id, schema_version=1, catalog_version, server created_at, idempotency_key, validated selections JSON, minimal typed fact snapshot JSON. Unique `(user_id,trade_id,idempotency_key)`; owner/trade/account referential checks and existing deletion/cascade policy. Public model label is catalog-derived; no provider config, prompt, key, diagnostic, raw payload, arbitrary model metadata or transport response enters this table.

Persist only completed validated runs in one transaction, with authorization/trade/account binding rechecked; commit precedes completed event. Failure/cancellation before commit writes no review; journal/trade financial rows never change. A disconnect immediately after commit can leave a valid review without delivered completion: history/idempotency reconciliation is intentional and testable. Do not claim zero writes after every disconnect regardless of commit timing.

Implement matching SQLite/PostgreSQL migrations, sqlc queries/generated stores and schema docs; no change to accounting/import/history rows. Do not make an arbitrary new JSON blob compatible by falling back to legacy parsing on errors. Unknown schema/catalog version is withheld, never interpreted as v1 or free text. Snapshot facts preserve historical meaning despite later journal/risk edits; current ownership and existence still gate readback. Retain a catalog version for the lifetime of its rows, or return a fixed unavailable placeholder when missing.

## Dual read, single safe write

1. New generation writes v2 only. Read both tables by authenticated user/trade with bounded merged pagination/order. Never run a migration that asks a model to “sanitize” historical text.
2. v2 readback validates all keys, limits, catalog, exact predicates, and stored user/account/trade bindings, then rerenders typed snapshot values. Stored display text is not trusted. Corrupt/unknown rows return a server-defined unavailable status with no raw values; readback never repairs them silently.
3. Legacy rows remain byte-for-byte unchanged. Default API returns only server-validated ID/time and `status=legacy_withheld` plus a fixed localized explanation. Do not expose legacy notes, next_action, model or snippets to members or owners through normal Coach APIs. Do not use current-secret recognition as an allow-to-render decision.
4. Do not add a raw legacy download route as a compatibility shortcut. Existing trusted operator backups remain an operational recovery path; any future export must be separately authorized, scoped and labeled sensitive. Regeneration from current allowed facts is an explicit user action and creates a new v2 row; it never overwrites the legacy record or claims to reproduce old advice.
5. User-owned arbitrary financial/journal text displayed elsewhere is not silently rewritten. Audit exports/shares/aggregations for Coach fields so no alternate route bypasses withheld-history policy; absence of such a route must be recorded by source inventory.

## API and Web rollout

Use `output_version=2` for v2 events and history. Version the **client** envelope separately from model schema version=1. Include `status` on history entries: `complete`, `legacy_withheld`, `unavailable`. Blocking v2 returns only validated rendered findings and a persisted completion result; save failure is a generic error.

During compatibility window, absent version invokes a server adapter from validated v2 data to existing `notes/next_action/source` DTOs and `note/done/error` events. It never invokes the old generator/guard. Ignore v2 progress for old clients, use server-generated note IDs/tone/priority, and send fixed generic errors with the existing HTTP/status contract. Legacy history must be adapted to a fixed placeholder note or omitted with a documented count; choose placeholder for consistent visible history. Omit raw `model`. Disabled capability returns the existing `source:off, notes:[]` shape or documented generic disabled error; enforce before any provider access. Test old client fixtures against this adapter. No automatic client fallback to the original free-text endpoint.

New Web updates API types, parser and hook together: runtime validate server envelopes, run/sequence handling, required terminal event, explicit provisional state, error/cancel cleanup, history placeholder, locale switching, privacy formatting and generic errors without unhandled rejected Promise. Do not ignore malformed v2 frames and pretend success. Include abort on navigation/reset; failed run retries cannot append old previews or duplicate completed rows. Authenticated capability contains only availability and supported output versions, never owner configuration.

Deploy compatible backend with effective access disabled first; deploy Web; validate real API, storage, auth, proxy flush and cold caches; enable approved scope only after independent acceptance and explicit operational authorization. Old open tabs/service worker cache must not restore unsafe display from cached Coach responses: no-store headers, cache invalidation/versioning and logout/user change clearing are required. Already copied historical responses cannot be retracted.

## Temporary containment (separate proposed Issue E)

A UI toggle is not a security boundary. Introduce one central backend effective-access policy applied to all trade JSON/SSE/fallback, history and shared Coach-powered News analysis entry points. Default `disabled`; No setting, including an owner-only option, may re-enable the unsafe legacy generator. Future enablement requires a v2-capable build, approved B contract, passing independent D2 acceptance for the deployed candidate, and explicit operator configuration. A requested enabled value on an incapable/unaccepted build fails closed. Any future role-restricted policy must reread authoritative DB identity for JWT/PAT. Environment defaults, DB settings, hot reload and job/direct service paths must not bypass policy. Demotion/revocation during a stream must terminate before further emission under a defined recheck policy; global disable cancels existing runs. History is withheld even with generation off; no provider call or unsafe readback for denied callers.

AI Coach remains planned and supported. Preserve legacy code/data for migration; containment is not permanent removal. This policy applies to single-user and multi-user release configurations alike: no owner-only legacy bypass. A configuration name is intentionally not fixed by this planning task; E must document its precedence, safe default, capability check and fail-closed semantics. Re-enablement of accepted v2 also requires separately authorized operational action; changing an environment variable alone cannot satisfy security acceptance.

News analysis remains independently developable as a Core feature. Inventory its shared Coach transport, then give it a separate safe capability boundary: provider errors, successful model output, credentials, logs and persistence must be accepted for the exposed News path. If unsafe, contain that shared path pending its own remediation; it need not wait for Coach C or D2. Keep manual News workflows functional. Core accounting/import, valuations, analytics, journal/plans, Review Inbox, multi-account and Setup analytics/Mistake Tax (#274) have no dependency on Coach recovery.

## Rollback and migration risk

Before migrations use an authorized consistent backup and restore rehearsal on disposable storage. No production database accessed in this task. Additive migrations do not drop/modify legacy rows; validate row counts and hashes before/after, and idempotent upgrade on SQLite/PostgreSQL. Never destructively down-migrate v2 data to satisfy an old binary.

Rollback target must contain the containment/history patch. Disable Coach and any independently unaccepted shared AI path, stop their in-flight jobs/streams, then roll back app with v2 table retained. Do not roll back directly to known unsafe main or expose legacy history. A fully older binary requires network/process isolation until a safe patch is restored. Restoring backup is separately authorized and must account for writes since backup; no automatic restore.

Principal risks: degraded personalized coaching and withheld historical display; limited initial template coverage; translation/catalog drift; older clients/settings capability assumptions; duplicate writes around commit/disconnect; account/trade deletion races; model/provider IDs leaking through telemetry; stale service-worker/browser data; unsupported provider structured streaming; custom prompts no longer applied. Resolve with explicit acceptance, not private-text compatibility exceptions. Re-enabling old free-text paths is not a rollback option.

## Future migration acceptance

- Synthetic legacy rows with known, encoded and rotated/unknown configuration values return placeholders without mutation under JWT/PAT and owner sessions; raw history/model values absent from responses, headers, logs, exports and caches.
- v2 valid rows persist once, re-render consistently across five locales, survive restart, and stay scoped to user/account/trade; corrupt/unknown versions fail closed.
- SQLite and PostgreSQL upgrade/rollback rehearsal preserves all legacy hashes and unrelated journal/trade rows; older safe binary cannot emit raw history.
- Client old/new fixtures, open stale tab, service worker, reset/re-entry/cancel/retry, missing catalog, deleted trade/account, revoked identity and commit-disconnect cases pass.
- Real proxy shows progress/finding before completion and safe terminal failure. Deployment environment and exact source SHA are recorded separately from local mock results.
