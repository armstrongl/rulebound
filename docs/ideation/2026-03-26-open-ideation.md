---
date: 2026-03-26
topic: open-ideation
focus: open-ended improvement opportunities
---

# Ideation: Rulebound Open Improvement Opportunities

## Codebase Context

**Project**: Rulebound -- Go CLI tool that generates static style guide websites from Vale rule packages.
**Stack**: Go 1.22+, Cobra CLI, Hugo (embedded theme via go:embed), Pagefind search.
**Pipeline**: parse YAML rules -> generate Hugo content -> scaffold temp Hugo project -> run Hugo -> index with Pagefind.
**Current state**: Core 5-phase pipeline complete. Guidelines content type designed (spec approved) but not yet implemented.

**Known pain points**: No watch/serve mode, no CI integration story, no machine-readable output, Hugo/Pagefind must be pre-installed, theme is opaque (no customization hooks), no cross-linking between content types.

**Past learnings**: Hugo type-based layout routing is the established pattern for new content types. `ParseResult` struct aggregates Rules, Guidelines, Warnings -- composable by design. `*bool` optional config pattern used for backward compat. Synthetic weight offsets (-10000) needed for deterministic ordering.

## Ranked Ideas

### 1. `rulebound lint` -- Fast Parse-Only Quality Check

**Description:** A new subcommand that runs only `parser.ParsePackage()` -- no Hugo, no Pagefind -- and reports all warnings, malformed rules, missing fields, and orphaned companion docs. Exits non-zero on issues. Supports `--strict` (already a pattern on `build`).

**Rationale:** Reuses 100% of existing parser code. Zero new infrastructure. Enables pre-commit hooks and CI gates without requiring Hugo to be installed. Every team using rulebound in automation wants this.

**Downsides:** Thin scope -- some might argue it is just `rulebound build` without the build step. Needs clear differentiation from `--strict` on build.

**Confidence:** 90%
**Complexity:** Low
**Status:** Unexplored

### 2. `--format json` -- Structured Build Output

**Description:** Add a `--format json` flag to `build` (and `lint`) that serializes build results as JSON to stdout: rule count, warning list, coverage stats, output path, timing. The data already exists in `ParseResult` and the build summary block in `cmd/build.go`.

**Rationale:** One `json.Marshal` call. Makes rulebound composable in CI pipelines, dashboards, and scripted workflows. Coverage reporting becomes a free addition -- include `companionDocCoverage` as a field in the JSON.

**Downsides:** Marginal for users who only run builds manually. Needs schema stability commitment once shipped.

**Confidence:** 85%
**Complexity:** Low
**Status:** Unexplored

### 3. `rulebound serve` -- Watch Mode with Live Reload

**Description:** A new subcommand that scaffolds the Hugo project into a persistent temp dir, starts `hugo server --source <tempDir> --watch`, and watches the Vale package directory for changes. On file change, re-runs parse + generate for affected files and Hugo's live reload handles the browser refresh.

**Rationale:** Hugo already has live reload built in -- rulebound just needs to expose it. The scaffolded temp dir is already a valid Hugo project. Drops the authoring feedback loop from ~10s full-rebuild to sub-second. Critical for companion doc and guideline writing.

**Downsides:** Requires changing the scaffold lifecycle (temp dir must persist, not `defer os.RemoveAll`). Signal handling needs adjustment. `fsnotify` would be a new dependency.

**Confidence:** 75%
**Complexity:** Medium
**Status:** Unexplored

### 4. Regex Explainer for Non-Technical Writers

**Description:** For existence, substitution, and conditional rules containing regex patterns, generate a human-readable gloss of common Vale regex idioms: `\b` -> "word boundary", `(?i)` -> "case-insensitive", `a|b` -> "either A or B". Patterns outside the handled subset get a graceful fallback.

**Rationale:** The most user-empathetic idea. Writers -- the primary audience -- see `(?i)\b(utilize|leverage)\b` and get nothing useful. A bounded translator covering the 10-15 most common Vale regex idioms would cover 80%+ of real-world rules. `swapSampler` in `frontmatter.go` already detects regex metacharacters.

