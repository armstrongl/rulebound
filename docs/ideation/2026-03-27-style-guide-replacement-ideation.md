---
date: 2026-03-27
topic: style-guide-replacement
focus: Identify what rulebound needs to fully replace the style guide repo (Next.js/Nextra, ~166 MDX pages)
---

# Ideation: Style Guide Replacement

## Codebase Context

**Rulebound today**: Go CLI (Go 1.22+, Cobra, embedded Hugo theme, Pagefind search). Pipeline: parse Vale YAML rules -> generate Hugo content -> scaffold temp Hugo project -> run Hugo -> index with Pagefind. Supports rules (11 Vale types) + guidelines (designed, not yet implemented). Config via `rulebound.yml`.

**Style guide being replaced**: Next.js 14 / Nextra site with ~166 MDX pages across ~12 content categories: language & grammar (19), formatting & organization (15), punctuation (8), naming (3), linking (5), code & commands (4), key resources (4), documentation framework (complex subtree at 5-6 nesting levels), sections & scenarios (8), changelog (4), resources (3). Features: shadcn/ui components (Accordion, Tabs, Table, Alert), dark mode, collapsible sidebar nav, cross-linking, reusable snippets, GitHub edit links. Deployed on Fly.io with Docker multi-stage build + Go tsnet proxy.

**The gap**: Rulebound generates rule reference sites from Vale YAML. The style guide is a full documentation site with hand-authored editorial guidance. To replace it, rulebound must host arbitrary editorial content alongside auto-generated rule pages -- becoming a style guide platform, not just a rule reference generator.

**Key data points from style guide analysis**:
- 109/168 pages are pure Markdown (no JSX imports) -- directly portable
- 25 pages use JSX components from only 4 families: Alert (22), Table (9), Badge (9), Tabs (2)
- Nesting goes up to 5-6 levels deep (documentation-framework subtree)
- 20 `_meta.js` files control sidebar ordering and display names
- 2 reusable snippet files

**Past learnings**: Guidelines content type is designed/audited/approved but not yet implemented. Cross-linking explicitly deferred. Glossary content type rejected as premature. Hugo theme embedded via `go:embed`. Config system flexible.

## Ranked Ideas

### 1. Nested Content Sections (Generalize Beyond Flat Guidelines)

**Description:** Extend rulebound's content model to support arbitrary named sections with nested subdirectories. Instead of just `guidelines/` (flat, single section), support a `docs/` directory (or configurable name) with nested subdirectories, `_index.md` hub pages at each level, and section-level config. Each top-level subdirectory becomes a sidebar section. This subsumes guidelines as one instance of the general pattern.

**Rationale:** The style guide has ~12 content categories across 166 pages with nesting up to 5-6 levels deep. The current `guidelines/` model explicitly skips subdirectories (`if entry.IsDir() { continue }`) and produces a flat list. 109 of 168 style guide pages are pure Markdown -- they could be dropped into nested directories immediately. Without this, rulebound physically cannot hold the content.

**Downsides:** Largest single change -- touches parser, generator, sidebar template, config schema. Risk of over-engineering if not scoped tightly. Must coexist with auto-generated rule pages.

**Confidence:** 85%
**Complexity:** High
**Status:** Explored (brainstorm 2026-03-27)

### 2. Hierarchical Collapsible Sidebar with `_meta.yml`

