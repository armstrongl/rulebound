---
title: "Hugo Sidebar Rendering: define/template in partials, JSON float64 vs int64, CSS compound selectors"
date: 2026-03-27
category: ui-bugs
module: hugo-theme
problem_type: ui_bug
component: tooling
symptoms:
  - "Rules section completely absent from rendered sidebar HTML"
  - "Hugo eq comparison always returns false for JSON-sourced position values"
  - "CSS depth indentation selectors never match, all items render at same indent"
  - "No Hugo build error or warning produced for any of the three bugs"
root_cause: wrong_api
resolution_type: code_fix
severity: medium
tags:
  - hugo
  - template
  - partial
  - define
  - float64
  - int64
  - css-selector
  - sidebar
  - navigation-json
---

# Hugo Sidebar Rendering: Three Bugs Fixed

## Problem

Three independent bugs prevented the data-driven sidebar from rendering correctly in rulebound's embedded Hugo theme. The rules section was completely absent, position interleaving never matched, and CSS depth indentation had no effect. All three bugs were silent — Hugo produced no errors.

## Symptoms

- Rules section HTML was completely missing from the rendered output, despite `navigation.json` containing correct data (7 categories, 14 rules, position: 2)
- Debug attributes confirmed data reached the template (`data-has-nav="yes"`, correct category counts), but the `{{ template "rules-section" }}` call produced no output
- Position comparison `{{ if eq $rulesPos (add $i 1) }}` always evaluated to `false` — debug revealed `data-postype="float64"` vs `data-checktype="int64"`
- Depth-2 sidebar items (e.g., "Email Template") rendered at the same indentation as depth-1 items
- No Hugo build error, warning, or stderr output for any of the three bugs

## What Didn't Work

- **Verifying navigation.json** — data was correct; the bug was in template resolution, not data generation
- **Adding HTML comments for debug** — Hugo strips HTML comments from output, so debug comments were invisible. Switching to visible `<div>` elements with `data-*` attributes worked
- **Checking CSS class application** — depth classes were applied correctly in the HTML; the selectors themselves were wrong
- **Assuming `{{ define }}` / `{{ template }}` works in Hugo partials** — this is a fundamental Hugo limitation that is not well-documented and produces no error

## Solution

### Bug 1: Extract inline define to a separate partial

Hugo's `{{ define }}` / `{{ template }}` mechanism doesn't work inside partial files. Partials compile in their own template namespace.

**Before** (broken — inside `sidebar.html`):
```html
{{- template "rules-section" (dict "rulesSection" $rulesSection "currentURL" $currentURL) -}}

{{- define "rules-section" -}}
  {{- $rulesSection := .rulesSection -}}
  {{- with $rulesSection.categories -}}
    <details class="sidebar-section sidebar-depth-1" open>
      ...
    </details>
  {{- end -}}
{{- end -}}
```

**After** (fixed — separate `sidebar-rules-section.html`):
```html
{{/* sidebar.html */}}
{{- partial "sidebar-rules-section.html" (dict "rulesSection" $rulesSection "currentURL" $currentURL) -}}

{{/* sidebar-rules-section.html — new file */}}
{{- $rulesSection := .rulesSection -}}
{{- with $rulesSection.categories -}}
  <details class="sidebar-section sidebar-depth-1" open>
    ...
  </details>
{{- end -}}
```

### Bug 2: Cast JSON float64 to int before eq comparison

JSON numbers are `float64` in Go/Hugo. Hugo's `add` returns `int64`. Hugo's `eq` is strict on types.

**Before** (broken):
```html
{{- $rulesPos := .rules_section.position -}}
{{- if eq $rulesPos (add $i 1) -}}
```

**After** (fixed):
```html
{{- $rulesPos := int .rules_section.position -}}
{{- if eq $rulesPos (add $i 1) -}}
```

### Bug 3: Use compound CSS selectors, not descendant combinators

The `<summary>` element IS the `.sidebar-section-title`, not a parent of it.

**Before** (broken — descendant combinator looks for child element):
```css
.sidebar-depth-2 > summary .sidebar-section-title { padding-left: 1.75rem; }
```

**After** (fixed — compound selector matches the element itself):
```css
.sidebar-depth-2 > summary.sidebar-section-title { padding-left: 1.75rem; }
```

## Why This Works

**Bug 1**: Hugo partials are compiled in isolated template namespaces. A `{{ define "name" }}` block inside a partial never registers in the global template namespace, so `{{ template "name" }}` cannot find it and silently produces nothing. Using `{{ partial "file.html" }}` is Hugo's native mechanism for partial inclusion and resolves correctly.

**Bug 2**: Go's `encoding/json` unmarshals all JSON numbers as `float64`. Hugo's `add` function returns `int64`. Hugo's `eq` function performs strict type comparison — `eq float64(2) int64(2)` returns `false` because the types differ. The `int` cast normalizes both sides to the same type. (auto memory [claude]: Hugo data access uses `index hugo.Data "site"` pattern which surfaces these float64 values)

**Bug 3**: `summary .class` (with space) is a CSS descendant combinator — it selects `.class` elements nested inside `<summary>`. `summary.class` (no space) is a compound selector — it selects `<summary>` elements that also have `.class`. Since `<summary class="sidebar-section-title">` is itself the target, only the compound form matches.

## Prevention

- **Never use `{{ define }}` / `{{ template }}` in Hugo partials.** Reserve `{{ define }}` for base template blocks (`baseof.html`). Always use `{{ partial "name.html" }}` with separate files for sub-template extraction in partials.
- **Always cast JSON-sourced numbers with `int` in Hugo templates** before comparing with loop counters or `add` results. Document this as a project convention.
- **When an HTML element carries a class directly, use compound selectors** (`element.class`), not descendant selectors (`element .class`). This is especially common with `<summary>`, `<button>`, and `<a>` elements.
- **Use visible debug elements, not HTML comments**, when debugging Hugo templates — Hugo may strip comments. `<div data-debug="value">` is reliable.
- **Add integration tests that assert expected HTML landmarks** in the rendered output (e.g., a rules section wrapper, correct DOM position, computed indentation) to catch silent template and CSS failures.

## Related Issues

- Files modified: `internal/hugo/theme/layouts/partials/sidebar.html`, `sidebar-rules-section.html` (new), `static/css/style.css`, `theme_test.go`
- Feature branch: `feat/nested-content-sections`
- PR: https://github.com/armstrongl/rulebound/pull/1
