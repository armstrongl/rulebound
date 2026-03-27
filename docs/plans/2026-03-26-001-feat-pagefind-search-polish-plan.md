---
title: "feat: Improve Pagefind search relevance with weight boosting, metadata, and filters"
type: feat
status: active
date: 2026-03-26
origin: docs/brainstorms/2026-03-26-pagefind-search-polish-requirements.md
deepened: 2026-03-26
---

# feat: Improve Pagefind search relevance with weight boosting, metadata, and filters

## Overview

Add Pagefind weight attributes, structured metadata, and filter UI to the generated style guide site so that search results rank correctly, show structured metadata per result, and support faceted filtering by severity, type, and category.

## Problem Frame

Writers searching for a specific rule get irrelevant results because Pagefind indexes all content equally. Search results lack structured metadata, and the existing `data-pagefind-filter` attributes are not exposed in the search UI. (see origin: docs/brainstorms/2026-03-26-pagefind-search-polish-requirements.md)

## Requirements Trace

- R1. Rule page titles receive higher Pagefind weight than body content
- R2. Rule message text receives moderate weight boost (higher than body, lower than title)
- R3. Search results display severity and rule type metadata per result
- R4. Search UI exposes filter controls for severity, type, and category facets
- R5. Guideline pages follow the same pattern when they ship (deferred)

## Scope Boundaries

- No Go build pipeline or parser changes -- purely Hugo template and Pagefind configuration
- No custom JavaScript beyond Pagefind's built-in UI configuration options
- R5 is deferred until the guidelines content type ships

## Context & Research

### Relevant Code and Patterns

- `internal/hugo/theme/layouts/_default/single.html` -- rule page template; already has `data-pagefind-body`, `data-pagefind-meta="title"`, and three `data-pagefind-filter` attributes (severity, type, category)
- `internal/hugo/theme/layouts/partials/search.html` -- Pagefind UI initialization; currently uses default `PagefindUI` config with only `showSubResults: true`
- `internal/hugo/theme/layouts/index.html` -- homepage; no Pagefind attributes needed

### Pagefind API Research

**Weights (R1, R2):**
- `data-pagefind-weight` accepts floats 0.0--10.0, default 1.0
- Scale is quadratic: weight 2.0 = ~4x impact, 3.0 = ~9x
- Pagefind v1.0+ already auto-boosts headings (`<h1>`--`<h6>`) internally
- Recommended: modest explicit values since headings are already boosted

**Meta (R3):**
- `data-pagefind-meta` supports inline-value syntax: `data-pagefind-meta="severity:error, type:substitution"`
- Standard PagefindUI renders only `meta.title` and `meta.image` by default; its `processResult` callback can modify data but not HTML structure
- Custom meta display requires the Component UI (formerly "Modular UI"), which supports Liquid-like templates: `{{ meta.severity }}` for escaped output, `{{+ excerpt +}}` for unescaped HTML

**Filters (R4):**
- Standard PagefindUI automatically shows filter checkboxes when filter data exists in the index
- Multi-select within a group (OR logic), AND across groups
- `openFilters: ["severity", "type"]` expands specified groups by default
- `showEmptyFilters: false` hides zero-count filter values

## Key Technical Decisions

- **Weight values**: `<h1>` gets no explicit weight (Pagefind auto-boosts headings). Message `<p>` gets `data-pagefind-weight="2"` (~4x body text). Code blocks in technical details get `data-pagefind-weight="0.5"` (~0.25x) to prevent code from dominating.

- **Per-result metadata display (R3)**: Switch from `pagefind-ui.js` to the Component UI (`pagefind-component-ui.js` / `pagefind-component-ui.css`). Note: despite the filename, the JS global namespace is `PagefindModularUI` (a naming inconsistency from a rename). The Component UI ships alongside the standard UI in `/pagefind/` from the standard `pagefind` CLI run — no extra installation needed. The script tag must use `type="module"`. This enables custom result templates with Liquid-like syntax where `{{ meta.severity }}` and `{{ meta.type }}` render per-result metadata as badges. The standard `PagefindUI.processResult` callback was considered but rejected: it can modify data but not the HTML structure of results, so it cannot add badge markup.