**Description:** Replace the flat taxonomy-based sidebar with multi-level collapsible navigation. Introduce `_meta.yml` files at each directory level (like Nextra's `_meta.js`) controlling: ordering (list of filenames), display name overrides, and collapse state. Sidebar uses `<details>/<summary>` for expand/collapse (already styled in the theme CSS).

**Rationale:** 166+ pages in a flat sidebar is unusable. The style guide has 20 `_meta.js` files defining a curated hierarchy. Without equivalent ordering/nesting, migrated content loses its information architecture. v1 needs only three fields: `order`, `title`, `collapsed`.

**Downsides:** Requires parsing `_meta.yml` in Go and threading data into Hugo's data layer. Depends on #1 (nested content sections).

**Confidence:** 82%
**Complexity:** Medium-High
**Status:** Unexplored

### 3. Hugo Shortcodes for Component Parity

**Description:** Create shortcodes replicating the component families used in the style guide. Minimum viable: `{{< alert >}}` (callout/admonition boxes -- used on 22 pages) and `{{< table >}}` (structured data tables -- used on 9 pages). The theme already has CSS for these patterns.

**Rationale:** 25/168 pages use JSX components. Without shortcode equivalents, those pages can't be migrated. The component surface is small -- 2 families cover the most common cases.

**Downsides:** Low risk. Shortcodes are drop-in Hugo template files. Migration work (converting JSX to shortcode syntax) is a separate one-time effort.

**Confidence:** 90%
**Complexity:** Low
**Status:** Unexplored

### 4. `rulebound serve` (Watch Mode with Live Reload)

**Description:** New subcommand that scaffolds the Hugo project into a persistent directory, starts `hugo server` with live reload, and watches the source directory for changes. On file change, re-runs parse + generate for affected files.

**Rationale:** The style guide has `next dev` for instant preview. Without a dev mode, content authors must run `rulebound build` after every edit. For 166 pages of active content development, this friction is a migration deal-breaker.

**Downsides:** Requires changing scaffold lifecycle (temp dir must persist). `fsnotify` is a new dependency. Signal handling needed for clean shutdown. Should ship after #1 and #2.

**Confidence:** 80%
**Complexity:** Medium
**Status:** Unexplored

### 5. GitHub Edit Links

**Description:** Add `repository` config field to `rulebound.yml`. Render "Edit this page on GitHub" links on every page using existing `SourceFile` paths. One config field, one template line.

**Rationale:** The Nextra site has edit links as a core collaboration feature. Losing them in migration is a regression. Implementation is trivial (~30 minutes).

**Downsides:** None meaningful.

**Confidence:** 95%
**Complexity:** Low (trivial)
**Status:** Unexplored

### 6. Static Asset Pipeline

**Description:** If a `static/` directory exists next to the Vale package, copy it into the scaffolded Hugo project's `static/` directory. Images, PDFs, and other files become available at predictable URLs.

**Rationale:** Content pages cannot include images without this. Implementation is ~15 lines of Go.

**Downsides:** None meaningful.

**Confidence:** 95%
**Complexity:** Low (trivial)
**Status:** Unexplored

### 7. Dark Mode

**Description:** Add `@media (prefers-color-scheme: dark)` block with alternate CSS custom property values. The theme already uses CSS variables for all colors. Optional: manual toggle with localStorage persistence.

**Rationale:** The Nextra site has dark mode. Its absence is immediately visible. The architecture already supports it -- every color is a CSS variable.

**Downsides:** Choosing good dark mode colors requires design attention. Toggle needs a small JS snippet.

**Confidence:** 90%
**Complexity:** Low
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| 1 | Content Overlay (user `content/` dir) | Redundant with nested content sections -- same mechanism, different framing |
| 2 | Theme Partial Overrides | Architectural surgery to `ExtractTheme()` for a power-user feature no one has asked for |
| 3 | Content Migration CLI (`rulebound import`) | One-time operation -- belongs as a script in `scripts/`, not a compiled subcommand |
| 4 | Unified Sections Config | Premature abstraction -- only 2 content types, not enough to justify a generic system |
| 5 | Cross-Linking System | Markdown links already work; frontmatter relationships are over-engineering for v1 |
| 6 | Reusable Snippets | Hugo already has shortcodes; blocked by same scaffold architecture constraints |
| 7 | Taxonomy Unification | Contradicts existing design spec -- guidelines deliberately excluded from rule taxonomies |
| 8 | Multi-Package Merge | Not relevant to single-site replacement; significant complexity for nonexistent use case |
| 9 | Rulebound as Hugo Module | Breaks "single binary" value proposition; users would need Go + Hugo installed |
| 10 | Tabs Shortcode | Replaceable with `<details>` elements already styled in the theme |
| 11 | Badge Shortcode | Already exists as a partial; inline `<span>` with CSS class suffices |
| 12 | Companion Docs as Primary | Requires rethinking working code for aesthetic preference |
| 13 | Versioning/Changelog | Hand-authored content that would live in content sections (#1) |
| 14 | Dual-Source Architecture | A framing, not a feature -- achieved implicitly by content sections + auto-generated rules |

## Session Log

- 2026-03-27: Initial ideation -- 40 raw ideas generated (5 agents x ~8 each), deduped to 15 unique, 7 survived adversarial filtering (2 critic agents)
- 2026-03-27: Brainstorm initiated for #1 (Nested Content Sections)
