---
date: 2026-03-26
topic: pagefind-search-polish
---

# Pagefind Search Polish

## Problem Frame

Writers searching the generated style guide site for a specific rule (e.g., "Oxford Comma") may get irrelevant results because Pagefind indexes all content equally. Search results also lack structured metadata -- a writer can't see at a glance whether a result is an error-level substitution rule or a suggestion-level existence rule. The `data-pagefind-filter` attributes for severity, type, and category already exist in the markup but are not exposed in the search UI.

## Requirements

- R1. Rule page titles (`<h1>`) receive a higher Pagefind weight than body content so exact rule-name searches rank correctly.
- R2. Rule message text (`<p class="rule-message">`) receives a moderate Pagefind weight boost -- higher than body text but lower than titles.
- R3. Search results display structured metadata (severity, rule type) alongside the page title so writers can identify result relevance without clicking through.
- R4. The Pagefind search UI exposes filter controls for the existing `severity`, `type`, and `category` facets, allowing writers to narrow results (e.g., "show only errors").
- R5. Guideline pages (when the guidelines content type ships) should follow the same weight-boosting pattern: guideline title boosted, section title at moderate weight.

## Success Criteria

- Searching for an exact rule name returns that rule's page as the top result.
- Search results show severity and type metadata without requiring a click.
- Writers can filter search results by severity, type, or category.

## Scope Boundaries

- No changes to the Go build pipeline or parser -- this is purely Hugo template and Pagefind configuration.
- No custom JavaScript beyond Pagefind's built-in UI configuration options.
- R5 (guideline pages) is deferred until the guidelines content type ships -- captured here for consistency but not implemented now.

## Key Decisions

- **Weight values**: Defer exact numeric values to planning (Pagefind uses relative weights; the planner should test with representative data to tune).
- **Filter UI via Pagefind config**: Use Pagefind UI's built-in `filterOptions` rather than building custom filter controls.

## Outstanding Questions

### Deferred to Planning

- [Affects R1, R2][Needs research] What Pagefind weight values produce the best ranking for typical Vale packages (10-200 rules)? Test with the Microsoft testdata package.
- [Affects R4][Needs research] Does Pagefind UI's `filterOptions` configuration support multi-select filters, or are they single-select? Verify against current Pagefind version.
- [Affects R3][Technical] Which `data-pagefind-meta` attributes should be added to surface severity and type in result snippets? Confirm Pagefind's meta rendering behavior.

## Next Steps

All questions are deferred to planning. Ready for `/ce:plan`.