- **Content-type filter for future guidelines**: Add `data-pagefind-filter="content_type:rule"` to the rule template now. When guideline pages ship, they will add `content_type:guideline`. This gives the filter UI a way to distinguish rules from guidelines.

## Open Questions

### Resolved During Planning

- **Q: What weight values?** Use no explicit weight on `<h1>` (auto-boosted), `2.0` on message, `0.5` on code blocks. Based on Pagefind's quadratic scale and heading auto-boost.
- **Q: Multi-select filters?** Standard PagefindUI uses multi-select checkboxes by default. No config change needed.
- **Q: How to show meta in results?** Use Pagefind Component UI (file: `pagefind-component-ui.js`, global: `PagefindModularUI`) with Liquid-like result templates. The standard `PagefindUI.processResult` can modify data but not HTML structure, so it cannot add badge markup. The Component UI's `<script type="text/pagefind-template">` blocks support `{{ meta.severity }}` and `{{ meta.type }}`.

### Deferred to Implementation

- CSS styling for severity/type badges within search results -- reuse existing `.severity-badge` and `.type-badge` class patterns
- Exact container element IDs and layout structure for the Component UI's `Input`, `ResultList`, `FilterPills`, and `Summary` components within the `#search` mount point

## Implementation Units

- [ ] **Unit 1: Add weight and meta attributes to single.html**

  **Goal:** Boost message weight, downweight code blocks, and capture severity/type as indexed metadata.

  **Requirements:** R1, R2, R3 (data capture)

  **Dependencies:** None

  **Files:**
  - Modify: `internal/hugo/theme/layouts/_default/single.html`

  **Approach:**
  - Add `data-pagefind-weight="2"` to the `<p class="rule-message">` element (line 46)
  - Add `data-pagefind-weight="0.5"` to the `<details class="technical-details">` element (line 65)
  - Add a hidden `<span>` with `data-pagefind-meta="severity:{{ $p.level | default \"suggestion\" }}, type:{{ $p.extends }}"` inside the `<article>` element
  - Add `data-pagefind-filter="content_type:rule"` as a hidden span for future rule/guideline distinction

  **Patterns to follow:**
  - Existing hidden `<span data-pagefind-filter="severity" hidden>` pattern on line 11

  **Test scenarios:**
  - Build the site with the Microsoft testdata package and run Pagefind. Search for an exact rule name (e.g., "Avoid") -- it should be the top result.
  - Search for a term that appears in both a rule title and a different rule's body text -- the title match should rank higher.
  - Search for a term from a code block -- it should rank lower than the same term in body text.

  **Verification:**
  - Rule pages include `data-pagefind-weight` attributes in the generated HTML
  - Rule pages include `data-pagefind-meta` with severity and type values
  - Pagefind index builds without errors

- [ ] **Unit 2: Switch search UI to Component UI with filters and result metadata**

  **Goal:** Enable filter controls and per-result severity/type display.

  **Requirements:** R3 (display), R4

  **Dependencies:** Unit 1 (meta/filter attributes must exist in templates)

  **Files:**
  - Modify: `internal/hugo/theme/layouts/partials/search.html`

  **Approach:**
  - Replace `pagefind-ui.js` and `pagefind-ui.css` imports with `pagefind-component-ui.js` (with `type="module"`) and `pagefind-component-ui.css`
  - Initialize using `PagefindModularUI.Instance()` (note: global is `PagefindModularUI` despite the `component` filename -- a Pagefind naming inconsistency)
  - Add components via `instance.add()`: `Input` (search box), `FilterPills` for severity/type/category, `ResultList` with a custom `<script type="text/pagefind-template">` block, and `Summary` for result count
  - The result template uses Liquid-like syntax: `{{ meta.title }}` for the heading, `{{ meta.severity }}` and `{{ meta.type }}` rendered as badge spans, `{{+ excerpt +}}` (unescaped) for the highlighted snippet
  - Guard: wrap initialization in `typeof PagefindModularUI !== 'undefined'` check inside `DOMContentLoaded`, consistent with existing pattern
  - Create container `<div>` elements within the `#search` mount point for each component

  **Patterns to follow:**
  - Existing severity-badge and type-badge CSS classes from `single.html`
  - The current `PagefindUI` initialization pattern (DOMContentLoaded + typeof guard) as the starting point

  **Test scenarios:**
  - Filters for severity, type, and category appear in the search UI
  - Selecting "error" in the severity filter narrows results to error-level rules only
  - Selecting multiple severity values shows the union (OR logic)
  - Each search result displays severity and type as inline badges next to the title
  - Search still works when no filters are selected (all results shown)
  - If `pagefind-component-ui.js` is missing (older Pagefind), the search area degrades gracefully (no JS errors)

  **Verification:**
  - Filter controls appear and correctly narrow search results
  - Search result items display severity and type metadata as badges
  - No JavaScript errors in the browser console
  - Graceful degradation when Pagefind assets are absent

