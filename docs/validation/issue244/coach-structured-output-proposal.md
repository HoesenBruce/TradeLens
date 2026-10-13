# Proposed follow-up Issue: render Coach notes from trusted structured results

Status: proposal only; no Issue created and no implementation in #295.

## Problem

Instance credentials/private instructions reach an external provider for legitimate authentication/configuration. A free-text review can echo that configuration. The current bounded output guard handles known representations but depends on a growing transformation/fragment contract, with conservative false positives. Server-controlled rendering can remove that confidentiality dependence.

## Proposed scope

- Provider output contains a finite note template ID, tone enum and references to owned trade facts; it contains no renderable arbitrary text. Unknown fields/IDs and excessive output are rejected.
- Server resolves facts using the authenticated user/trade, verifies references and finite numeric/enumerated values, then renders a localized template. Secrets/configuration and provider diagnostics are never eligible facts.
- Persist only validated structured results or the server-rendered review. Readback applies ownership/version validation; document migration separately without rewriting existing financial/journal data.
- Separate transport credentials, private provider instructions and trade context. Never put transport keys in model messages; assess removal of encoded accidental input credentials independently.
- Preserve completed-note incremental streaming, `note`/`done`/generic `error`, timeout/cancel, retry and the existing Web Coach interaction. Do not wait for the entire review before rendering a validated note.

## Acceptance criteria

1. JWT/PAT consumers cannot cause arbitrary provider text, headers, credentials or private instructions to appear in notes, next action, model metadata, history or logs.
2. Real-route provider fixtures include keys/prompts encoded or transformed into fields, forged foreign fact references and unknown schema fields; rejection causes no unsafe persistence or unrelated writes.
3. Distinct concurrent users receive only their own facts/reviews. Safe structured notes appear before final completion; disconnect/timeout/retry and event order pass.
4. Desktop Web behavior and supported locales remain usable; full Go/race/Web/Chromium gates pass on the final head.
5. Define a versioned finite schema, limits, supported templates and compatibility policy. Financial/journal/import behavior is preserved; operator-only historical cleanup requires a separate controlled plan.

## Tradeoff and rollout

Less free-form coaching flexibility in exchange for a stronger output boundary. Start with a small useful template set matching existing notes, retain the current guard as defense in depth during migration, and obtain independent security review. Real-provider and intended storage/proxy deployments require their own validation. Do not claim provider-side prompt confidentiality: the provider still receives permitted private instructions.
