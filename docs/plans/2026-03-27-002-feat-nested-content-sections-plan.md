---
title: "feat: Add nested content sections with pages/ directory support"
type: feat
status: completed
date: 2026-03-27
origin: docs/brainstorms/2026-03-27-nested-content-sections-requirements.md
---

# feat: Add nested content sections with pages/ directory support

## Overview

Add a `pages/` content directory system to rulebound that supports arbitrary nested Markdown sections alongside auto-generated Vale rule pages. This supersedes the existing flat guidelines implementation with a general-purpose content sections model supporting nested subdirectories, `_meta.yml` navigation control, and a data-driven sidebar. The result: rulebound can host full style guide content (editorial guidance, formatting conventions, naming standards, resources) in a navigable, hierarchically structured site.

This plan adds the pages system alongside the existing guidelines code. Guidelines code is deprecated but not removed — removal is deferred to a follow-up plan after users have migrated. This avoids a broken intermediate state in the `ParseResult` contract and honors the R27 transition period.

## Problem Frame

Rulebound generates style guide websites from Vale rule packages, but the generated sites can only contain auto-generated rule pages. Real style guides include hand-authored editorial guidance alongside rules. To replace a full style guide (~166 pages across ~12 content categories with nesting up to 5-6 levels), rulebound needs the ability to host arbitrary hand-authored content in a navigable, nested structure.

The existing guidelines content type (`guidelines/` — flat, single section, no nesting) is too limited. This feature adds a general-purpose content sections model that subsumes guidelines. The existing guidelines implementation (~500 lines production code + ~300 lines tests) remains functional during a transition period but is deprecated in favor of `pages/`. (see origin: `docs/brainstorms/2026-03-27-nested-content-sections-requirements.md`)

## Requirements Trace

### Content Discovery
- R1. Auto-discover `pages/` directory in Vale package root; absent = rules-only mode
- R2. Every subdirectory under `pages/` becomes a top-level content section
- R3. Nesting up to 6 levels deep; deeper structures produce `ParseWarning` and flatten to level 6
- R4. Any `.md` file at any depth is a content page
- R5. Non-`.md` files ignored without warning

### Page Format
- R6. Markdown with YAML frontmatter; required: `title`, optional: `description`
- R7. Missing `title` falls back to filename-derived title (kebab-to-Title-Case) with `ParseWarning`; pages never silently skipped
- R8. Pages with only frontmatter and no body are valid
- R9. Pages use Hugo content type `page` (`type: page` injected during generation)

### Navigation Control
- R10. `_meta.yml` files completely optional at every level
- R11. Absent `_meta.yml`: directory name as title, alphabetical sort, expanded, nothing hidden
- R12. `_meta.yml` fields: `title`, `order`, `collapsed`, `hidden`, `rules_title` (top-level only)
- R13. `pages/rules/` directory takes precedence over `rules` keyword; `ParseWarning` emitted

### Hub Pages
- R14. `_index.md` files completely optional
- R15. Absent `_index.md`: auto-generated during content generation with minimal frontmatter (`title`, `type: page`); renders child listing via `page/list.html`
- R16. Present `_index.md`: section header links to hub page with author's content

### Rules Section
- R17. Rules appear as special top-level sidebar section (titled "Rules" by default, overridable via `rules_title`)
- R18. Within rules section, rules grouped by categories as collapsible subsections
- R19. Rules position controlled by `rules` keyword in top-level `_meta.yml` `order` list; default: bottom
- R20. No `pages/` directory = sidebar renders exactly as today (backward compatible)

### Sidebar
- R21. Pages sections and rules section render as visual peers
- R22. Collapsible sections use `<details>/<summary>` (existing CSS)
- R23. Sidebar supports nesting up to 6 levels with distinct indentation
- R24. Sidebar driven by `data/navigation.json`; replaces taxonomy-driven rendering when pages present

### Config
- R25. Existing `rulebound.yml` fields work unchanged
- R26. New `pages.enabled` field (`*bool`, default: true) disables pages discovery when `false`
- R27. `guidelines` config key deprecated; ignored when `pages/` exists; preserved when only `guidelines/` exists

### Guidelines Migration
- R28. Existing guidelines implementation superseded by pages system (guidelines code deprecated, removal deferred to follow-up plan)
- R29. Reusable patterns carried forward: YAML fence-splitting logic, Hugo layout routing via `type:` frontmatter, `*bool` config pattern
- R30. `ParseResult` adds `Pages *SectionTree` field alongside existing `Guidelines` field (guidelines field removed in follow-up plan)

### Static Assets
- R31. `static/` directory alongside Vale package copied into Hugo project's `static/`

### Search
- R32. Pages indexed by Pagefind via `data-pagefind-body`
- R33. Pages have `data-pagefind-meta` for title and description
- R34. Hidden pages excluded from Pagefind indexing (no `data-pagefind-body`)

## Scope Boundaries

- No shortcodes or rich components
- No dark mode or theme customization
- No `rulebound serve` / watch mode
- No content migration tooling
- No cross-linking system between pages and rules
- No GitHub edit links
- No multi-package merge builds
- No guidelines code removal (deferred to follow-up plan after transition period)

## Context & Research

### Relevant Code and Patterns

