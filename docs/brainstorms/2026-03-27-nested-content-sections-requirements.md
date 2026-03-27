---
date: 2026-03-27
topic: nested-content-sections
---

# Nested Content Sections

## Problem Frame

Rulebound generates style guide websites from Vale rule packages, but the generated sites can only contain auto-generated rule pages. Real style guides include hand-authored editorial guidance (language and grammar, formatting conventions, naming standards, resources) alongside rules. To replace a full style guide (e.g., a ~166-page Nextra site), rulebound needs the ability to host arbitrary hand-authored content in a navigable, nested structure.

The existing guidelines content type (`guidelines/` -- flat, single section, no nesting) is already implemented in the codebase (parser, generator, config, Hugo layouts, sidebar rendering, tests) but is too limited for this purpose. This feature replaces the guidelines system with a general-purpose content sections model that supports arbitrary named sections with nested subdirectories and fine-grained navigation control.

**This is a refactor/replacement of working code, not a clean-sheet design.** The existing guidelines implementation (~500 lines of production code across parser/guideline.go, generator/guideline.go, config.go, two Hugo layouts, sidebar partial, plus ~300 lines of tests) must be migrated or removed. Several patterns from the guidelines code (frontmatter parsing via `parseFrontmatter()`, Hugo layout routing via `type:` frontmatter, sidebar section rendering) are directly reusable.

## Requirements

### Content Discovery

- R1. Rulebound auto-discovers a `pages/` directory in the Vale package root. If absent, rulebound operates as today (rules only, with existing sidebar rendering unchanged).
- R2. Every subdirectory under `pages/` becomes a top-level content section in the generated site.
- R3. Subdirectories can nest up to 6 levels deep, mirroring the filesystem structure in the generated site. Deeper structures produce a `ParseWarning` and are flattened to level 6.
- R4. Any `.md` file inside `pages/` (at any depth) is treated as a content page.
- R5. Non-`.md` files inside `pages/` are ignored without warning.

### Page Format

- R6. Content pages are Markdown files with YAML frontmatter. Required frontmatter: `title`. Optional: `description`.
- R7. Missing `title` falls back to a filename-derived title (kebab-case to Title Case, same conversion as R10 for directory names). A `ParseWarning` is emitted noting the fallback. Pages are never silently skipped for missing title.
- R8. Pages with only frontmatter and no body are valid (though unusual).
- R9. Pages use Hugo content type `page` with a dedicated layout (`layouts/page/single.html`). The generator injects `type: page` into frontmatter during content generation.

### Navigation Control (`_meta.yml`)

