# Fork Development Policy

## Purpose

This document defines the development scope and compatibility policy for the private fork
`HoesenBruce/TraderMemos-Private`.

The goal is to support fork-specific trading workflows while keeping future upstream merges and
possible mobile reactivation reasonably low-risk.

## Current platform scope

The fork currently follows a **Web-first, mobile-preserving** strategy.

### Active delivery target

The Web application is the primary product surface for new fork-specific features.

A Web-focused feature can be considered complete when its API/domain behavior and Web behavior
meet the issue acceptance criteria. iOS and Android validation are not required unless the issue
explicitly includes mobile support.

### Mobile status

The existing Expo mobile client remains part of the repository, but it is currently in
**maintenance-compatible, not actively validated** status.

This means:

- do not remove the mobile application merely because it is not currently used;
- do not intentionally break mobile contracts when a reasonable shared implementation is
  available;
- do not require mobile-specific UI work for ordinary Web-focused issues;
- do not require Android emulator, Android real-device, iOS simulator, or iOS real-device testing
  for Web-focused feature completion;
- document known mobile incompatibilities instead of silently ignoring them;
- avoid unnecessary platform-specific assumptions in shared API, domain, persistence, market-data,
  importer, and analytics layers.

## Future mobile reactivation

If iOS or Android support becomes a product goal again, use the existing mobile client as the
starting point.

The default reactivation process should be:

1. restore mobile to the active acceptance scope;
2. run compatibility and end-to-end testing against the current API;
3. identify regressions and unsupported flows accumulated during the Web-first period;
4. fix shared-contract incompatibilities first;
5. add platform-specific adaptations where required;
6. add or restore mobile-specific automated and manual validation.

Do **not** assume the mobile client must be rebuilt from scratch. A rewrite should require a
separate architectural justification.

## Shared-code rule

Fork-specific development should prefer reusable shared behavior when it is reasonable to do so.

Examples include:

- API contracts;
- domain models;
- importer semantics;
- market-data abstractions;
- analytics calculations;
- symbol/instrument metadata;
- validation rules.

A Web implementation may use browser-specific presentation or interaction code, but shared
business behavior should not become Web-only without a clear reason.

## Web-first acceptance policy

For ordinary fork issues:

- relevant API tests must pass;
- relevant Web tests must pass;
- changed Web workflows should be exercised end to end;
- loading, empty, error, and important state-transition behavior should be tested where applicable;
- lint and type checks should pass for affected components;
- mobile testing is optional unless explicitly required by the issue;
- any known mobile regression or incompatibility introduced by the change must be documented.

`AGENTS.md` contains the operational UI verification rules.

## Upstream compatibility

This is a private fork, but changes should still minimize avoidable conflict with upstream.

Prefer:

- additive changes over destructive rewrites;
- focused modules over broad refactors;
- extension of existing abstractions over parallel duplicate stacks;
- preserving existing public API semantics where possible;
- documenting fork-only behavior clearly;
- isolating provider- or broker-specific logic behind interfaces;
- keeping unrelated formatting or cleanup out of feature changes.

Before finishing a substantial feature, review the final diff for unnecessary merge risk.

## Issue scope guidance

An issue should explicitly identify the platform scope when UI work is involved.

Recommended wording for current fork work:

> Platform scope: Web is the active delivery target. iOS and Android are not required acceptance
> targets for this issue. Shared API/domain changes should remain reasonably compatible with
> future mobile reactivation, and known mobile incompatibilities must be documented.

Do not create Android/iOS implementation or validation sub-issues merely because the upstream
project supports those platforms.

If a mobile-only issue already exists and is no longer part of the current roadmap, close it as
`not planned` rather than deleting it. This preserves context for future reactivation.

## Scope changes

When mobile becomes active again, update this document first or as part of the issue that changes
the platform policy. Acceptance criteria and agent instructions should then be updated together so
that repository documentation does not contradict the active development scope.
