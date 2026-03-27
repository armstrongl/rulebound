# rulebound

[![CI](https://github.com/armstrongl/rulebound/actions/workflows/ci.yml/badge.svg)](https://github.com/armstrongl/rulebound/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/armstrongl/rulebound)](https://goreportcard.com/report/github.com/armstrongl/rulebound)
[![Go Reference](https://pkg.go.dev/badge/github.com/armstrongl/rulebound.svg)](https://pkg.go.dev/github.com/armstrongl/rulebound)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/armstrongl/rulebound)](https://github.com/armstrongl/rulebound/releases/latest)

Generate static style guide websites from Vale linting packages.

<img width="1185" height="973" alt="CleanShot 2026-03-26 at 11 08 53" src="https://github.com/user-attachments/assets/c3df108c-acb3-455f-a584-9b177f9c7b73" />

## What it does

rulebound takes a directory of Vale YAML rule definitions, parses them, and produces a static website documenting every rule. The site includes taxonomy pages organized by category, rule type, and severity level, along with responsive design, sidebar navigation, and optional Pagefind client-side search.

Packages can also include hand-authored content alongside the auto-generated rules. A `pages/` directory supports arbitrarily nested Markdown sections for editorial guidance, formatting conventions, resources, and more. A simpler flat `guidelines/` directory is also supported for packages that only need a single section of editorial guidelines.

## Quick start

Generate a style guide site from a Vale package:

```sh
rulebound build ./my-vale-package --output ./public/
```

## Requirements

rulebound requires Hugo to build sites. Pagefind is optional and adds client-side search.

**Hugo >= 0.128.0** (extended edition recommended):

```sh
# macOS
brew install hugo

# Linux (Debian/Ubuntu)
sudo apt install hugo

# Windows
winget install Hugo.Hugo.Extended
```

**Pagefind** (optional -- if absent, sites build without search):

```sh
npm install -g pagefind
```

**Go 1.22+** is only needed if you install from source.

## Installation

### Homebrew (macOS and Linux)

```sh
brew install armstrongl/tap/rulebound
```

### Pre-built binaries

Download a binary for your platform from the [latest release](https://github.com/armstrongl/rulebound/releases/latest), extract it, and add it to your `PATH`.

### Go install

```sh
go install github.com/armstrongl/rulebound@latest
```

### Build from source

```sh
git clone https://github.com/armstrongl/rulebound.git
cd rulebound
make build    # compile to ./rulebound
make install  # install to $GOPATH/bin
```

## Usage

### `rulebound build <package-path>`

Build a static style guide site from a Vale rule package.

| Flag | Short | Default | Description |
| --- | --- | --- | --- |
| `--output` | `-o` | `./public/` | Output directory for the generated site |
| `--config` | `-c` | auto-detect | Path to `rulebound.yml` configuration file |
| `--hugo` |  | auto-detect | Path to Hugo binary |
| `--strict` |  | `false` | Treat parse warnings as errors |
| `--verbose` | `-v` | `false` | Print verbose output |

### `rulebound --version`

Print the current version and exit.

## Configuration

Place a `rulebound.yml` file in the root of your Vale package directory, or pass one with `--config`. The configuration file is optional. rulebound applies defaults when it is absent.

```yaml
title: My Style Guide
description: Documentation for our Vale linting rules
baseURL: https://styleguide.example.com/
categories:
  punctuation:
    - Punctuation.Comma
    - Punctuation.Period
  casing:
    - Casing.HeadingTitle
pages:
  enabled: true
guidelines:
  section_title: Editorial Guidelines
  order:
    - voice-and-tone
    - inclusive-language
  exclude:
    - draft-notes
```

The following fields are available:

| Field | Default | Description |
| --- | --- | --- |
| `title` | Package directory name | Display name of the style guide |
| `description` | (empty) | Short description displayed on the site |
| `baseURL` | `/` | Base URL for the generated Hugo site |
| `categories` | Group by rule type | Map of category names to lists of rule identifiers. A rule may appear in multiple categories. |
| `pages.enabled` | `true` | Set to `false` to suppress content page generation even when a `pages/` directory exists |
| `guidelines.section_title` | `Guidelines` | Sidebar heading for the guidelines section |
| `guidelines.order` | Alphabetical | Page ordering by filename stem. Items listed in `order` take precedence over frontmatter `weight` values. |
| `guidelines.exclude` | (none) | Filename stems to skip. Takes precedence over `order`. |
| `guidelines.enabled` | `true` | Set to `false` to suppress guideline generation even when files exist |

## How it works

rulebound builds the site in six stages:

1. **Parse** -- Reads all Vale YAML rule files in the package directory (supports all 11 extension types). Also reads content pages from a `pages/` directory and editorial guidelines from a `guidelines/` subdirectory.
2. **Companion docs** -- Reads companion `.md` files alongside each rule for custom documentation content.
3. **Generate** -- Produces Hugo content files with frontmatter and taxonomy terms for each rule, plus content pages and guideline pages with their own layouts.
4. **Scaffold** -- Creates a Hugo project in a temporary directory with an embedded theme.
5. **Build** -- Runs Hugo to compile the static site into the output directory.
6. **Search index** -- If Pagefind is installed, runs it to generate a client-side search index.

## Content pages

To add hand-authored content sections alongside your Vale rules, place Markdown files in a `pages/` directory within your package. The `pages/` directory supports nested subdirectories up to 6 levels deep.

```
my-vale-package/
├── Avoid.yml
├── Terms.yml
├── rulebound.yml
└── pages/
    ├── _meta.yml
    ├── _index.md
    ├── formatting/
    │   ├── _meta.yml
    │   ├── headings.md
    │   └── lists.md
    ├── language-and-grammar/
    │   ├── _index.md
    │   ├── active-voice.md
    │   └── pronouns.md
    └── resources/
        ├── glossary.md
        └── templates/
            └── email-template.md
```

### Page files

Each `.md` file uses YAML frontmatter:

```markdown
---
title: "Headings"
description: "How to write effective headings"
---

Use sentence case for all headings.
```

| Field | Required | Description |
| --- | --- | --- |
| `title` | No | Page title. If omitted, rulebound derives it from the filename (for example, `active-voice.md` becomes "Active Voice"). |
| `description` | No | Short summary for the page |

### Section metadata (`_meta.yml`)

Place a `_meta.yml` file in any directory to control sidebar navigation:

```yaml
title: "Language and Grammar"
order:
  - active-voice
  - pronouns
  - rules
collapsed: false
hidden:
  - draft-notes
rules_title: "Linting Rules"
```

| Field | Default | Description |
| --- | --- | --- |
| `title` | Derived from directory name | Display name for the section in the sidebar |
| `order` | Alphabetical | List of filename stems defining sidebar sequence. Unlisted pages sort alphabetically after listed ones. |
| `collapsed` | `false` | Whether the section starts collapsed in the sidebar |
| `hidden` | (none) | Filename stems to exclude from sidebar navigation and search indexing |
| `rules_title` | `Rules` | Display name for the auto-generated rules section (top level only) |

The `order` list supports a reserved `rules` keyword at the top level. Including `rules` in the list controls where the auto-generated Vale rules section appears relative to your content sections.

### Hub pages (`_index.md`)

Place an `_index.md` file in any directory to add an introductory hub page for that section. Hub pages appear at the section root URL (for example, `/pages/language-and-grammar/`).

### Nesting

Directories nest up to 6 levels deep. Directories deeper than 6 levels are still parsed but flattened to level 6 with a warning. If a `pages/rules/` directory exists and `rules` also appears in the `_meta.yml` `order` list, the directory takes precedence and rulebound emits a collision warning.

## Editorial guidelines

For packages that only need a flat set of editorial guidelines, rulebound also supports a simpler `guidelines/` subdirectory. If your package has a `pages/` directory, consider migrating guidelines into `pages/guidelines/` instead.

```
my-vale-package/
├── Avoid.yml
├── Terms.yml
├── rulebound.yml
└── guidelines/
    ├── voice-and-tone.md
    └── inclusive-language.md
```

Each guideline file uses YAML frontmatter:

```markdown
---
title: "Voice and Tone"
description: "How to write in our company voice"
weight: 10
---

Write with clarity and confidence. Avoid jargon.
```

| Field | Required | Description |
| --- | --- | --- |
| `title` | Yes | Page title displayed in sidebar and heading |
| `description` | No | Short summary shown in the guidelines index |
| `weight` | No | Sort order (lower values appear first, default: 0) |

Guidelines appear in a dedicated sidebar section and have their own index page at `/guidelines/`. rulebound skips files without a `title` in frontmatter, files with malformed YAML, and files with non-`.md` extensions. rulebound ignores subdirectories inside `guidelines/`.

## Companion documentation

Any Vale rule can have a companion Markdown file with the same base name. When present, rulebound uses the companion file's content as the rule's documentation page body instead of the auto-generated description.

```
my-vale-package/
├── Avoid.yml
├── Avoid.md         <-- companion doc for the Avoid rule
├── Terms.yml
└── Terms.md         <-- companion doc for the Terms rule
```

Companion files use standard Markdown with no required frontmatter:

```markdown
## Why we avoid these words

These terms are exclusionary or unclear. Use the suggested alternatives instead.

**Example:** Instead of "whitelist," write "allowlist."
```

Rules without companion files display an auto-generated description based on the rule's YAML fields (message, severity, type, and sample patterns).

## Supported Vale rule types

rulebound parses all 11 Vale extension types:

- `existence`
- `substitution`
- `occurrence`
- `repetition`
- `consistency`
- `conditional`
- `capitalization`
- `metric`
- `script`
- `sequence`
- `spelling`

## Exit codes

rulebound returns the following exit codes:

| Code | Meaning                           |
| ---- | --------------------------------- |
| 0    | Success                           |
| 1    | General error                     |
| 2    | Configuration error               |
| 3    | Hugo not found or version too old |
| 4    | Hugo build failure                |

## Project structure

The repository is organized as follows:

```
cmd/           CLI commands (root, build)
internal/
  config/      rulebound.yml parsing
  parser/      Vale rules, guidelines, pages, and companion doc parser
  generator/   Hugo content and navigation generation
  hugo/        Hugo scaffolding, build, embedded theme
```

## Development

Run tests and build the binary with the following commands:

```sh
make test       # run all tests
make build      # compile binary
go test ./...   # run tests directly
```

## License

Refer to [LICENSE](LICENSE) for details.