**Downsides:** Not a full regex parser -- complex patterns hit the fallback. Requires testing against real-world Vale packages. Bounded but non-trivial implementation.

**Confidence:** 65%
**Complexity:** Medium
**Status:** Unexplored

### 5. Multi-Package Merge Build

**Description:** Allow `rulebound build` to accept multiple package paths and produce a single unified site. Rules namespaced by package directory name (already how `rule.Category` works). Sidebar gains package-level grouping. Pagefind indexes across all packages for cross-package search.

**Rationale:** Many teams layer multiple Vale packages (`Microsoft` + internal). A unified site with cross-package search is a fundamental capability upgrade. `ParseResult` is composable -- merging multiple results requires concatenating `.Rules` and `.Guidelines` slices.

**Downsides:** Significant multi-layer change: config schema, parser loop, theme sidebar, name collision handling. Highest complexity on this list. Single-package model should stabilize first.

**Confidence:** 60%
**Complexity:** High
**Status:** Unexplored

### 6. Rule-to-Guideline Cross-Linking

**Description:** Introduce `related_guidelines` frontmatter in companion `.md` files and `related_rules` in guideline frontmatter. The generator resolves references at build time, emitting Hugo `relref`-style links. The theme renders "Related Guidelines" / "Related Rules" panels on the respective pages.

**Rationale:** The spec for guidelines explicitly deferred cross-linking. A style guide where rules and rationale are disconnected is only half useful. Bridges two content types into a coherent whole.

**Downsides:** Timing-dependent -- guidelines are not shipped yet. Should ship after guidelines are stable and validated.

**Confidence:** 55%
**Complexity:** Medium
**Status:** Unexplored

### 7. Severity Rationale Line

**Description:** Add a one-sentence action statement below each severity badge: error = "Fix before publishing", warning = "Review and fix unless documented exception", suggestion = "Optional improvement." Three static strings in a Hugo partial.

**Rationale:** Writers don't know whether "warning" means "blocker" or "nice-to-have." Closes the gap between lint output and editorial decision-making with zero per-rule authoring effort.

**Downsides:** Opinionated defaults may not match all teams' severity policies.

**Confidence:** 92%
**Complexity:** Low (zero Go changes -- 3 strings in 1 Hugo partial)
**Status:** Unexplored

### 8. Hugo Aliases for Raw Rule Names

**Description:** Add Hugo `aliases` to each rule's frontmatter so `/rules/Microsoft.Avoid/` redirects to the canonical `/rules/avoid/`. Hugo generates static redirect pages at build time with zero runtime cost.

**Rationale:** Vale output shows `Microsoft.Avoid` but the site URL is `/rules/avoid/`. Writers who guess the URL from the Vale flag fail. Aliases bridge this gap declaratively with one frontmatter field.

**Downsides:** Requires surfacing the package prefix in the generator. Alias pages add a small amount of build output.

**Confidence:** 88%
**Complexity:** Low (1 frontmatter field addition)
**Status:** Unexplored

### 9. Pagefind Weight Boosting

**Description:** Add `data-pagefind-weight="10"` to the rule title `<h1>` and `data-pagefind-weight="5"` to the message text in `single.html`. Two HTML attribute additions.

**Rationale:** Pagefind currently indexes all content equally. Searching "Oxford Comma" may return pages that mention the phrase in body text ahead of the actual rule page. Weight boosting is a native Pagefind feature requiring zero pipeline changes.

**Downsides:** None meaningful. Pure improvement with zero risk.

**Confidence:** 95%
**Complexity:** Low (2 HTML attributes in 1 template file)
**Status:** Explored (brainstorm 2026-03-26)

### 10. Auto-Generated "Fix This" Block

**Description:** For substitution rules, auto-generate a prominent Before/After callout at the top of the rule page using existing `swapSampler` logic. For other rule types, render the message with annotated slots. Appears above companion prose so writers see the fix first.

**Rationale:** Writers arrive with one question: "What should I change?" Today they must read through the full page. A "Fix This" block answers in two lines. `swapSampler` already extracts illustrative examples -- this promotes them to a structured, styled block.