- R10. `_meta.yml` files are completely optional at every directory level.
- R11. When `_meta.yml` is absent: the directory name is used as the display title (kebab-case converted to Title Case via Hugo's `humanize | title` pipeline), pages sort alphabetically by filename, the section starts expanded, and nothing is hidden.
- R12. `_meta.yml` supports five fields:
  - `title` (string) -- override display name for this directory in the sidebar
  - `order` (list of strings) -- filenames (without extension) defining sidebar sequence; unlisted files sort alphabetically after ordered ones. Supports a reserved `rules` keyword at the top level (see R19).
  - `collapsed` (bool, default: false) -- whether the section starts collapsed
  - `hidden` (list of strings) -- filenames to exclude from sidebar navigation (pages remain accessible via direct URL but are excluded from Pagefind indexing)
  - `rules_title` (string, top-level only) -- override display name for the auto-generated rules section when positioned via the `rules` keyword
- R13. A directory named `rules` under `pages/` is a valid content section. If a top-level `_meta.yml` `order` list contains `rules` and a `pages/rules/` directory also exists, the directory takes precedence and the auto-generated rules section uses its default position (bottom). A `ParseWarning` is emitted.

### Hub Pages (`_index.md`)

- R14. `_index.md` files are completely optional.
- R15. When `_index.md` is absent: rulebound generates an internal `_index.md` during content generation (in `GenerateSite()`) with minimal frontmatter (`title`, `type: page`) and no body. The auto-generated page renders a listing of child pages using the `page/list.html` layout. The sidebar section header links to this listing page.
- R16. When `_index.md` is present: the section header links to the hub page. The hub page renders the author's Markdown content.

### Rules Section

- R17. Auto-generated rules appear in the sidebar as a special top-level section (titled "Rules" by default, overridable via `rules_title` in top-level `_meta.yml`).
- R18. Within the rules section, rules are grouped by their categories (as today), with category groups rendered as collapsible subsections.
- R19. The rules section's position among pages sections is controlled by the reserved `rules` keyword in the top-level `pages/_meta.yml` `order` list. If `rules` is not listed or no `_meta.yml` exists, rules appear at the bottom of the sidebar.
- R20. When no `pages/` directory exists, the sidebar renders rules exactly as it does today (flat category groups, no wrapping collapsible section). The new sidebar structure only activates when pages content is present.

### Sidebar

- R21. The sidebar renders pages sections and the rules section as visual peers -- consistent styling, uniform expand/collapse behavior.
- R22. Collapsible sections use `<details>/<summary>` elements (CSS for these already exists in the theme).
- R23. Sidebar supports nesting up to 6 levels with distinct visual indentation per level. Beyond 6 levels, content renders at level 6 indentation.
- R24. The sidebar is driven by a generated `data/navigation.json` file that encodes the full section tree (pages sections + rules section with their ordering, titles, collapse state, and nesting). Hugo templates read this data file to render the sidebar. This replaces the current taxonomy-driven sidebar rendering when pages content is present.

### Config Overrides (`rulebound.yml`)

- R25. Existing `rulebound.yml` fields (`title`, `description`, `baseURL`, `categories`) continue to work unchanged.
- R26. A new optional `pages.enabled` field (`*bool`, default: true) disables pages discovery when set to `false`.
- R27. The existing `guidelines` config key is deprecated. If present, rulebound emits a deprecation warning suggesting migration to `pages/guidelines/`. The `guidelines` config is ignored when `pages/` exists. If `pages/` does not exist and `guidelines/` does exist with a `guidelines` config, the existing guidelines behavior is preserved during a transition period.

### Guidelines Migration

- R28. The existing guidelines implementation (parser, generator, config, layouts, sidebar, tests) is replaced by the pages system. The `Guideline` struct, `parseGuidelines()`, `GuidelinesConfig`, `applyGuidelinesConfig()`, `GenerateGuideline()`, `generateGuidelinesIndex()`, guideline layouts, and sidebar guidelines block are removed.
- R29. Reusable patterns from the guidelines code are carried forward: `parseFrontmatter()` (used for page frontmatter parsing), Hugo layout routing via `type:` frontmatter, `*bool` config pattern for `pages.enabled`.
- R30. `ParseResult` replaces `Guidelines []*Guideline` with `Pages *SectionTree`. The `SectionTree` is a tree structure where each node has `Name`, `Title`, `Pages []*Page`, `Children []*SectionTree`, and `Meta *SectionMeta` (parsed from `_meta.yml`).

### Static Assets

- R31. If a `static/` directory exists alongside the Vale package, rulebound copies it into the scaffolded Hugo project's `static/` directory. Images, PDFs, and other files become available at predictable URLs from any content page.

### Search Integration

- R32. Pages content is indexed by Pagefind via `data-pagefind-body` on the page layout, consistent with rule pages.
- R33. Pages have `data-pagefind-meta` attributes for title and description.
- R34. Hidden pages (per `_meta.yml` `hidden` list) are excluded from Pagefind indexing (no `data-pagefind-body` attribute on their layout).

## Success Criteria

- A `pages/` directory with nested Markdown files produces a navigable, correctly structured style guide site alongside auto-generated rule pages.
- The sidebar renders pages sections and rules as visual peers with consistent expand/collapse behavior.
- Zero config produces a working site (alphabetical ordering, directory names as titles, rules at bottom). `_meta.yml` and `_index.md` enhance but are never required.
- Existing rulebound builds without a `pages/` directory work identically (backward compatible) -- same sidebar rendering, same output.
- A real-world test: the style guide repo's ~109 pure-Markdown pages can be dropped into `pages/` (with a `static/` directory for images) and produce a navigable site.

## Scope Boundaries

- No shortcodes or rich components (separate ideation item #3)
- No dark mode or theme customization (separate ideation item #7)
- No `rulebound serve` / watch mode (separate ideation item #4)
- No content migration tooling (one-time script, not a feature)
- No cross-linking system between pages and rules (use Markdown links)
- No GitHub edit links (separate ideation item #5)
- No multi-package merge builds

## Key Decisions

- **Replace guidelines**: the pages system replaces the existing guidelines implementation. This is a migration of working code, not prevention of throwaway code. Reusable patterns (`parseFrontmatter()`, layout routing, config patterns) are preserved.
- **Hybrid discovery**: auto-discover with optional config/meta overrides. Zero ceremony default.
- **Root directory `pages/`**: clear intent, no Hugo reserved name collision.
- **`_meta.yml` not `_meta.js`**: YAML is the project's existing config language (rulebound.yml, Vale YAML). No JS parsing needed.
- **All sidebar ordering in `_meta.yml`**: the `rules` keyword and `rules_title` field keep all sidebar ordering and naming in one surface. `rulebound.yml` does not control sidebar layout.
- **6-level nesting cap**: matches the style guide's known maximum (5-6 levels). Deeper structures are warned and flattened. Avoids shipping untested UX.
- **Section headers always linkable**: auto-generated `_index.md` renders a child listing. Simpler than trying to make Hugo pages non-linkable.
- **Tree data structure**: the parser returns a `SectionTree` that naturally represents the nested hierarchy. The generator serializes it to `data/navigation.json` for Hugo template consumption.
- **Data-driven sidebar**: when pages content exists, the sidebar reads `data/navigation.json` instead of iterating `site.Taxonomies.categories`. When no pages exist, the current taxonomy-driven sidebar is preserved unchanged.
- **Static assets included**: ~15 lines of Go, directly supports the success criterion of dropping in existing content.

## Dependencies / Assumptions

- Hugo's section-based content organization (`_index.md` for branch bundles) is leveraged. Rulebound generates the necessary Hugo structure internally.
- The embedded Hugo theme's existing `<details>/<summary>` CSS provides the foundation for sidebar collapse. CSS must be extended with depth-aware indentation for 6 levels.
- `go.yaml.in/yaml/v3` is used for parsing `_meta.yml` (consistent with existing YAML parsing).
- Kebab-to-Title-Case conversion uses Hugo's existing `humanize | title` pipeline in templates, matching the current sidebar pattern (`{{ $catName | humanize | title }}`).

## Outstanding Questions

### Deferred to Planning

- [Affects R15][Technical] What fields and structure should the auto-generated `_index.md` contain? Minimum: `title` from `_meta.yml` or directory name, `type: page`. Should it include `layout: list` or rely on Hugo's default section behavior?
- [Affects R24][Technical] What is the exact schema for `data/navigation.json`? The tree must encode: section title, order, collapse state, hidden items, active page highlighting, and the rules section position. Planning should define the JSON structure.
- [Affects R30][Technical] Should `SectionTree` be defined in `internal/parser/types.go` (alongside `ValeRule`) or in a new `internal/pages/` package? The parser currently only handles Vale YAML; pages parsing is a different concern.
- [Affects R27][Technical] During the transition period where both `guidelines/` and `pages/` coexist: what is the exact behavior? If both directories exist, does `pages/` win entirely? Does the `guidelines` config apply to `pages/guidelines/` if present?

## Next Steps

-> `/ce:plan` for structured implementation planning
