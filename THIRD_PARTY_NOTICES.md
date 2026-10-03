# Third-party notices

TradeLens remains licensed under [GNU AGPL-3.0](LICENSE). The upstream
TraderMemos attribution in [NOTICE](NOTICE) remains applicable. The following
notices preserve the terms of identified third-party material; they do not
relicense TradeLens or cover every ordinary package-manager dependency.

Review baseline: main `693545d706ce4b1f4f22fd869fa8cc2fbead88ce`, tree
`a1e5d61473d3008895499e38ce7f3d506273e36d` (Issue #226, related to #225).
Paths below identify copied source and local adaptations, not merely imports
of third-party dependencies. Existing source links and comments are retained.

## Distribution

- The complete MIT and ISC notices for copied Web code are in
  [web/public/third-party-notices.txt](web/public/third-party-notices.txt).
  Keep that file with source copies and compiled Web distributions. Vite's
  public-directory copying includes it at `/third-party-notices.txt`.
- The complete font copyright and OFL-1.1 texts are in
  [marketing/public/font-notices.txt](marketing/public/font-notices.txt).
  Keep that file with marketing output containing the fonts. Next.js serves
  it at `/font-notices.txt`; a custom/standalone deployment must retain `public/`
  alongside its generated static font assets.
- When redistributing an individual copied component outside this repository,
  include its relevant copyright and license text with that copy.

## Confirmed copied Web source

All three MIT families below were `CONFIRMED_NOTICE_REQUIRED`: the original
repository had upstream links, but no complete third-party MIT notices.
The linked public notice file now supplies the copyright, permission and
warranty text. No upstream per-file notice has been removed.

### shadcn/ui — MIT

Copyright (c) 2023 shadcn.
[Authoritative license](https://github.com/shadcn-ui/ui/blob/main/LICENSE.md).

Affected source/adaptations:

- `web/src/components/ui/item.tsx`:
  [Base UI Item](https://github.com/shadcn-ui/ui/blob/main/apps/v4/registry/bases/base/ui/item.tsx).
- `web/src/components/ToneToggle.tsx`:
  [Base UI Toggle](https://github.com/shadcn-ui/ui/blob/main/apps/v4/registry/bases/base/ui/toggle.tsx).
- `web/src/components/theme-provider.tsx`:
  [Vite theme-provider example](https://ui.shadcn.com/docs/dark-mode/vite),
  extended locally with system-theme updates and browser theme-color handling.
- `web/src/lib/hooks/use-mobile.ts`:
  [use-mobile hook](https://github.com/shadcn-ui/ui/blob/main/apps/v4/registry/new-york-v4/hooks/use-mobile.ts),
  subsequently adapted to `useSyncExternalStore`.
- shadcn-derived primitive structure retained through the coss/ReUI adaptations
  listed below, and theme-token setup in `web/src/global.css`.

The small `web/src/lib/cn.ts` helper is a conventional `clsx`/`tailwind-merge`
composition; the notice also preserves upstream attribution for this shared
registry helper without treating dependency imports as vendored packages.

### coss ui / Origin UI — MIT

Copyright (c) 2025 coss.com. Originally Copyright (c) 2025 Origin UI.

coss is a mixed-license repository. Its
[LICENSING.md](https://github.com/cosscom/coss/blob/8d1942467029030ed97b90d2e89591901448072e/LICENSING.md)
explicitly includes `apps/ui/` and `apps/origin/` in its MIT scope, and the
[apps/ui README](https://github.com/cosscom/coss/blob/8d1942467029030ed97b90d2e89591901448072e/apps/ui/README.md)
confirms MIT. The preserved MIT/copyright text comes from
[apps/origin/LICENSE.md](https://github.com/cosscom/coss/blob/8d1942467029030ed97b90d2e89591901448072e/apps/origin/LICENSE.md).
This entry does not apply the default coss AGPL license to those MIT directories.

Affected `web/src/components/ui/` files:

- `accordion.tsx`
- `alert.tsx`
- `autocomplete.tsx`
- `button-group.tsx`
- `button.tsx`
- `calendar.tsx`
- `card.tsx`
- `combobox.tsx`
- `dialog.tsx`
- `empty.tsx`
- `field.tsx`
- `input-group.tsx`
- `input.tsx`
- `kbd.tsx`
- `label.tsx`
- `menu.tsx`
- `native-select.tsx`
- `number-field.tsx`
- `popover.tsx`
- `scroll-area.tsx`
- `select.tsx`
- `separator.tsx`
- `skeleton.tsx`
- `slider.tsx`
- `spinner.tsx`
- `switch.tsx`
- `table.tsx`
- `tabs.tsx`
- `textarea.tsx`
- `toggle-group.tsx`
- `toggle.tsx`
- `tooltip.tsx`

The matching upstream source is the
[apps/ui registry](https://github.com/cosscom/coss/tree/8d1942467029030ed97b90d2e89591901448072e/apps/ui/registry/default/ui).
The historical source comparison and introduction commit
`1007b2ae78b599f0279495df1519f95120434cc4` establish registry adoption; the
upstream snapshot is comparison evidence, not a claim that every local file
is an unmodified copy of that precise revision. `button-group.tsx` and
`native-select.tsx` retain local adaptations of coss field chrome rather than
matching registry filenames. Related adapted styling is retained in
`web/src/components/field-styles.ts`, `web/src/components/surface-styles.ts`
and `web/src/global.css`.

### ReUI free components and examples — MIT

Copyright (c) 2025 Keenthemes Inc.
[Authoritative license](https://github.com/keenthemes/reui/blob/39c1f6849ab9a377896a9bcfc034a42546c017c5/LICENSE.md).
The affected material is present in this MIT repository's free registry,
not evidence of a license for a separate Pro catalog or agent-skill package.

Affected source/adaptations:

- `web/src/components/reui/alert.tsx`, `web/src/components/reui/badge.tsx` and
  `web/src/components/filters.tsx` from
  [Base UI ReUI primitives](https://github.com/keenthemes/reui/tree/39c1f6849ab9a377896a9bcfc034a42546c017c5/registry-reui/bases/base/reui).
- `web/src/hooks/use-file-upload.ts` from the
  [file-upload hook](https://github.com/keenthemes/reui/blob/39c1f6849ab9a377896a9bcfc034a42546c017c5/registry-reui/bases/base/hooks/use-file-upload.ts).
  Origin UI also provides this hook under its MIT license; both notices are retained.
- `web/src/components/examples/c-file-upload-6.tsx` from the
  [file-upload example](https://github.com/keenthemes/reui/blob/39c1f6849ab9a377896a9bcfc034a42546c017c5/registry-reui/bases/base/components/file-upload/c-file-upload-6.tsx),
  and its locally adapted import surface `web/src/components/CsvDropZone.tsx`.
- `web/src/components/examples/c-skeleton-2.tsx`, `c-skeleton-3.tsx`,
  `c-skeleton-4.tsx`, `c-skeleton-5.tsx`, `c-skeleton-6.tsx` and `c-skeleton-7.tsx`
  from the [skeleton examples](https://github.com/keenthemes/reui/tree/39c1f6849ab9a377896a9bcfc034a42546c017c5/registry-reui/bases/base/components/skeleton).
  Related local compositions are `web/src/components/skeletons/card-skeleton.tsx`,
  `form-skeleton.tsx`, `list-skeleton.tsx`, `stats-row-skeleton.tsx`,
  `table-skeleton.tsx` and `text-block-skeleton.tsx` in that directory.

### Lucide 0.469.0 GitHub icon — ISC

`CONFIRMED_NOTICE_REQUIRED`, now supplied in the Web public notice file.
Affected path: `web/src/components/icons/github.tsx`. Its two SVG paths match
[icons/github.svg at 0.469.0](https://github.com/lucide-icons/lucide/blob/0.469.0/icons/github.svg).
The React wrapper is locally adapted. The exact version's
[LICENSE](https://github.com/lucide-icons/lucide/blob/0.469.0/LICENSE)
credits Cole Bemis 2013–2022 for Feather portions and Lucide Contributors 2022
for all other portions. The complete upstream ISC material, including that
copyright wording, is preserved; the icon is not replaced.

## Generated exchange calendar output

`NO_REPOSITORY_NOTICE_REQUIRED` for the generator's Apache license;
`CALENDAR_OUTPUT_RIGHTS_CLEAR` for the reviewed factual CSV output.

Affected paths: `api/internal/marketdata/calendars/JP.csv` and `US.csv`.
[api/scripts/export_calendars.py](api/scripts/export_calendars.py) uses
`exchange-calendars==4.13.2` to query XTKS/XNYS schedules for 2020–2030 and emits
only `date,open,close` facts. It does not copy the library's source, comments,
data-file layout or documentation. The source distribution contains these
output rows, not the Python dependency. No separate output-license restriction
was identified in the reviewed upstream license. This conclusion concerns
these bounded factual schedules, not a blanket permission for other upstream
material or exchange market-data products.

The library's [Apache-2.0 license at 4.13.2](https://github.com/gerrymanoim/exchange_calendars/blob/4.13.2/LICENSE)
applies to the library; using it to calculate schedules does not by itself
assign Apache-2.0 to these CSVs. Existing generator and official-exchange
attribution remains in the [calendar README](api/internal/marketdata/calendars/README.md).
That provenance is retained without inventing a CSV license designation.

## Marketing fonts

No `.ttf`, `.otf`, `.woff`, `.woff2` or `.eot` font binaries are tracked.
The source-only font references are `NO_REPOSITORY_NOTICE_REQUIRED` for font
binary redistribution. Build output containing downloaded/self-hosted fonts
is `CONFIRMED_NOTICE_REQUIRED`; the marketing public notice now supplies it.

`marketing/app/[lang]/layout.tsx` uses `next/font/google` with the Latin subset:

| Font | Copyright | License and authoritative source |
| --- | --- | --- |
| Bricolage Grotesque | Copyright 2022 The Bricolage Grotesque Project Authors | [OFL-1.1](https://github.com/google/fonts/blob/main/ofl/bricolagegrotesque/OFL.txt); [upstream project](https://github.com/ateliertriay/bricolage) |
| JetBrains Mono | Copyright 2020 The JetBrains Mono Project Authors | [OFL-1.1](https://github.com/google/fonts/blob/main/ofl/jetbrainsmono/OFL.txt); [upstream project](https://github.com/JetBrains/JetBrainsMono) |

The complete copyright/license texts accompany builds via the marketing public
file. OFL obligations attach to redistributed font software, including subsets;
they do not relicense rendered documents or TradeLens. Do not sell fonts by
themselves, remove their notices or apply another license to them. Any future
font modification must also check the applicable reserved-name conditions.
No build, font download, deployment or font-binary addition is part of this change.

## External ReUI development resources

`REUI_DOCS_LICENSE_UNCERTAIN = RESOLVED_BY_REMOVAL`. The copied ReUI skill
bundle and generated Cursor rule have been removed from the current repository
content under Issue #226. No redistribution license is assigned to that removed
documentation. The confirmed MIT code and examples listed above remain with
their license notices. ReUI skills/documentation are external resources only;
see the original TradeLens note in [CONTRIBUTING.md](CONTRIBUTING.md#optional-reui-resources).