- **Pipeline contract**: `ParseResult` in `internal/parser/types.go` (lines 155-159) — central data structure flowing through `ParsePackage()` → `Scaffold()` → `GenerateSite()` → Hugo build → Pagefind
- **Guideline parser**: `internal/parser/guideline.go` — `parseGuidelines()` (136 lines), `parseFrontmatter()` returns `*guidelineFrontmatter` (tightly coupled to guideline type — not directly reusable for pages)
- **Guideline generator**: `internal/generator/guideline.go` — `GenerateGuideline()`, `generateGuidelinesIndex()`, `applyGuidelinesConfig()` (130 lines)
- **Config**: `internal/config/config.go` — `Config` struct with `GuidelinesConfig`, `*bool` pattern for `Enabled`
- **Scaffold**: `internal/hugo/scaffold.go` — `Scaffold()` signature: `(parseResult *parser.ParseResult, cfg *config.Config) (*ScaffoldResult, error)`. Does not receive the package directory path.
- **Sidebar template**: `internal/hugo/theme/layouts/partials/sidebar.html` — 3-mode rendering (categories, ruletypes, flat guidelines), taxonomy-driven via `site.Taxonomies.categories`
- **Hugo data access**: `index hugo.Data "site"` pattern (Hugo 0.120+), used in existing templates
- **Content type routing**: `type:` frontmatter routes to `layouts/<type>/single.html` (established pattern)
- **Generator pattern**: `GenerateRule()` in `internal/generator/generator.go` — writes frontmatter + body to Hugo content directory
- **CSS**: `internal/hugo/theme/static/css/style.css` — CSS custom properties, existing `<details>/<summary>` styling
- **Go version constraint**: `go.mod` specifies Go 1.22 — `os.CopyFS` is not available (Go 1.23+)

### Institutional Learnings

- `ParseResult` is the central pipeline contract; all new content types add fields to it
- Hugo type-based layout resolution is the established routing pattern for new content types
- Sidebar has 3-mode architecture; adding a 4th mode (data-driven) follows the existing conditional pattern
- `parseFrontmatter()` is tightly coupled to `guidelineFrontmatter` return type — the YAML fence-splitting logic is reusable but the function itself cannot be called directly for pages
- Flat-directory-only design (`if entry.IsDir() { continue }`) is being reversed by this feature
- Ordering uses synthetic weight offsets (-10000) for deterministic sort; navigation.json replaces this for pages
- `*bool` config pattern provides backward-compatible optional fields
- `data/site.json` pattern for Hugo data file generation is directly reusable for `navigation.json`
- Pagefind integration uses `data-pagefind-body` on content containers; new layouts follow the same pattern
- Test conventions: stdlib `testing`, `t.TempDir()`, file-based testdata in `internal/parser/testdata/`

## Key Technical Decisions

- **`SectionTree` in `internal/parser/types.go`**: Parser package already defines all types `ParseResult` references. A separate `internal/pages/` package would create unnecessary indirection. Parsing logic goes in `internal/parser/page.go` (parallel to `guideline.go`). (Resolves deferred question affecting R30)

- **Additive `ParseResult` change**: `Pages *SectionTree` is added alongside existing `Guidelines []*Guideline`. Both fields coexist during the transition period. Guidelines code continues to compile and function when no `pages/` directory exists. The `Guidelines` field is removed in a follow-up plan. This avoids breaking the `GenerateSite()` and test code that references `result.Guidelines`.