- [ ] **Unit 3: Verify end-to-end with Microsoft testdata package**

  **Goal:** Validate that weight boosting, metadata, and filters work together on a representative rule set.

  **Requirements:** R1, R2, R3, R4

  **Dependencies:** Units 1 and 2

  **Files:**
  - No file changes -- manual verification against built site

  **Approach:**
  - Run `rulebound build` against the Microsoft testdata package
  - Open the generated site in a browser and exercise search scenarios
  - Verify ranking, metadata display, and filter behavior

  **Test scenarios:**
  - Search "Avoid" -- the Avoid rule page should be the first result, with "error" severity badge and "existence" type badge visible in the result
  - Search "use" -- substitution rules with "use" in their swap table should rank, but the message-boosted results should appear above body-text-only matches
  - Filter to "error" severity only -- only error-level rules appear
  - Filter to "substitution" type + "warning" severity -- correct intersection appears
  - Search with no query but filters active -- shows all matching rules

  **Verification:**
  - All five search scenarios produce expected results
  - No console errors, no broken layout, filter state persists across searches

## System-Wide Impact

- **Interaction graph:** Only the Hugo theme templates change. The Go build pipeline, parser, and generator are untouched.
- **Error propagation:** If Pagefind is not installed (already handled gracefully), the search UI simply won't initialize -- no change to current behavior.
- **API surface parity:** When the guideline content type ships, `guideline/single.html` should add the same `data-pagefind-weight`, `data-pagefind-meta`, and `data-pagefind-filter="content_type:guideline"` attributes. Note this in the guideline implementation plan.

## Risks & Dependencies

- **Pagefind Component UI availability**: The Component UI (`pagefind-component-ui.js` / `pagefind-component-ui.css`) must be present in the `/pagefind/` output directory. Pagefind v1.0+ ships both standard and component UI assets by default. If a user has an older Pagefind, the module import will fail silently (module scripts don't throw global errors) -- but the `typeof PagefindModularUI` guard inside `DOMContentLoaded` will skip initialization gracefully.
- **Pagefind auto-heading boost interaction**: Since Pagefind v1.0 already boosts headings, adding an explicit `data-pagefind-weight` to the `<h1>` could over-boost titles. The plan avoids this by not adding explicit weight to the `<h1>`.

## Sources & References

- **Origin document:** [docs/brainstorms/2026-03-26-pagefind-search-polish-requirements.md](docs/brainstorms/2026-03-26-pagefind-search-polish-requirements.md)
- **Ideation context:** [docs/ideation/2026-03-26-open-ideation.md](docs/ideation/2026-03-26-open-ideation.md) (idea #9)
- Related code: `internal/hugo/theme/layouts/_default/single.html`, `internal/hugo/theme/layouts/partials/search.html`
- Pagefind docs: weight API (quadratic scale 0.0-10.0), meta API (inline colon syntax), filter API (auto-shown in standard UI), Component UI (formerly Modular UI -- file: `pagefind-component-ui.js`, global: `PagefindModularUI`, custom result templates via `<script type="text/pagefind-template">`)