**Downsides:** Not all rule types have clean auto-generatable fixes (existence rules just flag words). Needs graceful degradation for non-substitution rules.

**Confidence:** 72%
**Complexity:** Medium (generator change + new template partial)
**Status:** Unexplored

### 11. Annotated Message Template

**Description:** Instead of replacing all `%s` with identical `[matched text]` labels, annotate each slot with its semantic role based on rule type. Substitution `%s -> %s` becomes `[word you used] -> [preferred word]`. Deterministic mapping: rule type + slot position = label.

**Rationale:** Current display shows "'[matched text]' is not preferred. Use '[matched text]' instead." -- both slots identical, actively misleading. Type-keyed labels fix this with a small Go function.

**Downsides:** Positional format verbs (`%[1]s`, `%[2]s`) need parsing. Edge cases for rules with 3+ slots.

**Confidence:** 70%
**Complexity:** Medium (new Go function in `frontmatter.go` + template update)
**Status:** Unexplored

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| 1 | `rulebound init` | Config is optional with working defaults; friction is editorial judgment, not file creation |
| 2 | Embedded Hugo/Pagefind binary | Maintenance nightmare for single maintainer; `brew install hugo` solves in one line |
| 3 | Build artifact gitignore + CI | Two-line gitignore; CI workflow is a README snippet, not a tool feature |
| 4 | Theme override layer | Hugo multi-theme already handles this; document the capability instead |
| 5 | Incremental build cache | Premature optimization -- builds run in seconds at realistic scale |
| 6 | Prose examples as structured content | Companion `.md` already carries prose; structured examples create adoption hurdles |
| 7 | Coverage report (standalone) | Trivially absorbed into `--format json` output as a field |
| 8 | `rulebound ci` generator | Generates a static YAML template; better served by documentation |
| 9 | `rulebound new` scaffolding | Low friction solved -- rule authors already know Vale YAML |
| 10 | Embeddable Go library API | Zero downstream consumers; premature API contract |
| 11 | AI-assisted companion generation | External API dependency inappropriate for deterministic CLI tool |
| 12 | Glossary content type | Third content type before second is battle-tested; scope creep |
| 13 | Package diff / migration | Good idea but wrong priority; partially served by `git diff` |
| 14 | Embeddable fragment mode | Already works via `baseURL` config; documentation gap, not feature gap |
| 15 | Cross-cut: Quality Intelligence | Dissolves into #7 and #14, which are stronger individually |
| 16 | Cross-cut: Content Graph | Bundles three WEAK/CUT ideas; multiplies risk without reducing it |
| 17 | Cross-cut: Org-Scale Management | Bundles multi-package with weaker ideas; delays strongest one |
| 18 | Rule Type Tooltip | Incremental polish -- taxonomy link already leads to grouped page |
| 19 | Swap Table "Why" Column | Requires new companion format with high adoption barrier |
| 20 | Cross-Rule Co-Firing Graph | High complexity for uncertain value; related rules via categories already exist |
| 21 | Hover-Preview Rule Cards | CSS/JS complexity for marginal gain |
| 22 | Browse by Scope taxonomy | Mechanical extension -- useful but not priority over above |
| 23 | "Start Here" Config Path | Moderate value but adds config complexity |
| 24 | Pagefind Quick-Lookup Endpoint | Niche -- few writers query via curl |
| 25 | Rule URL Registry (JSON) | Superseded by Hugo Aliases (#8) which solve the same problem more directly |
| 26 | Vale Config Snippet | Most writers don't touch .vale.ini -- niche audience |
| 27 | `rulebound inspect` CLI | High cost for overlap with aliases + URL registry |
| 28 | Errors-First Landing | Taxonomy pages already exist; promotion is a template choice, not a feature |

## Session Log

- 2026-03-26: Initial ideation -- 48 raw ideas generated (6 agents x ~8 each), deduped to 23 unique + 3 cross-cutting, 6 survived adversarial filtering
- 2026-03-26: Refinement round -- content quality focus. 18 new ideas generated (3 agents x 6 each), 5 new survivors added (items 7-11)
- 2026-03-26: Brainstorm initiated for #9 (Pagefind Weight Boosting)