- **New `parsePageFrontmatter()` instead of reusing `parseFrontmatter()`**: The existing `parseFrontmatter()` returns `*guidelineFrontmatter` (a private type with `Weight` field that pages don't need). A new `parsePageFrontmatter()` function in `page.go` handles page-specific frontmatter (`title`, `description`). The YAML fence-splitting logic (detecting `---` delimiters) can be extracted into a shared `splitFrontmatter()` helper if the duplication is significant, but this is an implementation detail.

- **Auto-generated `_index.md` structure**: Minimal frontmatter — `title` (from `_meta.yml` or directory name), `type: page`. Hugo's default section behavior routes `_index.md` files with `type: page` to `page/list.html`. No explicit `layout` field needed. The root `pages/` directory also gets an auto-generated `_index.md` at `content/pages/_index.md`. (Resolves deferred question affecting R15)

- **`navigation.json` schema**: Tree structure with `sections` array (preserves order) containing recursive nodes, plus `rules_section` object with position index. Each section node has `name`, `title`, `path`, `collapsed`, `pages` array, and `children` array. Hidden pages are filtered out during JSON generation — they do not appear in `navigation.json` at all (no consumer needs them). Rules section has `title` and `position` semantics: 0 = before all sections, N = after Nth section, -1 = after all sections. Out-of-range positive values clamped to -1 (bottom). (Resolves deferred question affecting R24)

- **Page path computation**: Each `Page` has a `Path` field computed during parsing from its filesystem position relative to the `pages/` root. For a file at `pages/language/active-voice.md`, the path is `/pages/language/active-voice/`. This matches Hugo's default permalink generation for content in a `pages/` section. Paths use the filename slug directly (no additional slugification).

- **`Scaffold()` signature change for static assets**: `Scaffold()` needs the package directory path to locate `static/`. The simplest approach: add a `packageDir string` parameter to `Scaffold()`. This ripples to `cmd/build.go` (caller) and scaffold tests.

- **Static asset copy uses `filepath.WalkDir`**: Go 1.22 does not have `os.CopyFS` (Go 1.23+). Use `filepath.WalkDir` + `os.MkdirAll` + `os.ReadFile`/`os.WriteFile` for recursive copy. Package `static/` assets take precedence over theme `static/` assets in case of path collision.

- **Transition behavior**: When `pages/` exists, the pages system handles all content sections and `guidelines` config is ignored (deprecation warning emitted). When no `pages/` exists, guidelines code runs unchanged. Guidelines code is NOT removed in this plan — removal is a separate follow-up plan after the transition period. (Resolves deferred question affecting R27)

- **Dual-mode sidebar**: `sidebar.html` uses `{{ with (index hugo.Data "navigation") }}` for the data-driven path, with `{{ else }}` wrapping the entire existing taxonomy + guidelines rendering. This ensures only one mode executes.

- **Depth-aware CSS via Hugo template**: Sidebar nesting indentation generated by Hugo template using a depth variable threaded through recursive partial calls. CSS classes `.sidebar-depth-1` through `.sidebar-depth-6` with `padding-left` increments. These classes are defined in Unit 6 (sidebar) since they are sidebar concerns.

- **Empty `pages/` directory**: A `pages/` directory that exists but contains no `.md` files or subdirectories at any level is treated as nil for navigation.json generation purposes. The sidebar falls back to taxonomy mode.

## Open Questions

### Resolved During Planning

- **Auto-generated `_index.md` fields**: Minimal — `title` + `type: page`. Hugo section behavior handles routing to `page/list.html`. Root `pages/` also gets one.
- **`navigation.json` schema**: Tree with `sections` array + `rules_section` object. Hidden pages filtered during generation. Position semantics: 0=before all, N=after Nth, -1=bottom.
- **`SectionTree` location**: `internal/parser/types.go` alongside other `ParseResult` types. Parsing logic in `internal/parser/page.go`.
- **Transition period behavior**: `pages/` wins entirely. Guidelines code preserved but deprecated. Removal in follow-up plan.
- **`parseFrontmatter()` reuse**: Not directly reusable — returns guideline-specific type. New `parsePageFrontmatter()` in `page.go`. YAML fence-splitting logic may be extracted into shared helper.
- **`Scaffold()` package path**: Add `packageDir string` parameter. Simplest approach, matches existing input pattern.
- **Static asset copy method**: `filepath.WalkDir` + manual copy (Go 1.22 constraint, `os.CopyFS` unavailable).
- **ParseResult contract change**: Additive — `Pages` field added alongside `Guidelines`. No removal until follow-up plan.

### Deferred to Implementation

- **Hugo `_index.md` content rendering in list layout**: Whether the list layout should render both the hub page body and the child listing, or just the body when present. Determine during Unit 5 by testing with Hugo.
- **Sidebar active-page highlighting**: The exact mechanism for highlighting the current page in the data-driven sidebar. Hugo's `.IsAncestor` / `.IsDescendant` methods or URL comparison. Determine during Unit 5 implementation.
- **Shared `splitFrontmatter()` extraction**: Whether the YAML fence-splitting logic warrants extraction into a shared helper depends on how much duplication exists between `parsePageFrontmatter()` and the existing `parseFrontmatter()`. Assess during Unit 2.

## High-Level Technical Design

> *This illustrates the intended approach and is directional guidance for review, not implementation specification. The implementing agent should treat it as context, not code to reproduce.*

```
Pipeline Data Flow (pages/ integration):

                    Vale Package Root
                    ├── rules/          (existing)
                    ├── pages/          (NEW)
                    │   ├── _meta.yml
                    │   ├── language/
                    │   │   ├── _meta.yml
                    │   │   ├── _index.md
                    │   │   ├── active-voice.md
                    │   │   └── pronouns.md
                    │   └── formatting/
                    │       └── headings.md
                    └── static/         (NEW)
                        └── images/

    ┌─────────────────────────────────────────────────┐
    │  ParsePackage()                                  │
    │  ├── parseRules()      → []*ValeRule             │
    │  ├── parsePages()      → *SectionTree  (NEW)     │
    │  └── return ParseResult{Rules, Pages, Warnings}  │
    │      (Guidelines field preserved, not removed)    │
    └──────────────────┬──────────────────────────────┘
                       │
                       ▼
    ┌─────────────────────────────────────────────────┐
    │  Scaffold(parseResult, cfg, packageDir)           │
    │  ├── create temp Hugo project                    │
    │  ├── extract theme                               │
    │  ├── copy static/ assets    (NEW, uses pkgDir)   │
    │  └── return ScaffoldResult                       │
    └──────────────────┬──────────────────────────────┘
                       │
                       ▼
    ┌─────────────────────────────────────────────────┐
    │  GenerateSite()                                  │
    │  ├── GenerateRule() for each rule    (existing)  │
    │  ├── generatePageTree()              (NEW)       │
    │  ├── generateNavigationJSON()        (NEW)       │
    │  ├── generateSiteJSON()              (existing)  │
    │  └── generateHomepageIndex()         (existing)  │
    │  (guideline generation still runs if no pages)    │
    └──────────────────┬──────────────────────────────┘
                       │
                       ▼
    ┌─────────────────────────────────────────────────┐
    │  Hugo Build + Pagefind                           │
    │  ├── Hugo reads data/navigation.json             │
    │  ├── sidebar.html: {{ with nav }} data-driven    │
    │  │                 {{ else }} taxonomy-driven     │
    │  ├── page/single.html: renders content pages     │
    │  ├── page/list.html: renders section listings    │
    │  └── Pagefind indexes pages (excl. hidden)       │
    └─────────────────────────────────────────────────┘
```

```
navigation.json schema (directional):

{
  "sections": [
    {
      "name": "language",
      "title": "Language & Grammar",
      "path": "/pages/language/",
      "collapsed": false,
      "pages": [
        { "title": "Active Voice", "path": "/pages/language/active-voice/" },
        { "title": "Pronouns", "path": "/pages/language/pronouns/" }
      ],
      "children": [
        { "name": "advanced", "title": "Advanced", "path": "/pages/language/advanced/", ... }
      ]
    }
  ],
  "rules_section": {
    "title": "Rules",
    "position": -1,
    "categories": [
      { "name": "Avoid", "title": "Avoid", "rules": [
        { "title": "Avoid jargon", "path": "/rules/avoid-jargon/" }
      ]}
    ]
  }
}

Position semantics:
  0    = before all sections
  N    = after Nth section (1-indexed)
  -1   = after all sections (default)
  >len = clamped to -1

Hidden pages are NOT included — they are filtered during generation.
```

## Implementation Units

- [x] **Unit 1: Types, config, and data structures**

  **Goal:** Define foundational types for the pages system and update config schema. Add `Pages` field to `ParseResult` alongside existing `Guidelines` (additive, non-breaking change).

  **Requirements:** R25, R26, R27, R30

  **Dependencies:** None

  **Files:**
  - Modify: `internal/parser/types.go`
  - Modify: `internal/config/config.go`
  - Test: `internal/config/config_test.go`

  **Approach:**
  - Add `Page` struct: `Title`, `Description`, `Body`, `SourceFile`, `Path string`, `Hidden bool`
  - Add `SectionMeta` struct: `Title string`, `Order []string`, `Collapsed bool`, `Hidden []string`, `RulesTitle string` (with `yaml` struct tags)
  - Add `SectionTree` struct: `Name`, `Title`, `Path string`, `Pages []*Page`, `Children []*SectionTree`, `Meta *SectionMeta`, `IndexPage *Page` (hub page from `_index.md`)
  - Add `Pages *SectionTree` field to `ParseResult` — alongside existing `Guidelines`, NOT replacing it
  - Add `PagesConfig` struct to config: `Enabled *bool` (following existing `*bool` pattern from `GuidelinesConfig`)
  - Add `Pages PagesConfig` field to `Config`

  **Execution note:** Start with config parsing tests (pages.enabled states, deprecation warning). Types are verified at compile time and do not need separate unit tests.

  **Patterns to follow:**
  - `Guideline` struct in `types.go` for struct layout conventions
  - `GuidelinesConfig` in `config.go` for `*bool` pattern
  - Existing `yaml` struct tags throughout config types

  **Test scenarios:**
  - `PagesConfig` with `Enabled: true`, `false`, `nil` (default true behavior)
  - Config YAML with `pages.enabled: false` parses correctly
  - Config YAML with `guidelines` key + `pages/` present — deprecation warning emitted
  - Existing config fields (`title`, `description`, `baseURL`, `categories`) still parse correctly (R25)

  **Verification:**
  - All new types compile and are importable
  - `ParseResult` compiles with both `Guidelines` and `Pages` fields — no existing code breaks
  - Config parsing handles all `*bool` states correctly

- [x] **Unit 2: Page parser**

  **Goal:** Implement `pages/` directory discovery and recursive parsing into `SectionTree`.

  **Requirements:** R1-R8, R10-R13

  **Dependencies:** Unit 1 (types)

  **Files:**
  - Create: `internal/parser/page.go`
  - Modify: `internal/parser/parser.go` (wire `parsePages` into `ParsePackage`)
  - Test: `internal/parser/page_test.go`
  - Create: testdata fixtures under `internal/parser/testdata/` for pages scenarios

  **Approach:**
  - `parsePages(dir string) (*SectionTree, []ParseWarning, error)` — entry point, checks for `pages/` existence
  - `parsePagesDir(dir string, name string, depth int) (*SectionTree, []ParseWarning, error)` — recursive walker
  - Write a new `parsePageFrontmatter()` function in `page.go` for page-specific frontmatter parsing (Title, Description). Do NOT reuse `parseFrontmatter()` from `guideline.go` — it returns `*guidelineFrontmatter` which is tightly coupled. Extract shared YAML fence-splitting logic into a helper if duplication is significant.
  - Parse `_meta.yml` at each directory level using `go.yaml.in/yaml/v3` into `SectionMeta`. Malformed YAML produces a `ParseWarning` and falls back to defaults (does not fail the entire parse).
  - Build tree structure matching filesystem hierarchy
  - Compute `Page.Path` from filesystem position relative to `pages/` root (e.g., `pages/language/active-voice.md` → `/pages/language/active-voice/`)
  - Apply 6-level depth cap with `ParseWarning` for deeper structures (R3)
  - Generate filename-derived titles for missing `title` frontmatter (R7) — kebab-case to Title Case
  - Detect `_index.md` hub pages and attach to `SectionTree.IndexPage` (R14, R16)
  - After parsing all pages in a directory, cross-reference `SectionMeta.Hidden` entries against page filenames and set `Page.Hidden = true` for matches
  - Handle `pages/rules/` directory collision with `rules` keyword (R13) — directory takes precedence, emit warning
  - Skip non-`.md` files silently (R5)
  - Wire `parsePages()` call into `ParsePackage()` — called after `parseRules()`, gated by `pages.enabled` config

  **Execution note:** Test-first. Build testdata fixtures first, then write failing tests, then implement parser functions.

  **Patterns to follow:**
  - `parseGuidelines()` in `guideline.go` for directory walking pattern
  - `ParsePackage()` for how parsing functions are called and results aggregated
  - Testdata fixture pattern from `internal/parser/testdata/Microsoft/`

  **Test scenarios:**
  - No `pages/` directory — returns nil `SectionTree`, no warnings
  - Empty `pages/` directory — returns nil (treated as absent for downstream consumers)
  - Single level with 3 `.md` files — correct `Pages` slice, correct `Path` fields
  - Nested 3 levels deep — correct `Children` tree structure
  - 7+ levels deep — depth cap warning, flattened to level 6
  - `_meta.yml` with all 5 fields — correctly parsed into `SectionMeta`
  - `_meta.yml` absent — defaults applied (alphabetical, expanded, nothing hidden)
  - `_meta.yml` with `order` list — pages ordered accordingly, unlisted files alphabetical after
  - `_meta.yml` with malformed YAML — `ParseWarning` emitted, defaults applied
  - `_meta.yml` with wrong types (e.g., `collapsed: "yes"`) — `ParseWarning`, defaults
  - Missing `title` in page frontmatter — filename-derived title, `ParseWarning` emitted
  - Page with only frontmatter, no body — valid parse (R8)
  - `_index.md` present — attached to `SectionTree.IndexPage`
  - `_index.md` absent — `IndexPage` is nil (auto-generation deferred to generator)
  - `pages/rules/` directory exists + `rules` keyword in top-level `_meta.yml` — directory wins, warning
  - Non-`.md` files in directory — silently ignored
  - `pages.enabled: false` config — `parsePages()` not called
  - Hidden pages: file listed in `_meta.yml` `hidden` array has `Page.Hidden = true`
  - Mixed: pages/ with some dirs having `_meta.yml` and some without

  **Verification:**
  - `parsePages()` returns correct `SectionTree` for all fixture scenarios
  - All warnings have descriptive messages identifying the file/directory
  - `ParsePackage()` integrates pages parsing without breaking rule parsing
  - Page paths match expected Hugo permalink format

- [x] **Unit 3: Page content and navigation generation**

  **Goal:** Generate Hugo content files from parsed pages (including auto-generated `_index.md`) and generate `data/navigation.json` for sidebar consumption.

  **Requirements:** R9, R15-R19, R24, R34

  **Dependencies:** Unit 1 (types), Unit 2 (parser)

  **Files:**
  - Create: `internal/generator/page.go`
  - Create: `internal/generator/navigation.go`
  - Modify: `internal/generator/generator.go` (wire page + navigation generation into `GenerateSite`)
  - Test: `internal/generator/page_test.go`
  - Test: `internal/generator/navigation_test.go`

  **Approach:**

  *Page content generation:*
  - `GeneratePage(page *Page, contentDir string) error` — writes single page `.md` with `type: page` frontmatter injected (R9), `pagefind: false` param when `page.Hidden` is true (R34)
  - `generatePageTree(tree *SectionTree, contentDir string, basePath string) error` — recursive walker that generates all pages and `_index.md` files
  - Root `pages/` container: generate `content/pages/_index.md` with title from top-level `_meta.yml` or default
  - For sections without `IndexPage`: auto-generate `_index.md` with `title` (from `_meta.yml` or directory name) and `type: page` (R15)
  - For sections with `IndexPage`: write author's content with `type: page` injected (R16)
  - When `ParseResult.Pages` is non-nil, run page generation; otherwise skip (guideline generation continues to run as before)

  *Navigation data generation:*
  - `generateNavigationJSON(pages *SectionTree, rules []*ValeRule, dataDir string) error` — main entry point
  - Build navigation structure: `sections` array from `SectionTree` (hidden pages filtered out — not emitted in JSON), `rules_section` from rules grouped by category
  - Position rules section based on `rules` keyword in top-level `_meta.yml` `order` list (R19). Position semantics: 0 = before all, N = after Nth, -1 = bottom (default). Out-of-range clamped to -1.
  - Apply `rules_title` override from top-level `_meta.yml` (R17)
  - Only generate `navigation.json` when `pages` is non-nil and non-empty (R20)

  **Execution note:** Test-first. Page generation and navigation generation have distinct test files but share the same SectionTree input, so test fixtures can be reused.

  **Patterns to follow:**
  - `GenerateRule()` in `generator.go` for frontmatter injection and file writing pattern
  - `generateSiteJSON()` in `internal/generator/index.go` for data file generation pattern
  - `aggregateCounts()` for category grouping logic

  **Test scenarios:**

  *Page generation:*
  - Single page generates `.md` with correct `type: page` frontmatter
  - Page with `title` and `description` — both in frontmatter
  - Page with body content — body preserved after frontmatter
  - Section without `IndexPage` — auto-generated `_index.md` with title from directory name
  - Section with `IndexPage` — author content preserved, `type: page` injected
  - Root `pages/` container — `content/pages/_index.md` generated
  - Nested tree — correct directory structure created
  - Hidden page — `pagefind: false` param added to frontmatter
  - Empty section (directory with no pages, only children) — `_index.md` still generated

  *Navigation generation:*
  - Simple tree with 2 sections — correct JSON structure
  - `rules` keyword in `order` at position 1 — rules section at position 1
  - `rules` keyword at position 0 — rules before all sections
  - No `rules` keyword — rules section position is -1 (bottom)
  - Position value exceeds section count — clamped to -1
  - `rules_title: "Style Rules"` — overrides default "Rules" title
  - Nested sections — recursive children in JSON
  - Hidden pages — not present in JSON output at all
  - Section with `collapsed: true` — reflected in JSON
  - Nil pages (no pages/ directory) — no `navigation.json` generated
  - Empty SectionTree (pages/ exists but empty) — no `navigation.json` generated
  - Rules with multiple categories — categories array populated correctly

  **Verification:**
  - Generated Hugo content directory matches `SectionTree` structure
  - All pages have `type: page` in frontmatter
  - Auto-generated `_index.md` files have correct titles, including root container
  - Hidden pages have Pagefind exclusion param but do not appear in navigation JSON
  - Navigation JSON position semantics work correctly for all edge cases
  - No `navigation.json` when pages is nil or empty

- [x] **Unit 4: Hugo page layouts**

  **Goal:** Create Hugo layouts for the `page` content type — single page rendering and section listing.

  **Requirements:** R9, R15, R32-R34

  **Dependencies:** None (template files, can be developed in parallel with Units 2-3)

  **Files:**
  - Create: `internal/hugo/theme/layouts/page/single.html`
  - Create: `internal/hugo/theme/layouts/page/list.html`
  - Modify: `internal/hugo/theme/static/css/style.css` (`.page-content` container styles)

  **Approach:**
  - `page/single.html`: Render page content with `data-pagefind-body` (R32), `data-pagefind-meta` for title and description (R33). Conditionally omit `data-pagefind-body` when `.Params.pagefind` is `false` (R34 — hidden pages). Follow existing `single.html` patterns from rule layout.
  - `page/list.html`: Render section hub page content (if present) followed by child page listing. Use Hugo's `.Pages` to iterate children. Provide clean default listing when `_index.md` has no body.
  - CSS: Add `.page-content` container styles only. Sidebar depth classes belong in Unit 5.

  **Patterns to follow:**
  - Existing `layouts/rule/single.html` for Pagefind attributes and content structure (note: rules use `_default/single.html`, not `layouts/rules/single.html`)
  - Existing `layouts/guideline/single.html` for content type layout pattern
  - Existing `<details>/<summary>` CSS in `style.css`

  **Test scenarios:**
  - `single.html` renders page title and body content
  - `single.html` includes `data-pagefind-body` for normal pages
  - `single.html` excludes `data-pagefind-body` when `pagefind: false` param
  - `single.html` includes `data-pagefind-meta` attributes
  - `list.html` renders child page listing
  - `list.html` renders hub page body when present
  - Verify `type: page` frontmatter actually routes to `page/single.html` (not `_default/single.html`) — test with a page in `content/pages/subsection/file.md`
  - Verify `_index.md` with `type: page` routes to `page/list.html`

  **Verification:**
  - Hugo builds successfully with new layouts
  - Pagefind indexes normal pages but not hidden ones
  - Page content renders correctly in generated site
  - `type: page` override correctly selects page layouts over Hugo's section-inferred type

- [x] **Unit 5: Data-driven sidebar**

  **Goal:** Rewrite sidebar partial to render from `navigation.json` when present, preserving taxonomy-driven rendering as fallback. Includes sidebar-specific CSS.

  **Requirements:** R20-R24

  **Dependencies:** Unit 3 (navigation.json schema), Unit 4 (page layout existence for link targets)

  **Files:**
  - Modify: `internal/hugo/theme/layouts/partials/sidebar.html`
  - Create: `internal/hugo/theme/layouts/partials/sidebar-section.html` (recursive partial for nested sections)
  - Modify: `internal/hugo/theme/static/css/style.css` (`.sidebar-depth-1` through `.sidebar-depth-6` classes)

  **Approach:**
  - Structure `sidebar.html` as: `{{ with (index hugo.Data "navigation") }}` [data-driven rendering] `{{ else }}` [entire existing taxonomy + guidelines rendering] `{{ end }}`. The `{{ else }}` branch wraps ALL existing sidebar content to ensure only one mode executes.
  - Data-driven path: iterate `sections` array, interleave `rules_section` at its `position` index
  - Recursive `sidebar-section.html` partial: renders one section node with `<details>/<summary>` for collapsibility (R22), threads `depth` variable via `dict` context passing for CSS class assignment, recurses into `children`
  - Active page highlighting via URL comparison with `.RelPermalink`
  - Rules section renders category groups as collapsible subsections within the rules `<details>` block (R18)
  - Sections and rules render as visual peers — same `<details>/<summary>` pattern, same styling (R21)
  - CSS: Add `.sidebar-depth-1` through `.sidebar-depth-6` classes with incremental `padding-left` for nested sidebar indentation (R23)

  **Patterns to follow:**
  - Existing sidebar taxonomy iteration for rules category rendering
  - Hugo `{{ partial }}` recursion pattern with `dict` context passing
  - Existing `<details>/<summary>` usage in sidebar

  **Test scenarios:**
  - No `navigation.json` — sidebar renders identically to current behavior (taxonomy + guidelines)
  - Simple navigation with 2 sections + rules at bottom — correct order
  - Rules positioned at index 1 — renders between sections
  - Rules at position 0 — renders before all sections
  - Collapsed section — `<details>` without `open` attribute
  - Expanded section — `<details open>`
  - Nested 3-level section — correct depth classes applied (`.sidebar-depth-1`, `-2`, `-3`)
  - Active page — highlighted with active class
  - Rules section with multiple categories — each category collapsible
  - Depth CSS classes produce visually distinct indentation at each level

  **Verification:**
  - Generated site with pages has data-driven sidebar
  - Generated site without pages has identical sidebar to current builds (backward compat)
  - All nesting levels render with correct indentation
  - Collapse/expand works correctly
  - Only one sidebar mode executes (not both)

- [x] **Unit 6: Static assets pipeline**

  **Goal:** Copy `static/` directory from Vale package root into scaffolded Hugo project.

  **Requirements:** R31

  **Dependencies:** None (can be developed in parallel)

  **Files:**
  - Modify: `internal/hugo/scaffold.go` (add `packageDir` parameter, implement copy)
  - Modify: `cmd/build.go` (pass package path to `Scaffold()`)
  - Test: `internal/hugo/scaffold_test.go`

  **Approach:**
  - Add `packageDir string` parameter to `Scaffold()` signature
  - Update `cmd/build.go` to pass the package path when calling `Scaffold()`
  - After theme extraction, check for `static/` directory at `filepath.Join(packageDir, "static")`
  - If present, recursively copy using `filepath.WalkDir` + `os.MkdirAll` + `os.ReadFile`/`os.WriteFile` (Go 1.22 — `os.CopyFS` unavailable)
  - Package `static/` assets take precedence over theme `static/` assets in case of path collision
  - ~15-20 lines of Go implementation

  **Patterns to follow:**
  - Existing file copy patterns in `scaffold.go` (theme extraction)

  **Test scenarios:**
  - No `static/` directory — no error, no copy
  - `static/` with images — files appear in Hugo project `static/`
  - `static/` with nested subdirectories — structure preserved
  - `static/` with various file types (images, PDFs) — all copied
  - `static/` file collides with theme file — package version appears (precedence)
  - `Scaffold()` with new signature — existing callers updated, tests pass

  **Verification:**
  - Files from source `static/` are accessible at expected Hugo URLs in built site
  - Scaffold still works correctly when no `static/` exists
  - No compilation errors from signature change

- [x] **Unit 7: Pipeline integration, deprecation, and build output**

  **Goal:** Wire pages system into the full build pipeline. Add deprecation warnings for guidelines config. Update build summary output. Ensure guidelines code continues to function when no `pages/` exists.

  **Requirements:** R25, R27-R29

  **Dependencies:** Units 1-6 (all pages functionality must be in place)

  **Files:**
  - Modify: `internal/parser/parser.go` (ensure `parsePages` is wired alongside existing `parseGuidelines`)
  - Modify: `internal/generator/generator.go` (add page generation path alongside existing guideline generation)
  - Modify: `internal/hugo/theme/layouts/partials/sidebar.html` (guidelines sidebar block moves inside `{{ else }}` branch — already part of Unit 5, verify here)
  - Modify: `cmd/build.go` (update build summary to report pages count when pages present, deprecation warnings)
  - Modify: `internal/config/config.go` (deprecation warning logic for `guidelines` key when `pages/` exists)

  **Approach:**
  - In `GenerateSite()`: when `ParseResult.Pages` is non-nil, run page generation + navigation.json generation. When nil, existing guideline generation runs as before. Both code paths coexist.
  - In `cmd/build.go`: update build summary to show pages count when pages present. Show guidelines count when only guidelines present (backward compat).
  - Deprecation warning: if `guidelines` config key present AND `pages/` directory exists, emit warning suggesting migration to `pages/guidelines/`. Guidelines config ignored in this case.
  - If no `pages/` directory exists, guidelines code runs unchanged — zero behavior change for existing users.
  - Guidelines code is NOT removed. It continues to function. Removal is a separate follow-up plan.

  **Patterns to follow:**
  - Existing deprecation/warning patterns in codebase
  - `cmd/build.go` summary output format

  **Test scenarios:**
  - Build with `pages/` directory — page generation runs, guidelines generation skipped
  - Build without `pages/` — guidelines generation runs as before, no pages code triggered
  - Build without `pages/` AND without `guidelines/` — rules-only, identical to current behavior
  - Config with `guidelines` key + `pages/` exists — deprecation warning emitted, guidelines config ignored
  - Config with `guidelines` key + no `pages/` — existing guidelines behavior preserved
  - Build summary shows pages count when pages present
  - Build summary shows guidelines count when only guidelines present
  - All existing tests continue to pass (no guidelines code removed)

  **Verification:**
  - `go build` succeeds with all code intact
  - Existing rule-only and guideline test cases still pass
  - Build output is correct for pages mode, guidelines mode, and rules-only mode
  - Deprecation warning appears when expected
  - Zero behavior change for users without `pages/` directory

- [x] **Unit 8: End-to-end integration tests**

  **Goal:** Verify the complete pipeline works with pages content, and backward compatibility is preserved for rules-only builds.

  **Requirements:** All (integration verification)

  **Dependencies:** Units 1-7 (all previous units must be complete)

  **Files:**
  - Create: `cmd/build_pages_test.go` (integration tests for pages pipeline, alongside existing `cmd/build.go`)
  - Create: test fixtures with a realistic pages directory structure

  **Approach:**
  - Full pipeline test: create a test Vale package with `pages/` directory (nested 3+ levels, `_meta.yml` files, `_index.md` files, hidden pages), run the full build pipeline, verify output
  - Backward compatibility test: run full pipeline with rules-only package (no `pages/`), verify output is identical to current behavior
  - Guidelines backward compat test: run with `guidelines/` but no `pages/`, verify guidelines still work
  - Verify `navigation.json` is generated correctly in data directory
  - Verify page content files exist with correct frontmatter
  - Verify `static/` assets are copied
  - Verify Hugo build succeeds and produces valid site

  **Patterns to follow:**
  - Existing test patterns in `cmd/` and `internal/` packages

  **Test scenarios:**
  - Full build with realistic pages structure (3 top-level sections, 2-3 levels deep, mixed `_meta.yml` presence)
  - Rules-only build (backward compat) — identical output to current builds
  - Guidelines-only build (backward compat) — identical output to current builds
  - Build with `pages.enabled: false` — pages ignored, existing behavior preserved
  - Build with `static/` directory — assets copied to Hugo project
  - Build with hidden pages — not in Pagefind index, not in navigation.json
  - Build with `rules` keyword in top-level `_meta.yml` — rules positioned correctly
  - Build with `_index.md` hub pages — hub content rendered

  **Verification:**
  - Complete pipeline produces valid Hugo site with pages and rules
  - Hugo build succeeds without errors
  - Generated site structure matches expected output
  - Backward compatibility confirmed for both rules-only and guidelines-only builds

## System-Wide Impact

- **Interaction graph:** `ParsePackage()` gains `parsePages()` call. `Scaffold()` gains `packageDir` parameter and static asset copy. `GenerateSite()` gains page generation + navigation.json. `sidebar.html` gains data-driven mode (with `{{ else }}` preserving existing mode). `cmd/build.go` summary output changes. All callers of `Scaffold()` must pass `packageDir`.
- **Error propagation:** Parse warnings from page parsing aggregate into `ParseResult.Warnings` (same pattern as rule warnings). Generation errors propagate up through `GenerateSite()` return value. Malformed `_meta.yml` produces warnings (not errors).
- **State lifecycle risks:** Navigation.json must be written before Hugo build runs. Auto-generated `_index.md` must not overwrite author-provided `_index.md` (checked via `SectionTree.IndexPage != nil`). `Scaffold()` creates directories before `GenerateSite()` writes content — ordering preserved. `Scaffold()` signature change affects `cmd/build.go` and all scaffold tests.
- **API surface parity:** `ParseResult` struct gains a field (additive, non-breaking). `Scaffold()` gains a parameter (breaking — all callers updated in Unit 6). No other public API changes.
- **Integration coverage:** Unit tests cover parser and generator independently. Integration tests (Unit 8) cover the full pipeline for pages, guidelines, and rules-only modes.

## Risks & Dependencies

- **Hugo section routing assumption:** The plan assumes `_index.md` with `type: page` routes to `page/list.html` via Hugo's section behavior. If Hugo's type-based routing doesn't work this way for branch bundles, the list layout may need a different routing mechanism. Mitigation: explicit verification test in Unit 4.
- **`Scaffold()` signature change:** Adding `packageDir` parameter is a breaking change to `Scaffold()`. All callers and tests must be updated. Mitigation: small blast radius — only `cmd/build.go` calls `Scaffold()`, plus scaffold tests. Updated atomically in Unit 6.
- **Recursive template performance:** Deep nesting with many pages could make sidebar rendering slow. Mitigation: 6-level cap limits recursion depth; real-world style guide has ~166 pages across ~12 sections which is well within Hugo's capability.
- **`_meta.yml` parsing edge cases:** YAML parsing of `order` lists, boolean `collapsed`, and string arrays for `hidden` needs careful handling of type mismatches and missing fields. Mitigation: malformed YAML produces `ParseWarning` and falls back to defaults. Comprehensive test fixtures in Unit 2.
- **Coexisting guidelines code:** Guidelines code remains in the codebase during the transition period, adding maintenance surface. Mitigation: guidelines code is stable and tested; it does not interact with pages code paths. Follow-up plan handles removal.

## Follow-Up Plan: Guidelines Code Removal

After the transition period (users have migrated from `guidelines/` to `pages/guidelines/`):
- Remove `Guideline` struct, `parseGuidelines()`, `GuidelinesConfig`, `applyGuidelinesConfig()`, `GenerateGuideline()`, `generateGuidelinesIndex()` from parser and generator
- Remove `internal/parser/guideline.go` and `internal/generator/guideline.go`
- Remove `internal/hugo/theme/layouts/guideline/single.html` and `list.html`
- Remove `.guideline-*` CSS classes
- Remove `Guidelines` field from `ParseResult`
- Remove `guidelines` config key support (no more deprecation warning — hard error or silent ignore)
- Remove guidelines sidebar block from `{{ else }}` branch
- Update/remove guidelines test files and testdata

This is a straightforward removal plan once the pages system is stable and users have migrated.

## Sources & References

- **Origin document:** [docs/brainstorms/2026-03-27-nested-content-sections-requirements.md](docs/brainstorms/2026-03-27-nested-content-sections-requirements.md)
- **Ideation:** [docs/ideation/2026-03-27-style-guide-replacement-ideation.md](docs/ideation/2026-03-27-style-guide-replacement-ideation.md) (Idea #1)
- **Existing guidelines spec:** [.claude/specs/2026-03-25-docs-and-guidelines-design.md](.claude/specs/2026-03-25-docs-and-guidelines-design.md) (superseded)
- Related code: `internal/parser/types.go`, `internal/parser/guideline.go`, `internal/generator/guideline.go`, `internal/hugo/scaffold.go`, `internal/hugo/theme/layouts/partials/sidebar.html`, `cmd/build.go`
